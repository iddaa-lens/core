package jobs

import (
	"context"
	"time"

	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/services"
)

type SportsSyncJob struct {
	sportsService *services.SportService
}

// NewSportsSyncJob creates a new sports sync job
func NewSportsSyncJob(sportsService *services.SportService) Job {
	return &SportsSyncJob{
		sportsService: sportsService,
	}
}

func (j *SportsSyncJob) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx, "sports-sync")
	start := time.Now()

	log.Info().
		Str("action", "sync_start").
		Msg("Starting sports sync job")

	err := j.sportsService.SyncSports(ctx)
	duration := time.Since(start)

	if err != nil {
		log.Error().
			Err(err).
			Str("action", "sync_failed").
			Dur("duration", duration).
			Msg("Sports sync failed")
		return err
	}

	log.LogJobComplete("sports_sync", duration, 1, 0)
	return nil
}

func (j *SportsSyncJob) Name() string {
	return "sports_sync"
}

func (j *SportsSyncJob) Schedule() string {
	// Run every 30 minutes to keep sports info up to date
	return "*/30 * * * *"
}
