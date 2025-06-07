package jobs

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/services"
)

// DetailedOddsSyncJob handles high-frequency detailed odds synchronization for live and near-live events
type DetailedOddsSyncJob struct {
	queries *database.Queries
	client  services.IddaaClientInterface
	events  services.EventsServiceInterface
}

// NewDetailedOddsSyncJob creates a new detailed odds sync job
func NewDetailedOddsSyncJob(queries *database.Queries, client services.IddaaClientInterface, events services.EventsServiceInterface) *DetailedOddsSyncJob {
	return &DetailedOddsSyncJob{
		queries: queries,
		client:  client,
		events:  events,
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

// Execute runs the detailed odds synchronization
func (j *DetailedOddsSyncJob) Execute(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	ctx = timeoutCtx

	log := logger.WithContext(ctx, "detailed-odds-sync")
	start := time.Now()

	log.Info().
		Str("action", "sync_start").
		Msg("Starting detailed odds sync job")

	// Get active events (live + scheduled within next 24 hours)
	activeEvents, err := j.queries.GetActiveEventsForDetailedSync(ctx, 100) // Limit to prevent overload
	if err != nil {
		return fmt.Errorf("failed to get active events: %w", err)
	}

	log.Info().
		Str("action", "events_fetched").
		Int("event_count", len(activeEvents)).
		Int("limit", 100).
		Msg("Found active events for detailed odds sync")

	successCount := 0
	errorCount := 0

	for _, event := range activeEvents {
		eventStart := time.Now()
		externalID, err := strconv.Atoi(event.ExternalID)
		if err != nil {
			errorCount++
			log.Error().
				Err(err).
				Str("action", "invalid_external_id").
				Int("event_id", int(event.ID)).
				Str("external_id", event.ExternalID).
				Msg("Invalid external ID for event")
			continue
		}

		err = j.syncEventDetails(ctx, int(event.ID), externalID)
		if err != nil {
			errorCount++
			log.Error().
				Err(err).
				Str("action", "event_sync_failed").
				Int("event_id", int(event.ID)).
				Int("external_id", externalID).
				Dur("duration", time.Since(eventStart)).
				Msg("Failed to sync detailed odds for event")
		} else {
			successCount++
			log.Debug().
				Str("action", "event_sync_complete").
				Int("event_id", int(event.ID)).
				Int("external_id", externalID).
				Dur("duration", time.Since(eventStart)).
				Msg("Event detailed odds sync completed")
		}

		// Rate limiting: small delay between requests to avoid overwhelming API
		time.Sleep(100 * time.Millisecond)
	}

	duration := time.Since(start)
	log.LogJobComplete("detailed_odds_sync", duration, successCount, errorCount)
	return nil
}

// syncEventDetails fetches and processes detailed odds for a specific event
func (j *DetailedOddsSyncJob) syncEventDetails(ctx context.Context, eventID int, externalEventID int) error {
	// Add per-event timeout to prevent individual events from blocking the entire job
	eventCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// Fetch detailed event data using the external event ID for the API call
	eventResponse, err := j.client.GetSingleEvent(externalEventID)
	if err != nil {
		return fmt.Errorf("failed to fetch event details: %w", err)
	}

	// Process the detailed markets and odds using the internal event ID
	return j.events.ProcessDetailedMarkets(eventCtx, eventID, eventResponse.Data.Markets, time.Now())
}
