package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/betslib/iddaa-core/pkg/logger"
	"github.com/betslib/iddaa-core/pkg/models"
	"github.com/betslib/iddaa-core/pkg/services"
)

type EventsSyncJob struct {
	iddaaClient   *services.IddaaClient
	eventsService *services.EventsService
}

func NewEventsSyncJob(iddaaClient *services.IddaaClient, eventsService *services.EventsService) Job {
	return &EventsSyncJob{
		iddaaClient:   iddaaClient,
		eventsService: eventsService,
	}
}

func (j *EventsSyncJob) Name() string {
	return "events_sync"
}

func (j *EventsSyncJob) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx, "events-sync")
	start := time.Now()

	log.Info().
		Str("action", "sync_start").
		Msg("Starting events sync job")

	// Fetch all active sports from database
	sports, err := j.eventsService.GetActiveSports(ctx)
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

	totalEvents := 0
	errorCount := 0

	for _, sport := range sports {
		sportStart := time.Now()
		log.Debug().
			Str("action", "sport_sync_start").
			Str("sport_name", sport.Name).
			Int("sport_id", int(sport.ID)).
			Msg("Fetching events for sport")

		// Fetch events from iddaa API for this sport
		url := fmt.Sprintf("https://sportsbookv2.iddaa.com/sportsbook/events?st=%d&type=0&version=0", sport.ID)

		data, err := j.iddaaClient.FetchData(url)
		if err != nil {
			errorCount++
			log.Error().
				Err(err).
				Str("action", "fetch_failed").
				Str("sport_name", sport.Name).
				Int("sport_id", int(sport.ID)).
				Str("url", url).
				Msg("Failed to fetch events")
			continue // Continue with other sports
		}

		// Parse the response
		var response models.IddaaEventsResponse
		if err := json.Unmarshal(data, &response); err != nil {
			errorCount++
			log.Error().
				Err(err).
				Str("action", "unmarshal_failed").
				Str("sport_name", sport.Name).
				Int("sport_id", int(sport.ID)).
				Msg("Failed to unmarshal events response")
			continue // Continue with other sports
		}

		// Process and store the events
		if err := j.eventsService.ProcessEventsResponse(ctx, &response); err != nil {
			errorCount++
			log.Error().
				Err(err).
				Str("action", "process_failed").
				Str("sport_name", sport.Name).
				Int("sport_id", int(sport.ID)).
				Msg("Failed to process events response")
			continue // Continue with other sports
		}

		eventCount := len(response.Data.Events)
		totalEvents += eventCount
		log.Info().
			Str("action", "sport_sync_complete").
			Str("sport_name", sport.Name).
			Int("sport_id", int(sport.ID)).
			Int("event_count", eventCount).
			Dur("duration", time.Since(sportStart)).
			Msg("Processed events for sport")
	}

	duration := time.Since(start)
	log.LogJobComplete("events_sync", duration, totalEvents, errorCount)
	return nil
}

// Schedule returns the cron schedule for this job
func (j *EventsSyncJob) Schedule() string {
	// Run every 5 minutes to capture rapid odds movements
	return "*/5 * * * *"
}
