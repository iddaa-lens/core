package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/betslib/iddaa-core/pkg/database"
	"github.com/betslib/iddaa-core/pkg/logger"
	"github.com/betslib/iddaa-core/pkg/services"
)

type DistributionSyncJob struct {
	distributionService *services.DistributionService
	db                  *database.Queries
}

func NewDistributionSyncJob(distributionService *services.DistributionService, db *database.Queries) Job {
	return &DistributionSyncJob{
		distributionService: distributionService,
		db:                  db,
	}
}

func (j *DistributionSyncJob) Name() string {
	return "distribution_sync"
}

func (j *DistributionSyncJob) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx, "distribution-sync")
	start := time.Now()

	log.Info().
		Str("action", "sync_start").
		Msg("Starting distribution sync job")

	// Fetch all active sports from database
	sports, err := j.db.ListSports(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch sports from database: %w", err)
	}

	if len(sports) == 0 {
		log.Warn().
			Str("action", "no_sports").
			Msg("No active sports found in database")
		return nil
	}

	log.Info().
		Str("action", "sports_fetched").
		Int("sport_count", len(sports)).
		Msg("Found active sports to sync")

	totalProcessed := 0
	errorCount := 0

	for _, sport := range sports {
		sportStart := time.Now()
		log.Debug().
			Str("action", "sport_sync_start").
			Str("sport_name", sport.Name).
			Int("sport_id", int(sport.ID)).
			Msg("Fetching distribution data for sport")

		err := j.distributionService.FetchAndUpdateDistributions(ctx, int(sport.ID))
		if err != nil {
			errorCount++
			log.Error().
				Err(err).
				Str("action", "sport_sync_failed").
				Str("sport_name", sport.Name).
				Int("sport_id", int(sport.ID)).
				Dur("duration", time.Since(sportStart)).
				Msg("Failed to fetch distribution data")
			continue // Continue with other sports
		}

		log.Debug().
			Str("action", "sport_sync_complete").
			Str("sport_name", sport.Name).
			Int("sport_id", int(sport.ID)).
			Dur("duration", time.Since(sportStart)).
			Msg("Distribution sync completed for sport")
		totalProcessed++
	}

	duration := time.Since(start)
	log.LogJobComplete("distribution_sync", duration, totalProcessed, errorCount)
	return nil
}

func (j *DistributionSyncJob) Schedule() string {
	// Run every hour to track betting distribution changes
	return "0 * * * *"
}
