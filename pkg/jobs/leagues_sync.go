package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/services"
)

type LeaguesSyncJob struct {
	db             *generated.Queries
	leaguesService *services.LeaguesService
}

func NewLeaguesSyncJob(db *generated.Queries, iddaaClient *services.IddaaClient) *LeaguesSyncJob {
	// Create leagues service (only needs db and iddaaClient for Iddaa sync)
	leaguesService := services.NewLeaguesService(db, nil, "", iddaaClient, "")

	return &LeaguesSyncJob{
		db:             db,
		leaguesService: leaguesService,
	}
}

func (j *LeaguesSyncJob) Name() string {
	return "leagues_sync"
}

func (j *LeaguesSyncJob) Description() string {
	return "Syncs leagues from Iddaa API"
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

	// Football API league matching has been moved to the dedicated api_football_league_matching job
	// This improves separation of concerns and rate limit management
	// The api_football_league_matching job runs weekly on Tuesdays at 3 AM

	// Team syncing has been moved to the dedicated api_football_team_matching job
	// The api_football_team_matching job runs weekly on Tuesdays at 4 AM

	duration := time.Since(start)
	log.LogJobComplete("leagues_sync", duration, completedSteps, errorCount)

	return nil
}

func (j *LeaguesSyncJob) Timeout() time.Duration {
	return 10 * time.Minute // Allow up to 10 minutes for sync
}
