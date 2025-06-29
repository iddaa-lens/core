package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/iddaa-lens/core/pkg/apifootball"
	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models"
	"github.com/iddaa-lens/core/pkg/services"
)

// APIFootballLeagueMatchingJob handles league matching with API-Football
type APIFootballLeagueMatchingJob struct {
	db        *generated.Queries
	matcher   *services.TeamLeagueMatcher
	apiclient *apifootball.Client
}

// NewAPIFootballLeagueMatchingJob creates a new API-Football league matching job
func NewAPIFootballLeagueMatchingJob(db *generated.Queries) *APIFootballLeagueMatchingJob {
	apiKey := os.Getenv("API_FOOTBALL_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")

	// Create API-Football client
	apiConfig := apifootball.DefaultConfig(apiKey)
	apiclient := apifootball.NewClient(apiConfig)

	return &APIFootballLeagueMatchingJob{
		db:        db,
		matcher:   services.NewTeamLeagueMatcher(openaiKey),
		apiclient: apiclient,
	}
}

// Name returns the job name
func (j *APIFootballLeagueMatchingJob) Name() string {
	return "api_football_league_matching"
}

// Schedule returns the cron schedule - run weekly on Tuesdays at 3 AM
func (j *APIFootballLeagueMatchingJob) Schedule() string {
	return "0 3 * * 2"
}

// Execute runs the league matching process
func (j *APIFootballLeagueMatchingJob) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx, "api-football-league-matching")
	start := time.Now()

	log.Info().
		Str("action", "sync_start").
		Msg("Starting API-Football league matching job")

	// Check if API client is available
	if !j.apiclient.IsAvailable() {
		log.Warn().
			Str("action", "api_key_missing").
			Msg("API_FOOTBALL_API_KEY not set, skipping league matching")
		return nil
	}

	// Step 1: Get unmapped football leagues (sport_id = 1)
	unmappedLeagues, err := j.getUnmappedFootballLeagues(ctx)
	if err != nil {
		return fmt.Errorf("failed to get unmapped leagues: %w", err)
	}

	if len(unmappedLeagues) == 0 {
		log.Info().
			Str("action", "no_unmapped_leagues").
			Msg("No unmapped football leagues found")

		duration := time.Since(start)
		log.LogJobComplete("api_football_league_matching", duration, 0, 0)
		return nil
	}

	log.Info().
		Str("action", "leagues_found").
		Int("league_count", len(unmappedLeagues)).
		Msg("Found unmapped football leagues to process")

	// Step 2: Get all available leagues from API-Football
	apiLeagues, err := j.fetchAllAPIFootballLeagues(ctx)
	if err != nil {
		// Check if it's a rate limit error
		var rateLimitErr *apifootball.RateLimitError
		if errors.As(err, &rateLimitErr) {
			log.Warn().
				Str("action", "rate_limit_hit").
				Str("error", rateLimitErr.Error()).
				Msg("Football API rate limit exceeded, will retry in next job run")

			// For rate limit errors, exit gracefully without failing the job
			duration := time.Since(start)
			log.LogJobComplete("api_football_league_matching", duration, 0, 0)
			return nil
		}
		return fmt.Errorf("failed to fetch API-Football leagues: %w", err)
	}

	log.Info().
		Str("action", "api_leagues_fetched").
		Int("api_league_count", len(apiLeagues)).
		Msg("Fetched leagues from API-Football")

	// Step 3: Process each unmapped league
	successCount := 0
	errorCount := 0

	for i, league := range unmappedLeagues {
		// Rate limiting between requests
		if i > 0 {
			time.Sleep(200 * time.Millisecond)
		}

		leagueStart := time.Now()
		log.Debug().
			Str("action", "league_processing_start").
			Int("league_id", int(league.ID)).
			Str("league_name", league.Name).
			Str("country", *league.Country).
			Msg("Processing league for matching")

		// Match with API-Football
		match, err := j.matcher.MatchLeagueWithAPI(ctx, league, apiLeagues)
		if err != nil {
			// Check if it's a rate limit error during matching
			var rateLimitErr *apifootball.RateLimitError
			if errors.As(err, &rateLimitErr) {
				log.Warn().
					Str("action", "rate_limit_during_matching").
					Str("error", rateLimitErr.Error()).
					Int("league_id", int(league.ID)).
					Str("league_name", league.Name).
					Msg("Rate limit hit during league matching, stopping processing")

				// Exit the loop gracefully when hitting rate limits
				break
			}

			errorCount++
			log.Error().
				Err(err).
				Str("action", "league_matching_failed").
				Int("league_id", int(league.ID)).
				Str("league_name", league.Name).
				Dur("duration", time.Since(leagueStart)).
				Msg("Failed to match league")
			continue
		}

		if match == nil {
			log.Debug().
				Str("action", "no_match_found").
				Int("league_id", int(league.ID)).
				Str("league_name", league.Name).
				Dur("duration", time.Since(leagueStart)).
				Msg("No suitable match found for league")
			continue
		}

		// Store the mapping
		err = j.storeLeagueMapping(ctx, league, match)
		if err != nil {
			errorCount++
			log.Error().
				Err(err).
				Str("action", "mapping_storage_failed").
				Int("league_id", int(league.ID)).
				Str("league_name", league.Name).
				Str("matched_name", match.Name).
				Float64("confidence", match.Confidence).
				Msg("Failed to store league mapping")
			continue
		}

		successCount++
		log.Info().
			Str("action", "league_matched").
			Int("league_id", int(league.ID)).
			Str("league_name", league.Name).
			Str("matched_name", match.Name).
			Str("country", match.Country).
			Float64("confidence", match.Confidence).
			Str("method", match.Method).
			Dur("duration", time.Since(leagueStart)).
			Msg("League successfully matched and stored")
	}

	duration := time.Since(start)
	log.LogJobComplete("api_football_league_matching", duration, successCount, errorCount)

	if errorCount > 0 {
		log.Warn().
			Int("success_count", successCount).
			Int("error_count", errorCount).
			Int("total_processed", len(unmappedLeagues)).
			Msg("League matching completed with some errors")
	} else {
		log.Info().
			Int("success_count", successCount).
			Int("total_processed", len(unmappedLeagues)).
			Msg("League matching completed successfully")
	}

	return nil
}

