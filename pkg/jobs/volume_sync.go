package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/betslib/iddaa-core/pkg/database"
	"github.com/betslib/iddaa-core/pkg/logger"
	"github.com/betslib/iddaa-core/pkg/services"
)

type VolumeSyncJob struct {
	volumeService *services.VolumeService
	db            *database.Queries
}

func NewVolumeSyncJob(volumeService *services.VolumeService, db *database.Queries) Job {
	return &VolumeSyncJob{
		volumeService: volumeService,
		db:            db,
	}
}

func (j *VolumeSyncJob) Name() string {
	return "volume_sync"
}

func (j *VolumeSyncJob) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx, "volume-sync")
	start := time.Now()

	log.Info().
		Str("action", "sync_start").
		Msg("Starting volume sync job")

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
			Msg("Fetching volume data for sport")

		err := j.volumeService.FetchAndUpdateVolumes(ctx, int(sport.ID))
		if err != nil {
			errorCount++
			log.Error().
				Err(err).
				Str("action", "sport_sync_failed").
				Str("sport_name", sport.Name).
				Int("sport_id", int(sport.ID)).
				Dur("duration", time.Since(sportStart)).
				Msg("Failed to fetch volume data")
			continue // Continue with other sports
		}

		log.Debug().
			Str("action", "sport_sync_complete").
			Str("sport_name", sport.Name).
			Int("sport_id", int(sport.ID)).
			Dur("duration", time.Since(sportStart)).
			Msg("Volume sync completed for sport")
		totalProcessed++
	}

	duration := time.Since(start)
	log.LogJobComplete("volume_sync", duration, totalProcessed, errorCount)
	return nil
}

func (j *VolumeSyncJob) Schedule() string {
	// Run every 20 minutes to track volume changes
	return "*/20 * * * *"
}
