package jobs

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models"
	"github.com/iddaa-lens/core/pkg/services"
)

// DetailedOddsSyncJob handles high-frequency detailed odds synchronization for live and near-live events
type DetailedOddsSyncJob struct {
	queries *generated.Queries
	client  services.IddaaClientInterface
	events  services.EventsServiceInterface
	logger  *logger.Logger
}

// eventResult holds the result of processing a single event
type eventResult struct {
	eventID    int
	externalID int
	markets    []models.IddaaDetailedMarket
	err        error
	duration   time.Duration
}

// NewDetailedOddsSyncJob creates a new detailed odds sync job
func NewDetailedOddsSyncJob(queries *generated.Queries, client services.IddaaClientInterface, events services.EventsServiceInterface) *DetailedOddsSyncJob {
	return &DetailedOddsSyncJob{
		queries: queries,
		client:  client,
		events:  events,
		logger:  logger.New("detailed-odds-sync"),
	}
}

// Name returns the job name for CLI execution
func (j *DetailedOddsSyncJob) Name() string {
	return "detailed_odds"
}

// Schedule returns the cron schedule - every 2 minutes for high-frequency tracking
func (j *DetailedOddsSyncJob) Schedule() string {
	return "*/2 * * * *"
}

// Execute runs the detailed odds synchronization with parallel processing
func (j *DetailedOddsSyncJob) Execute(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	ctx = timeoutCtx

	start := time.Now()

	j.logger.Info().
		Str("action", "sync_start").
		Msg("Starting optimized detailed odds sync job")

	// Get ALL active events (no limit - we'll process them all efficiently)
	activeEvents, err := j.queries.GetAllActiveEventsForDetailedSync(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active events: %w", err)
	}

	if len(activeEvents) == 0 {
		j.logger.Info().Msg("No active events found for detailed sync")
		return nil
	}

	j.logger.Info().
		Str("action", "events_fetched").
		Int("total_events", len(activeEvents)).
		Msg("Found ALL active events for detailed odds sync")

	// Process events in smart batches to handle large volumes
	totalSuccess, totalErrors := j.processBatchedEvents(ctx, activeEvents)

	duration := time.Since(start)
	j.logger.Info().
		Str("action", "job_complete").
		Str("job_name", "detailed_odds_sync").
		Dur("duration", duration).
		Int("total_events", len(activeEvents)).
		Int("successful_events", totalSuccess).
		Int("failed_events", totalErrors).
		Float64("success_rate", float64(totalSuccess)/float64(len(activeEvents))*100).
		Bool("has_errors", totalErrors > 0).
		Msg("All active events processing completed")

	return nil
}

// processBatchedEvents processes all events in manageable batches
func (j *DetailedOddsSyncJob) processBatchedEvents(ctx context.Context, allEvents []generated.Event) (int, int) {
	const batchSize = 50 // Larger batches with increased parallelism
	totalSuccess := 0
	totalErrors := 0

	j.logger.Info().
		Int("total_events", len(allEvents)).
		Int("batch_size", batchSize).
		Int("total_batches", (len(allEvents)+batchSize-1)/batchSize).
		Msg("Starting batched processing of all active events")

	for i := 0; i < len(allEvents); i += batchSize {
		end := i + batchSize
		if end > len(allEvents) {
			end = len(allEvents)
		}

		batch := allEvents[i:end]
		batchNum := (i / batchSize) + 1
		totalBatches := (len(allEvents) + batchSize - 1) / batchSize

		j.logger.Info().
			Int("batch_number", batchNum).
			Int("total_batches", totalBatches).
			Int("batch_size", len(batch)).
			Int("events_remaining", len(allEvents)-end).
			Msg("Processing batch")

		// Process this batch
		batchStart := time.Now()
		eventResults := j.parallelFetchEventData(ctx, batch)
		successCount, errorCount := j.bulkProcessMarkets(ctx, eventResults)

		totalSuccess += successCount
		totalErrors += errorCount

		j.logger.Info().
			Int("batch_number", batchNum).
			Dur("batch_duration", time.Since(batchStart)).
			Int("batch_success", successCount).
			Int("batch_errors", errorCount).
			Int("total_success", totalSuccess).
			Int("total_errors", totalErrors).
			Msg("Batch processing completed")

		// Brief pause between batches to respect rate limits
		if end < len(allEvents) {
			time.Sleep(2 * time.Second)
		}
	}

	return totalSuccess, totalErrors
}

// // Legacy Execute method content moved to processBatchedEvents
// func (j *DetailedOddsSyncJob) executeLegacyBatch(ctx context.Context, activeEvents []generated.Event) (int, int) {

// 	// Step 1: Parallel fetch all event data
// 	eventResults := j.parallelFetchEventData(ctx, activeEvents)

// 	// Step 2: Bulk process all markets
// 	successCount, errorCount := j.bulkProcessMarkets(ctx, eventResults)

// 	return successCount, errorCount
// }

