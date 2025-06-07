package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/iddaa-lens/core/pkg/apifootball"
	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/jackc/pgx/v5/pgtype"
)

// APIFootballTeamEnrichmentJob handles team enrichment with API-Football data
type APIFootballTeamEnrichmentJob struct {
	db        *database.Queries
	apiclient *apifootball.Client
}

// NewAPIFootballTeamEnrichmentJob creates a new API-Football team enrichment job
func NewAPIFootballTeamEnrichmentJob(db *database.Queries) *APIFootballTeamEnrichmentJob {
	apiKey := os.Getenv("API_FOOTBALL_API_KEY")

	// Create API-Football client
	apiConfig := apifootball.DefaultConfig(apiKey)
	apiclient := apifootball.NewClient(apiConfig)

	return &APIFootballTeamEnrichmentJob{
		db:        db,
		apiclient: apiclient,
	}
}

// Name returns the job name
func (j *APIFootballTeamEnrichmentJob) Name() string {
	return "api_football_team_enrichment"
}

// Schedule returns the cron schedule - run monthly on the 1st at 3 AM
func (j *APIFootballTeamEnrichmentJob) Schedule() string {
	return "0 3 1 * *"
}

// Execute runs the team enrichment process
func (j *APIFootballTeamEnrichmentJob) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx, "api-football-team-enrichment")
	start := time.Now()

	log.Info().
		Str("action", "sync_start").
		Msg("Starting API-Football team enrichment job")

	// Check if API client is available
	if !j.apiclient.IsAvailable() {
		log.Warn().
			Str("action", "api_key_missing").
			Msg("API_FOOTBALL_API_KEY not set, skipping team enrichment")
		return nil
	}

	// Step 1: Get teams that need enrichment (mapped teams without recent API data)
	teamsToEnrich, err := j.getTeamsNeedingEnrichment(ctx)
	if err != nil {
		return fmt.Errorf("failed to get teams needing enrichment: %w", err)
	}

	if len(teamsToEnrich) == 0 {
		log.Info().
			Str("action", "no_teams_to_enrich").
			Msg("No teams need enrichment")

		duration := time.Since(start)
		log.LogJobComplete("api_football_team_enrichment", duration, 0, 0)
		return nil
	}

	log.Info().
		Str("action", "teams_found").
		Int("team_count", len(teamsToEnrich)).
		Msg("Found teams that need enrichment")

	// Step 2: Process each team
	successCount := 0
	errorCount := 0

	for i, team := range teamsToEnrich {
		// Rate limiting between requests
		if i > 0 {
			time.Sleep(1 * time.Second) // Longer delay for enrichment calls
		}

		teamStart := time.Now()
		log.Debug().
			Str("action", "team_enrichment_start").
			Int("team_id", int(team.ID)).
			Str("team_name", team.Name).
			Msg("Starting team enrichment")

		// Get team mapping to find API-Football ID
		apiFootballID, err := j.getAPIFootballIDForTeam(ctx, team)
		if err != nil {
			errorCount++
			log.Error().
				Err(err).
				Str("action", "api_id_lookup_failed").
				Int("team_id", int(team.ID)).
				Str("team_name", team.Name).
				Msg("Failed to get API-Football ID for team")
			continue
		}

		if apiFootballID == 0 {
			log.Debug().
				Str("action", "no_api_mapping").
				Int("team_id", int(team.ID)).
				Str("team_name", team.Name).
				Msg("Team has no API-Football mapping, skipping enrichment")
			continue
		}

		// Fetch detailed team data from API-Football
		err = j.enrichTeamData(ctx, team, apiFootballID)
		if err != nil {
			errorCount++
			log.Error().
				Err(err).
				Str("action", "team_enrichment_failed").
				Int("team_id", int(team.ID)).
				Str("team_name", team.Name).
				Int("api_football_id", apiFootballID).
				Dur("duration", time.Since(teamStart)).
				Msg("Failed to enrich team data")
			continue
		}

		successCount++
		log.Info().
			Str("action", "team_enriched").
			Int("team_id", int(team.ID)).
			Str("team_name", team.Name).
			Int("api_football_id", apiFootballID).
			Dur("duration", time.Since(teamStart)).
			Msg("Team successfully enriched")
	}

	duration := time.Since(start)
	log.LogJobComplete("api_football_team_enrichment", duration, successCount, errorCount)

	if errorCount > 0 {
		log.Warn().
			Int("success_count", successCount).
			Int("error_count", errorCount).
			Int("total_processed", len(teamsToEnrich)).
			Msg("Team enrichment completed with some errors")
	} else {
		log.Info().
			Int("success_count", successCount).
			Int("total_processed", len(teamsToEnrich)).
			Msg("Team enrichment completed successfully")
	}

	return nil
}

