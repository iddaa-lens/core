package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/iddaa-lens/core/pkg/apifootball"
	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models"
	"github.com/iddaa-lens/core/pkg/services"
	"github.com/jackc/pgx/v5/pgtype"
)

// APIFootballTeamMatchingJob handles team matching with API-Football
type APIFootballTeamMatchingJob struct {
	db        *generated.Queries
	matcher   *services.TeamLeagueMatcher
	apiclient *apifootball.Client
}

// NewAPIFootballTeamMatchingJob creates a new API-Football team matching job
func NewAPIFootballTeamMatchingJob(db *generated.Queries) *APIFootballTeamMatchingJob {
	apiKey := os.Getenv("API_FOOTBALL_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")

	// Create API-Football client
	apiConfig := apifootball.DefaultConfig(apiKey)
	apiclient := apifootball.NewClient(apiConfig)

	return &APIFootballTeamMatchingJob{
		db:        db,
		matcher:   services.NewTeamLeagueMatcher(openaiKey),
		apiclient: apiclient,
	}
}

// Name returns the job name
func (j *APIFootballTeamMatchingJob) Name() string {
	return "api_football_team_matching"
}

// Schedule returns the cron schedule - run weekly on Tuesdays at 4 AM (after league matching)
func (j *APIFootballTeamMatchingJob) Schedule() string {
	return "0 4 * * 2"
}

// Execute runs the team matching process
func (j *APIFootballTeamMatchingJob) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx, "api-football-team-matching")
	start := time.Now()

	log.Info().
		Str("action", "sync_start").
		Msg("Starting API-Football team matching job")

	// Check if API client is available
	if !j.apiclient.IsAvailable() {
		log.Warn().
			Str("action", "api_key_missing").
			Msg("API_FOOTBALL_API_KEY not set, skipping team matching")
		return nil
	}

	// Step 1: Get mapped leagues (we need league mappings to get teams)
	mappedLeagues, err := j.getMappedLeagues(ctx)
	if err != nil {
		return fmt.Errorf("failed to get mapped leagues: %w", err)
	}

	if len(mappedLeagues) == 0 {
		log.Info().
			Str("action", "no_mapped_leagues").
			Msg("No mapped leagues found, skipping team matching")

		duration := time.Since(start)
		log.LogJobComplete("api_football_team_matching", duration, 0, 0)
		return nil
	}

	log.Info().
		Str("action", "mapped_leagues_found").
		Int("league_count", len(mappedLeagues)).
		Msg("Found mapped leagues to process teams for")

	// Step 2: Process each mapped league
	totalSuccessCount := 0
	totalErrorCount := 0

	for i, mapping := range mappedLeagues {
		// Rate limiting between league requests
		if i > 0 {
			time.Sleep(500 * time.Millisecond) // Longer delay for team fetching
		}

		leagueStart := time.Now()
		log.Debug().
			Str("action", "league_processing_start").
			Int("internal_league_id", int(mapping.InternalLeagueID)).
			Int("api_league_id", int(mapping.FootballApiLeagueID)).
			Msg("Processing teams for league")

		successCount, errorCount, err := j.processTeamsForLeague(ctx, mapping)
		if err != nil {
			log.Error().
				Err(err).
				Str("action", "league_team_processing_failed").
				Int("internal_league_id", int(mapping.InternalLeagueID)).
				Int("api_league_id", int(mapping.FootballApiLeagueID)).
				Dur("duration", time.Since(leagueStart)).
				Msg("Failed to process teams for league")
			totalErrorCount++
			continue
		}

		totalSuccessCount += successCount
		totalErrorCount += errorCount

		log.Info().
			Str("action", "league_team_processing_complete").
			Int("internal_league_id", int(mapping.InternalLeagueID)).
			Int("api_league_id", int(mapping.FootballApiLeagueID)).
			Int("teams_matched", successCount).
			Int("teams_failed", errorCount).
			Dur("duration", time.Since(leagueStart)).
			Msg("Completed team processing for league")
	}

	duration := time.Since(start)
	log.LogJobComplete("api_football_team_matching", duration, totalSuccessCount, totalErrorCount)

	if totalErrorCount > 0 {
		log.Warn().
			Int("success_count", totalSuccessCount).
			Int("error_count", totalErrorCount).
			Int("leagues_processed", len(mappedLeagues)).
			Msg("Team matching completed with some errors")
	} else {
		log.Info().
			Int("success_count", totalSuccessCount).
			Int("leagues_processed", len(mappedLeagues)).
			Msg("Team matching completed successfully")
	}

	return nil
}