// parallelFetchEventData fetches all event data in parallel with controlled concurrency
func (j *DetailedOddsSyncJob) parallelFetchEventData(ctx context.Context, activeEvents []generated.Event) []eventResult {
	const maxConcurrency = 25 // Increased concurrent API calls for faster fetching
	semaphore := make(chan struct{}, maxConcurrency)
	results := make([]eventResult, len(activeEvents))
	var wg sync.WaitGroup

	j.logger.Info().
		Int("event_count", len(activeEvents)).
		Int("max_concurrency", maxConcurrency).
		Msg("Starting parallel event data fetch")

	for i, event := range activeEvents {
		wg.Add(1)
		go func(index int, evt generated.Event) {
			defer wg.Done()

			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			start := time.Now()
			externalID, err := strconv.Atoi(evt.ExternalID)
			if err != nil {
				results[index] = eventResult{
					eventID:    int(evt.ID),
					externalID: 0,
					err:        fmt.Errorf("invalid external ID: %w", err),
					duration:   time.Since(start),
				}
				return
			}

			// Individual event timeout (context used in API call)
			_ = ctx // API call will use the main context with its own timeout

			// Fetch event data
			eventResponse, err := j.client.GetSingleEvent(externalID)
			if err != nil {
				results[index] = eventResult{
					eventID:    int(evt.ID),
					externalID: externalID,
					err:        fmt.Errorf("failed to fetch event %d: %w", externalID, err),
					duration:   time.Since(start),
				}
				return
			}

			results[index] = eventResult{
				eventID:    int(evt.ID),
				externalID: externalID,
				markets:    eventResponse.Data.Markets,
				err:        nil,
				duration:   time.Since(start),
			}

			j.logger.Debug().
				Int("event_id", int(evt.ID)).
				Int("external_id", externalID).
				Int("markets_count", len(eventResponse.Data.Markets)).
				Dur("duration", time.Since(start)).
				Msg("Event data fetched successfully")
		}(i, event)
	}

	wg.Wait()

	// Log summary
	successful := 0
	failed := 0
	for _, result := range results {
		if result.err == nil {
			successful++
		} else {
			failed++
		}
	}

	j.logger.Info().
		Int("successful_fetches", successful).
		Int("failed_fetches", failed).
		Msg("Parallel fetch completed")

	return results
}

// bulkProcessMarkets processes all markets using worker pool pattern for parallel processing
func (j *DetailedOddsSyncJob) bulkProcessMarkets(ctx context.Context, eventResults []eventResult) (int, int) {
	const marketWorkers = 10 // Number of workers processing markets in parallel

	type processResult struct {
		success bool
		eventID int
	}

	// Create channels for work distribution and results
	workChan := make(chan eventResult, len(eventResults))
	resultChan := make(chan processResult, len(eventResults))

	j.logger.Info().
		Int("worker_count", marketWorkers).
		Int("events_to_process", len(eventResults)).
		Msg("Starting parallel market processing with worker pool")

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := range marketWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for result := range workChan {
				// Skip events with fetch errors
				if result.err != nil {
					j.logger.Error().
						Err(result.err).
						Int("event_id", result.eventID).
						Int("external_id", result.externalID).
						Int("worker_id", workerID).
						Msg("Skipping event due to fetch error")
					resultChan <- processResult{success: false, eventID: result.eventID}
					continue
				}

				if len(result.markets) == 0 {
					j.logger.Debug().
						Int("event_id", result.eventID).
						Int("worker_id", workerID).
						Msg("No markets found for event")
					resultChan <- processResult{success: true, eventID: result.eventID}
					continue
				}

				// Process markets for this event
				start := time.Now()
				err := j.events.ProcessDetailedMarkets(ctx, result.eventID, result.markets)
				if err != nil {
					j.logger.Error().
						Err(err).
						Int("event_id", result.eventID).
						Int("external_id", result.externalID).
						Int("markets_count", len(result.markets)).
						Int("worker_id", workerID).
						Dur("processing_duration", time.Since(start)).
						Msg("Failed to process markets for event")
					resultChan <- processResult{success: false, eventID: result.eventID}
				} else {
					j.logger.Debug().
						Int("event_id", result.eventID).
						Int("external_id", result.externalID).
						Int("markets_count", len(result.markets)).
						Int("worker_id", workerID).
						Dur("processing_duration", time.Since(start)).
						Dur("total_duration", result.duration+time.Since(start)).
						Msg("Successfully processed markets for event")
					resultChan <- processResult{success: true, eventID: result.eventID}
				}
			}
		}(i)
	}

	// Send all work to the channel
	go func() {
		for _, result := range eventResults {
			workChan <- result
		}
		close(workChan)
	}()

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Count results
	successCount := 0
	errorCount := 0
	for result := range resultChan {
		if result.success {
			successCount++
		} else {
			errorCount++
		}
	}

	j.logger.Info().
		Int("successful_events", successCount).
		Int("failed_events", errorCount).
		Msg("Parallel market processing completed")

	return successCount, errorCount
}

// // Legacy method for compatibility - now calls the optimized version
// func (j *DetailedOddsSyncJob) syncEventDetails(ctx context.Context, eventID int, externalEventID int) error {
// 	// This method is kept for compatibility but should use the new parallel approach
// 	eventCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
// 	defer cancel()

// 	eventResponse, err := j.client.GetSingleEvent(externalEventID)
// 	if err != nil {
// 		return fmt.Errorf("failed to fetch event details: %w", err)
// 	}

// 	return j.events.ProcessDetailedMarkets(eventCtx, eventID, eventResponse.Data.Markets)
// }
