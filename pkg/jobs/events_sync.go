package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

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
	log.Printf("Starting events sync job...")

	// Fetch all active sports from database
	sports, err := j.eventsService.GetActiveSports(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch sports from database: %w", err)
	}

	if len(sports) == 0 {
		log.Printf("No active sports found in database")
		return nil
	}

	log.Printf("Found %d active sports to sync events for", len(sports))
	totalEvents := 0

	for _, sport := range sports {
		log.Printf("Fetching events for sport %s (ID: %d)...", sport.Name, sport.ID)

		// Fetch events from iddaa API for this sport
		url := fmt.Sprintf("https://sportsbookv2.iddaa.com/sportsbook/events?st=%d&type=0&version=0", sport.ID)

		data, err := j.iddaaClient.FetchData(url)
		if err != nil {
			log.Printf("Failed to fetch events for sport %s (ID: %d): %v", sport.Name, sport.ID, err)
			continue // Continue with other sports
		}

		// Parse the response
		var response models.IddaaEventsResponse
		if err := json.Unmarshal(data, &response); err != nil {
			log.Printf("Failed to unmarshal events response for sport %s (ID: %d): %v", sport.Name, sport.ID, err)
			continue // Continue with other sports
		}

		// Process and store the events
		if err := j.eventsService.ProcessEventsResponse(ctx, &response); err != nil {
			log.Printf("Failed to process events response for sport %s (ID: %d): %v", sport.Name, sport.ID, err)
			continue // Continue with other sports
		}

		eventCount := len(response.Data.Events)
		totalEvents += eventCount
		log.Printf("Processed %d events for sport %s (ID: %d)", eventCount, sport.Name, sport.ID)
	}

	log.Printf("Events sync completed successfully. Total processed: %d events", totalEvents)
	return nil
}

// Schedule returns the cron schedule for this job
func (j *EventsSyncJob) Schedule() string {
	// Run every 5 minutes to capture rapid odds movements
	return "*/5 * * * *"
}
