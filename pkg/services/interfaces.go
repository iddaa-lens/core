package services

import (
	"context"
	"time"

	"github.com/iddaa-lens/core/pkg/models"
)

// CompetitionSyncer defines the interface for competition synchronization
type CompetitionSyncer interface {
	SyncCompetitions(ctx context.Context) error
}

// IddaaClientInterface defines the interface for Iddaa API client
type IddaaClientInterface interface {
	GetSingleEvent(eventID int) (*models.IddaaSingleEventResponse, error)
	GetSportInfo() (*models.IddaaAPIResponse[models.IddaaSportInfo], error)
	GetEvents(sportID int) (*models.IddaaEventsResponse, error)
}

// EventsServiceInterface defines the interface for events service
type EventsServiceInterface interface {
	ProcessDetailedMarkets(ctx context.Context, eventID int, markets []models.IddaaDetailedMarket, timestamp time.Time) error
}
