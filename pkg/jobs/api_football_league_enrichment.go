package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models"
	"github.com/jackc/pgx/v5/pgtype"
)

// APIFootballLeagueEnrichmentJob enriches league data with detailed API-Football information
type APIFootballLeagueEnrichmentJob struct {
	db     *generated.Queries
	client *http.Client
	apiKey string
}

// NewAPIFootballLeagueEnrichmentJob creates a new league enrichment job
func NewAPIFootballLeagueEnrichmentJob(db *generated.Queries) *APIFootballLeagueEnrichmentJob {
	apiKey := os.Getenv("API_FOOTBALL_API_KEY")

	return &APIFootballLeagueEnrichmentJob{
		db: db,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: apiKey,
	}
}

// Name returns the job name
func (j *APIFootballLeagueEnrichmentJob) Name() string {
	return "api_football_league_enrichment"
}

// Schedule returns the cron schedule - run monthly on the 1st at 2 AM
func (j *APIFootballLeagueEnrichmentJob) Schedule() string {
	return "0 2 1 * *"
}

// Execute runs the league enrichment process
func (j *APIFootballLeagueEnrichmentJob) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx, "api-football-league-enrichment")
	start := time.Now()

	log.Info().
		Str("action", "enrichment_start").
		Msg("Starting API-Football league enrichment job")

	// Check if API key is available
	if j.apiKey == "" {
		log.Warn().
			Str("action", "api_key_missing").
			Msg("API_FOOTBALL_API_KEY not set, skipping league enrichment")
		return nil
	}

	// Get leagues that need enrichment (mapped leagues with stale or missing data)
	leaguesToEnrich, err := j.db.ListLeaguesForAPIEnrichment(ctx, 50) // Process 50 leagues per run
	if err != nil {
		return fmt.Errorf("failed to get leagues for enrichment: %w", err)
	}

	if len(leaguesToEnrich) == 0 {
		log.Info().
			Str("action", "no_leagues_to_enrich").
			Msg("No leagues need enrichment at this time")

		duration := time.Since(start)
		log.LogJobComplete("api_football_league_enrichment", duration, 0, 0)
		return nil
	}

	log.Info().
		Str("action", "leagues_found").
		Int("league_count", len(leaguesToEnrich)).
		Msg("Found leagues to enrich")

	successCount := 0
	errorCount := 0

	for i, league := range leaguesToEnrich {
		// Rate limiting between requests
		if i > 0 {
			time.Sleep(1 * time.Second) // 1 second delay between detailed API calls
		}

		leagueStart := time.Now()
		log.Debug().
			Str("action", "league_enrichment_start").
			Int("league_id", int(league.ID)).
			Str("league_name", league.Name).
			Msg("Starting league enrichment")

		// Get the league mapping to find the API-Football ID
		mapping, err := j.db.GetLeagueMapping(ctx, int32(league.ID))
		if err != nil {
			errorCount++
			log.Error().
				Err(err).
				Str("action", "mapping_fetch_failed").
				Int("league_id", int(league.ID)).
				Msg("Failed to get league mapping")
			continue
		}

		// Fetch detailed data from API-Football
		enrichmentData, err := j.fetchLeagueDetails(ctx, mapping.FootballApiLeagueID)
		if err != nil {
			errorCount++
			log.Error().
				Err(err).
				Str("action", "api_fetch_failed").
				Int("league_id", int(league.ID)).
				Int("api_league_id", int(mapping.FootballApiLeagueID)).
				Dur("duration", time.Since(leagueStart)).
				Msg("Failed to fetch league details from API")
			continue
		}

		// Enrich the league with API data
		err = j.enrichLeague(ctx, league, enrichmentData)
		if err != nil {
			errorCount++
			log.Error().
				Err(err).
				Str("action", "enrichment_failed").
				Int("league_id", int(league.ID)).
				Dur("duration", time.Since(leagueStart)).
				Msg("Failed to enrich league")
			continue
		}

		successCount++
		log.Info().
			Str("action", "league_enriched").
			Int("league_id", int(league.ID)).
			Str("league_name", league.Name).
			Str("api_name", enrichmentData.League.Name).
			Dur("duration", time.Since(leagueStart)).
			Msg("League successfully enriched")
	}

	duration := time.Since(start)
	log.LogJobComplete("api_football_league_enrichment", duration, successCount, errorCount)

	if errorCount > 0 {
		log.Warn().
			Int("success_count", successCount).
			Int("error_count", errorCount).
			Int("total_processed", len(leaguesToEnrich)).
			Msg("League enrichment completed with some errors")
	} else {
		log.Info().
			Int("success_count", successCount).
			Int("total_processed", len(leaguesToEnrich)).
			Msg("League enrichment completed successfully")
	}

	return nil
}

