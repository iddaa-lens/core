package jobs

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/betslib/iddaa-core/pkg/database"
	"github.com/betslib/iddaa-core/pkg/logger"
	"github.com/betslib/iddaa-core/pkg/services"
)

type LeaguesSyncJob struct {
	db             *database.Queries
	leaguesService *services.LeaguesService
}

func NewLeaguesSyncJob(db *database.Queries, iddaaClient *services.IddaaClient) *LeaguesSyncJob {
	// Get Football API key from environment
	apiKey := os.Getenv("FOOTBALL_API_KEY")
	// Note: Missing API keys will be logged when job executes

	// Get OpenAI API key from environment
	openaiKey := os.Getenv("OPENAI_API_KEY")
	// Note: Missing API keys will be logged when job executes

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create leagues service
	leaguesService := services.NewLeaguesService(db, client, apiKey, iddaaClient, openaiKey)

	return &LeaguesSyncJob{
		db:             db,
		leaguesService: leaguesService,
	}
}

func (j *LeaguesSyncJob) Name() string {
	return "leagues_sync"
}

func (j *LeaguesSyncJob) Description() string {
	return "Syncs leagues and teams with Football API"
}

func (j *LeaguesSyncJob) Schedule() string {
	return "0 * * * *" // Every hour at minute 0
}

func (j *LeaguesSyncJob) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx, "leagues-sync")
	start := time.Now()

	log.Info().
		Str("action", "sync_start").
		Msg("Starting leagues sync from Iddaa and Football API")

	// Check if service is initialized
	if j.leaguesService == nil {
		return fmt.Errorf("leagues service is not initialized")
	}

	errorCount := 0
	completedSteps := 0

	// Step 1: Sync leagues from Iddaa competitions endpoint first
	stepStart := time.Now()
	log.Info().
		Str("action", "step_start").
		Str("step", "iddaa_leagues").
		Msg("Syncing leagues from Iddaa")

	if err := j.leaguesService.SyncLeaguesFromIddaa(ctx); err != nil {
		log.Error().
			Err(err).
			Str("action", "step_failed").
			Str("step", "iddaa_leagues").
			Dur("duration", time.Since(stepStart)).
			Msg("Failed to sync Iddaa leagues")
		return fmt.Errorf("failed to sync Iddaa leagues: %w", err)
	}
	completedSteps++
	log.Info().
		Str("action", "step_complete").
		Str("step", "iddaa_leagues").
		Dur("duration", time.Since(stepStart)).
		Msg("Iddaa leagues sync completed")

	// Step 2: Check if Football API key is available for additional enrichment
	apiKey := os.Getenv("FOOTBALL_API_KEY")
	if apiKey == "" {
		log.Warn().
			Str("action", "api_key_missing").
			Str("api", "football_api").
			Msg("FOOTBALL_API_KEY not set, skipping Football API sync")

		duration := time.Since(start)
		log.LogJobComplete("leagues_sync", duration, completedSteps, errorCount)
		return nil
	}

	// Step 3: Sync leagues with Football API for mapping
	stepStart = time.Now()
	log.Info().
		Str("action", "step_start").
		Str("step", "football_api_leagues").
		Msg("Syncing leagues with Football API")

	if err := j.leaguesService.SyncLeaguesWithFootballAPI(ctx); err != nil {
		errorCount++
		log.Error().
			Err(err).
			Str("action", "step_failed").
			Str("step", "football_api_leagues").
			Dur("duration", time.Since(stepStart)).
			Msg("Error syncing leagues with Football API")
		// Don't fail the entire job, Iddaa sync was successful
	} else {
		completedSteps++
		log.Info().
			Str("action", "step_complete").
			Str("step", "football_api_leagues").
			Dur("duration", time.Since(stepStart)).
			Msg("Football API leagues sync completed")
	}

	// Step 4: Sync teams (only for leagues that are already mapped)
	stepStart = time.Now()
	log.Info().
		Str("action", "step_start").
		Str("step", "football_api_teams").
		Msg("Syncing teams with Football API")

	if err := j.leaguesService.SyncTeamsWithFootballAPI(ctx); err != nil {
		errorCount++
		log.Error().
			Err(err).
			Str("action", "step_failed").
			Str("step", "football_api_teams").
			Dur("duration", time.Since(stepStart)).
			Msg("Error syncing teams with Football API")
		// Don't fail the entire job, Iddaa sync was successful
	} else {
		completedSteps++
		log.Info().
			Str("action", "step_complete").
			Str("step", "football_api_teams").
			Dur("duration", time.Since(stepStart)).
			Msg("Football API teams sync completed")
	}

	duration := time.Since(start)
	log.LogJobComplete("leagues_sync", duration, completedSteps, errorCount)

	return nil
}

func (j *LeaguesSyncJob) Timeout() time.Duration {
	return 10 * time.Minute // Allow up to 10 minutes for sync
}
