package jobs

import (
	"context"
	"time"

	"github.com/betslib/iddaa-core/pkg/logger"
	"github.com/betslib/iddaa-core/pkg/services"
)

type StatisticsSyncJob struct {
	statisticsService *services.StatisticsService
	sportType         int
}

func NewStatisticsSyncJob(statisticsService *services.StatisticsService, sportType int) Job {
	return &StatisticsSyncJob{
		statisticsService: statisticsService,
		sportType:         sportType,
	}
}

func (j *StatisticsSyncJob) Name() string {
	return "statistics_sync"
}

func (j *StatisticsSyncJob) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx, "statistics-sync")
	start := time.Now()

	log.Info().
		Str("action", "sync_start").
		Int("sport_type", j.sportType).
		Msg("Starting statistics sync job")

	errorCount := 0
	processedDates := 0

	// Sync today's events
	today := time.Now().Format("2006-01-02")
	dateStart := time.Now()
	log.Debug().
		Str("action", "date_sync_start").
		Str("date", today).
		Str("date_type", "today").
		Msg("Syncing statistics for date")

	err := j.statisticsService.SyncEventStatistics(ctx, j.sportType, today)
	if err != nil {
		errorCount++
		log.Error().
			Err(err).
			Str("action", "date_sync_failed").
			Str("date", today).
			Str("date_type", "today").
			Dur("duration", time.Since(dateStart)).
			Msg("Failed to sync today's statistics")
		// Continue with yesterday's events even if today fails
	} else {
		processedDates++
		log.Debug().
			Str("action", "date_sync_complete").
			Str("date", today).
			Str("date_type", "today").
			Dur("duration", time.Since(dateStart)).
			Msg("Today's statistics sync completed")
	}

	// Also sync yesterday's events to catch any late updates
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	dateStart = time.Now()
	log.Debug().
		Str("action", "date_sync_start").
		Str("date", yesterday).
		Str("date_type", "yesterday").
		Msg("Syncing statistics for date")

	err = j.statisticsService.SyncEventStatistics(ctx, j.sportType, yesterday)
	if err != nil {
		errorCount++
		log.Error().
			Err(err).
			Str("action", "date_sync_failed").
			Str("date", yesterday).
			Str("date_type", "yesterday").
			Dur("duration", time.Since(dateStart)).
			Msg("Failed to sync yesterday's statistics")
	} else {
		processedDates++
		log.Debug().
			Str("action", "date_sync_complete").
			Str("date", yesterday).
			Str("date_type", "yesterday").
			Dur("duration", time.Since(dateStart)).
			Msg("Yesterday's statistics sync completed")
	}

	duration := time.Since(start)
	log.LogJobComplete("statistics_sync", duration, processedDates, errorCount)
	return nil
}

func (j *StatisticsSyncJob) Schedule() string {
	// Run every 15 minutes during active hours (8 AM to 11 PM)
	// This covers most European football match times
	return "*/15 8-23 * * *"
}
