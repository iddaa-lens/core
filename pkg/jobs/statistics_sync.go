package jobs

import (
	"context"
	"log"
	"time"

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
	log.Printf("Starting statistics sync job for sport type %d...", j.sportType)

	// Sync today's events
	today := time.Now().Format("2006-01-02")
	err := j.statisticsService.SyncEventStatistics(ctx, j.sportType, today)
	if err != nil {
		log.Printf("Failed to sync today's statistics: %v", err)
		// Continue with yesterday's events even if today fails
	}

	// Also sync yesterday's events to catch any late updates
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	err = j.statisticsService.SyncEventStatistics(ctx, j.sportType, yesterday)
	if err != nil {
		log.Printf("Failed to sync yesterday's statistics: %v", err)
	}

	log.Printf("Statistics sync completed successfully")
	return nil
}

func (j *StatisticsSyncJob) Schedule() string {
	// Run every 15 minutes during active hours (8 AM to 11 PM)
	// This covers most European football match times
	return "*/15 8-23 * * *"
}