// getTeamsNeedingEnrichment returns teams that need API-Football enrichment
func (j *APIFootballTeamEnrichmentJob) getTeamsNeedingEnrichment(ctx context.Context) ([]database.Team, error) {
	// Get teams that either have no API enrichment or haven't been updated in 7 days
	return j.db.GetTeamsNeedingEnrichment(ctx, 50) // Process 50 teams per run
}

// getAPIFootballIDForTeam gets the API-Football ID for a team from team mappings
func (j *APIFootballTeamEnrichmentJob) getAPIFootballIDForTeam(ctx context.Context, team database.Team) (int, error) {
	// Try to get team mapping
	mapping, err := j.db.GetTeamMapping(ctx, team.ID)
	if err != nil {
		// No mapping found
		return 0, nil
	}

	return int(mapping.FootballApiTeamID), nil
}

// enrichTeamData fetches detailed team data from API-Football and updates the database
func (j *APIFootballTeamEnrichmentJob) enrichTeamData(ctx context.Context, team database.Team, apiFootballID int) error {
	// Fetch detailed team data by ID
	teamData, err := j.apiclient.GetTeamByID(ctx, apiFootballID)
	if err != nil {
		return fmt.Errorf("failed to fetch team data from API-Football: %w", err)
	}

	if teamData == nil {
		return fmt.Errorf("team with ID %d not found in API-Football", apiFootballID)
	}

	// Prepare enrichment data
	enrichmentData := map[string]interface{}{
		"team":            teamData.Team,
		"venue":           teamData.Venue,
		"enrichment_date": time.Now().UTC(),
		"api_response":    teamData,
	}

	enrichmentJSON, err := json.Marshal(enrichmentData)
	if err != nil {
		return fmt.Errorf("failed to marshal enrichment data: %w", err)
	}

	// Convert values to appropriate pgtype values
	apiFootballIDPg := pgtype.Int4{Int32: int32(apiFootballID), Valid: true}

	teamCodePg := pgtype.Text{String: teamData.Team.Code, Valid: teamData.Team.Code != ""}

	var foundedYearPg pgtype.Int4
	if teamData.Team.Founded > 0 {
		foundedYearPg = pgtype.Int4{Int32: int32(teamData.Team.Founded), Valid: true}
	}

	isNationalPg := pgtype.Bool{Bool: teamData.Team.National, Valid: true}

	var venueIDPg pgtype.Int4
	if teamData.Venue.ID > 0 {
		venueIDPg = pgtype.Int4{Int32: int32(teamData.Venue.ID), Valid: true}
	}

	venueNamePg := pgtype.Text{String: teamData.Venue.Name, Valid: teamData.Venue.Name != ""}

	venueAddressPg := pgtype.Text{String: teamData.Venue.Address, Valid: teamData.Venue.Address != ""}

	venueCityPg := pgtype.Text{String: teamData.Venue.City, Valid: teamData.Venue.City != ""}

	var venueCapacityPg pgtype.Int4
	if teamData.Venue.Capacity > 0 {
		venueCapacityPg = pgtype.Int4{Int32: int32(teamData.Venue.Capacity), Valid: true}
	}

	venueSurfacePg := pgtype.Text{String: teamData.Venue.Surface, Valid: teamData.Venue.Surface != ""}

	venueImagePg := pgtype.Text{String: teamData.Venue.Image, Valid: teamData.Venue.Image != ""}

	// Update team with enrichment data
	_, err = j.db.EnrichTeamWithAPIFootball(ctx, database.EnrichTeamWithAPIFootballParams{
		ID:                team.ID,
		ApiFootballID:     apiFootballIDPg,
		TeamCode:          teamCodePg,
		FoundedYear:       foundedYearPg,
		IsNationalTeam:    isNationalPg,
		VenueID:           venueIDPg,
		VenueName:         venueNamePg,
		VenueAddress:      venueAddressPg,
		VenueCity:         venueCityPg,
		VenueCapacity:     venueCapacityPg,
		VenueSurface:      venueSurfacePg,
		VenueImageUrl:     venueImagePg,
		ApiEnrichmentData: enrichmentJSON,
	})

	return err
}

// Timeout returns the job timeout duration
func (j *APIFootballTeamEnrichmentJob) Timeout() time.Duration {
	return 60 * time.Minute // Longer timeout for enrichment
}