// getUnmappedFootballLeagues returns football leagues that don't have API-Football mappings
func (j *APIFootballLeagueMatchingJob) getUnmappedFootballLeagues(ctx context.Context) ([]generated.League, error) {
	return j.db.ListUnmappedFootballLeagues(ctx)
}

// fetchAllAPIFootballLeagues fetches all available leagues from API-Football
func (j *APIFootballLeagueMatchingJob) fetchAllAPIFootballLeagues(ctx context.Context) ([]models.SearchResult, error) {
	// Fetch current active leagues using the new client
	leaguesData, err := j.apiclient.GetCurrentLeagues(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch current leagues: %w", err)
	}

	if len(leaguesData) == 0 {
		return nil, fmt.Errorf("no leagues returned from API")
	}

	// Convert to SearchResult format
	results := make([]models.SearchResult, 0, len(leaguesData))
	for _, item := range leaguesData {
		results = append(results, models.SearchResult{
			ID:      item.League.ID,
			Name:    item.League.Name,
			Country: item.Country.Name,
		})
	}

	return results, nil
}

// storeLeagueMapping stores a league mapping in the database
func (j *APIFootballLeagueMatchingJob) storeLeagueMapping(ctx context.Context, league generated.League, match *services.MatchCandidate) error {
	// Get English translations for storage
	translations, err := j.getLeagueTranslations(ctx, league)
	if err != nil {
		return fmt.Errorf("failed to get translations: %w", err)
	}

	// Create match factors JSON
	matchFactors := map[string]any{
		"method":             match.Method,
		"confidence":         match.Confidence,
		"original_name":      league.Name,
		"translated_name":    translations.LeagueName,
		"matched_name":       match.Name,
		"original_country":   league.Country,
		"translated_country": translations.Country,
		"matched_country":    match.Country,
		"timestamp":          time.Now().UTC(),
	}

	matchFactorsJSON, err := json.Marshal(matchFactors)
	if err != nil {
		return fmt.Errorf("failed to marshal match factors: %w", err)
	}

	// Determine if this mapping needs review
	needsReview := match.Confidence < 0.85

	// Helper function to create a pointer to a bool
	boolPtr := func(b bool) *bool {
		return &b
	}

	// Helper function to create a pointer to a float32
	float32Ptr := func(f float64) *float32 {
		f32 := float32(f)
		return &f32
	}

	// Store the mapping using the enhanced parameters
	_, err = j.db.CreateEnhancedLeagueMapping(ctx, generated.CreateEnhancedLeagueMappingParams{
		InternalLeagueID:     int32(league.ID),
		FootballApiLeagueID:  int32(match.ID),
		Confidence:           float32(match.Confidence),
		MappingMethod:        match.Method,
		TranslatedLeagueName: &translations.LeagueName,
		TranslatedCountry:    &translations.Country,
		OriginalLeagueName:   &league.Name,
		OriginalCountry:      league.Country,
		MatchFactors:         matchFactorsJSON,
		NeedsReview:          &needsReview,
		AiTranslationUsed:    boolPtr(j.matcher.UsesAI()),
		NormalizationApplied: boolPtr(true),
		MatchScore:           float32Ptr(match.Confidence),
	})

	return err
}

// getLeagueTranslations gets English translations for a league
func (j *APIFootballLeagueMatchingJob) getLeagueTranslations(ctx context.Context, league generated.League) (*services.LeagueTranslations, error) {
	// Translate league name
	leagueName, err := j.matcher.GetLeagueNameWithAI(ctx, league.Name, *league.Country)
	if err != nil {
		// Fallback to basic translation
		leagueName = league.Name
	}

	// Translate country
	country := ""
	if league.Country != nil && *league.Country != "" {
		// Use the enhanced translator's country mapping
		enhancedTranslator := services.NewEnhancedTranslator(os.Getenv("OPENAI_API_KEY"))
		country = enhancedTranslator.TranslateCountryName(*league.Country)
	}

	return &services.LeagueTranslations{
		LeagueName: leagueName,
		Country:    country,
		Original:   league,
	}, nil
}

// Timeout returns the job timeout duration
func (j *APIFootballLeagueMatchingJob) Timeout() time.Duration {
	return 30 * time.Minute
}