// getMappedLeagues returns all league mappings
func (j *APIFootballTeamMatchingJob) getMappedLeagues(ctx context.Context) ([]generated.LeagueMapping, error) {
	return j.db.ListLeagueMappings(ctx)
}

// processTeamsForLeague processes all teams for a specific league mapping
func (j *APIFootballTeamMatchingJob) processTeamsForLeague(ctx context.Context, mapping generated.LeagueMapping) (int, int, error) {
	// Get internal teams for this league
	internalTeams, err := j.getTeamsForLeague(ctx, mapping.InternalLeagueID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get internal teams: %w", err)
	}

	if len(internalTeams) == 0 {
		return 0, 0, nil // No teams to process
	}

	// Get API-Football teams for this league
	apiTeams, err := j.fetchAPIFootballTeams(ctx, mapping.FootballApiLeagueID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to fetch API-Football teams: %w", err)
	}

	if len(apiTeams) == 0 {
		return 0, 0, nil // No API teams available
	}

	// Match teams
	successCount := 0
	errorCount := 0

	for _, team := range internalTeams {
		// Skip if already mapped
		if existing, err := j.db.GetTeamMapping(ctx, team.ID); err == nil && existing.ID > 0 {
			continue
		}

		// Match with API-Football
		match, err := j.matcher.MatchTeamWithAPI(ctx, team, apiTeams)
		if err != nil {
			errorCount++
			continue
		}

		if match == nil {
			continue // No suitable match found
		}

		// Store the mapping
		err = j.storeTeamMapping(ctx, team, match)
		if err != nil {
			errorCount++
			continue
		}

		successCount++
	}

	return successCount, errorCount, nil
}

// getTeamsForLeague returns all teams for a specific league
func (j *APIFootballTeamMatchingJob) getTeamsForLeague(ctx context.Context, leagueID int32) ([]generated.Team, error) {
	leagueIDPtr := &leagueID
	return j.db.ListTeamsByLeagueID(ctx, leagueIDPtr)
}

// fetchAPIFootballTeams fetches teams for a specific league from API-Football
func (j *APIFootballTeamMatchingJob) fetchAPIFootballTeams(ctx context.Context, leagueID int32) ([]models.SearchResult, error) {
	// Use current year for team fetching
	currentYear := time.Now().Year()

	// Fetch teams using the new client
	teamsData, err := j.apiclient.GetTeamsByLeagueAndSeason(ctx, int(leagueID), currentYear)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch teams for league %d season %d: %w", leagueID, currentYear, err)
	}

	if len(teamsData) == 0 {
		return nil, nil // No teams found (not an error)
	}

	// Convert to SearchResult format
	results := make([]models.SearchResult, 0, len(teamsData))
	for _, item := range teamsData {
		results = append(results, models.SearchResult{
			ID:      item.Team.ID,
			Name:    item.Team.Name,
			Country: item.Team.Country,
		})
	}

	return results, nil
}

