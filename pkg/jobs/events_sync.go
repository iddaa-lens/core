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

	// Fetch events from iddaa API
	// Parameters: st=1 (sport type), type=0 (all events), version=0 (full data)
	url := "https://sportsbookv2.iddaa.com/sportsbook/events?st=1&type=0&version=0"

	data, err := j.iddaaClient.FetchData(url)
	if err != nil {
		return fmt.Errorf("failed to fetch events data: %w", err)
	}

	// Parse the response
	var response models.IddaaEventsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return fmt.Errorf("failed to unmarshal events response: %w", err)
	}

	// Process and store the events
	if err := j.eventsService.ProcessEventsResponse(ctx, &response); err != nil {
		return fmt.Errorf("failed to process events response: %w", err)
	}

	log.Printf("Events sync completed successfully. Processed %d events",
		len(response.Data.Events))

	return nil
}

// Schedule returns the cron schedule for this job
func (j *EventsSyncJob) Schedule() string {
	// Run every 5 minutes to capture rapid odds movements
	return "*/5 * * * *"
}