// fetchLeagueDetails fetches detailed league information from API-Football
func (j *APIFootballLeagueEnrichmentJob) fetchLeagueDetails(ctx context.Context, leagueID int32) (*models.APIFootballLeagueDetail, error) {
	url := fmt.Sprintf("https://v3.football.api-sports.io/leagues?id=%d", leagueID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-RapidAPI-Key", j.apiKey)
	req.Header.Set("X-RapidAPI-Host", "v3.football.api-sports.io")

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResponse models.FootballAPILeagueDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API errors
	if apiResponse.HasErrors() {
		errorMessages := apiResponse.GetErrorMessages()
		return nil, fmt.Errorf("API returned errors: %v", errorMessages)
	}

	if apiResponse.Results == 0 || len(apiResponse.Response) == 0 {
		return nil, fmt.Errorf("no league details returned from API")
	}

	return &apiResponse.Response[0], nil
}

// enrichLeague updates the league with API-Football data
func (j *APIFootballLeagueEnrichmentJob) enrichLeague(ctx context.Context, league generated.League, apiData *models.APIFootballLeagueDetail) error {
	// Find the current season (most recent or explicitly marked as current)
	var currentSeason *models.Season
	for i := range apiData.Seasons {
		season := &apiData.Seasons[i]
		if season.Current {
			currentSeason = season
			break
		}
		// If no current season is marked, use the most recent one
		if currentSeason == nil || season.Year > currentSeason.Year {
			currentSeason = season
		}
	}

	// Prepare enrichment parameters
	var currentSeasonYear *int32
	var currentSeasonStart, currentSeasonEnd *time.Time
	var hasStandings, hasFixtures, hasPlayers, hasTopScorers, hasInjuries, hasPredictions, hasOdds bool

	if currentSeason != nil {
		year32 := int32(currentSeason.Year)
		currentSeasonYear = &year32

		if startTime, err := time.Parse("2006-01-02", currentSeason.Start); err == nil {
			currentSeasonStart = &startTime
		}
		if endTime, err := time.Parse("2006-01-02", currentSeason.End); err == nil {
			currentSeasonEnd = &endTime
		}

		// Extract coverage information
		coverage := currentSeason.Coverage
		hasStandings = coverage.Standings
		hasFixtures = coverage.Fixtures.Events
		hasPlayers = coverage.Players
		hasTopScorers = coverage.TopScorers
		hasInjuries = coverage.Injuries
		hasPredictions = coverage.Predictions
		hasOdds = coverage.Odds
	}

	// Store the full API response as JSONB for future reference
	apiDataJSON, err := json.Marshal(apiData)
	if err != nil {
		return fmt.Errorf("failed to marshal API data: %w", err)
	}

	// Update the league with enrichment data
	apiFootballID := int32(apiData.League.ID)

	// Set optional fields conditionally
	params := generated.EnrichLeagueWithAPIFootballParams{
		ID:                league.ID,
		ApiFootballID:     &apiFootballID,
		LeagueType:        &apiData.League.Type,
		HasStandings:      &hasStandings,
		HasFixtures:       &hasFixtures,
		HasPlayers:        &hasPlayers,
		HasTopScorers:     &hasTopScorers,
		HasInjuries:       &hasInjuries,
		HasPredictions:    &hasPredictions,
		HasOdds:           &hasOdds,
		CurrentSeasonYear: currentSeasonYear,
		ApiEnrichmentData: apiDataJSON,
	}

	// Convert time.Time to pgtype.Date for season dates
	if currentSeasonStart != nil {
		params.CurrentSeasonStart = pgtype.Date{Time: *currentSeasonStart, Valid: true}
	}
	if currentSeasonEnd != nil {
		params.CurrentSeasonEnd = pgtype.Date{Time: *currentSeasonEnd, Valid: true}
	}

	if apiData.League.Logo != "" {
		params.LogoUrl = &apiData.League.Logo
	}
	if apiData.Country.Code != "" {
		params.CountryCode = &apiData.Country.Code
	}
	if apiData.Country.Flag != "" {
		params.CountryFlagUrl = &apiData.Country.Flag
	}

	_, err = j.db.EnrichLeagueWithAPIFootball(ctx, params)

	return err
}

// Timeout returns the job timeout duration
func (j *APIFootballLeagueEnrichmentJob) Timeout() time.Duration {
	return 60 * time.Minute
}