// storeTeamMapping stores a team mapping in the database
func (j *APIFootballTeamMatchingJob) storeTeamMapping(ctx context.Context, team generated.Team, match *services.MatchCandidate) error {
	// Get English translations for storage
	translations, err := j.getTeamTranslations(ctx, team)
	if err != nil {
		return fmt.Errorf("failed to get translations: %w", err)
	}

	// Create match factors JSON
	matchFactors := map[string]interface{}{
		"method":             match.Method,
		"confidence":         match.Confidence,
		"original_name":      team.Name,
		"translated_name":    translations.TeamName,
		"matched_name":       match.Name,
		"original_country":   team.Country,
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

	// Helper functions for creating pointers
	strPtr := func(s string) *string {
		if s == "" {
			return nil
		}
		return &s
	}

	boolPtr := func(b bool) *bool {
		return &b
	}

	float32Ptr := func(f float64) *float32 {
		f32 := float32(f)
		return &f32
	}

	// Store the mapping using the enhanced parameters
	_, err = j.db.CreateEnhancedTeamMapping(ctx, generated.CreateEnhancedTeamMappingParams{
		InternalTeamID:       int32(team.ID),
		FootballApiTeamID:    int32(match.ID),
		Confidence:           float32(match.Confidence),
		MappingMethod:        match.Method,
		TranslatedTeamName:   strPtr(translations.TeamName),
		TranslatedCountry:    strPtr(translations.Country),
		TranslatedLeague:     strPtr(translations.League),
		OriginalTeamName:     &team.Name,
		OriginalCountry:      team.Country,
		OriginalLeague:       strPtr(translations.League), // League from team context or empty
		MatchFactors:         matchFactorsJSON,
		NeedsReview:          &needsReview,
		AiTranslationUsed:    boolPtr(j.matcher.UsesAI()),
		NormalizationApplied: boolPtr(true), // We always apply normalization
		MatchScore:           float32Ptr(match.Confidence),
	})

	return err
}

// getTeamTranslations gets English translations for a team
func (j *APIFootballTeamMatchingJob) getTeamTranslations(ctx context.Context, team generated.Team) (*services.TeamTranslations, error) {
	// Get country value for translation
	countryStr := ""
	if team.Country != nil {
		countryStr = *team.Country
	}

	// Translate team name
	teamName, err := j.matcher.GetTeamNameWithAI(ctx, team.Name, countryStr)
	if err != nil {
		// Fallback to basic translation
		teamName = team.Name
	}

	// Translate country
	country := ""
	if team.Country != nil && *team.Country != "" {
		// Use the enhanced translator's country mapping
		enhancedTranslator := services.NewEnhancedTranslator(os.Getenv("OPENAI_API_KEY"))
		country = enhancedTranslator.TranslateCountryName(*team.Country)
	}

	// Get league name from team's recent event participation
	league := ""

	// Look up recent events to find league context
	recentEvents, err := j.db.GetEventsByTeam(ctx, generated.GetEventsByTeamParams{
		TeamID:     team.ID,
		SinceDate:  pgtype.Timestamp{Time: time.Now().Add(-60 * 24 * time.Hour), Valid: true}, // Look back 60 days
		LimitCount: 3,                                                                         // Check a few recent events
	})

	if err == nil && len(recentEvents) > 0 {
		// Use league ID from the most recent event to look up league name
		for _, event := range recentEvents {
			if event.LeagueID != nil {
				// Get league details
				if leagueData, err := j.db.GetLeague(ctx, *event.LeagueID); err == nil {
					// Translate the league name
					enhancedTranslator := services.NewEnhancedTranslator(os.Getenv("OPENAI_API_KEY"))
					league, _ = enhancedTranslator.TranslateLeagueName(ctx, leagueData.Name, country)
					break
				}
			}
		}
	}

	return &services.TeamTranslations{
		TeamName: teamName,
		Country:  country,
		League:   league,
		Original: team,
	}, nil
}

// Timeout returns the job timeout duration
func (j *APIFootballTeamMatchingJob) Timeout() time.Duration {
	return 45 * time.Minute // Longer timeout for team matching
}
