package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/services"
)

type DistributionSyncJob struct {
	distributionService *services.DistributionService
	db                  *generated.Queries
}

func NewDistributionSyncJob(distributionService *services.DistributionService, db *generated.Queries) Job {
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

	// Process sports concurrently
	totalProcessed, errorCount := j.processSportsConcurrently(ctx, sports)

	duration := time.Since(start)
	log.LogJobComplete("distribution_sync", duration, totalProcessed, errorCount)
	return nil
}

// processSportsConcurrently processes multiple sports in parallel with controlled concurrency
func (j *DistributionSyncJob) processSportsConcurrently(ctx context.Context, sports []generated.Sport) (int, int) {
	const maxConcurrency = 3 // Process up to 3 sports simultaneously

	type result struct {
		sportID   int32
		sportName string
		err       error
		duration  time.Duration
	}

	// Create channels for work distribution
	workCh := make(chan generated.Sport, len(sports))
	resultCh := make(chan result, len(sports))

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for sport := range workCh {
				start := time.Now()
				err := j.distributionService.FetchAndUpdateDistributions(ctx, int(sport.ID))

				resultCh <- result{
					sportID:   sport.ID,
					sportName: sport.Name,
					err:       err,
					duration:  time.Since(start),
				}
			}
		}(i)
	}

	// Send work to channel
	go func() {
		for _, sport := range sports {
			workCh <- sport
		}
		close(workCh)
	}()

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	totalProcessed := 0
	errorCount := 0
	log := logger.WithContext(ctx, "distribution-sync")

	for res := range resultCh {
		if res.err != nil {
			errorCount++
			log.Error().
				Err(res.err).
				Str("sport_name", res.sportName).
				Int32("sport_id", res.sportID).
				Dur("duration", res.duration).
				Msg("Failed to sync distribution for sport")
		} else {
			totalProcessed++
			log.Debug().
				Str("sport_name", res.sportName).
				Int32("sport_id", res.sportID).
				Dur("duration", res.duration).
				Msg("Distribution sync completed for sport")
		}
	}

	return totalProcessed, errorCount
}

func (j *DistributionSyncJob) Schedule() string {
	// Run every 15 minutes to track betting distribution changes
	return "*/15 * * * *"
}
