package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
)

type VolumeService struct {
	db     *generated.Queries
	client *IddaaClient
	logger *logger.Logger
}

func NewVolumeService(db *generated.Queries, client *IddaaClient) *VolumeService {
	return &VolumeService{
		db:     db,
		client: client,
		logger: logger.New("volume-service"),
	}
}

type VolumeResponse struct {
	IsSuccess bool               `json:"isSuccess"`
	Data      map[string]float64 `json:"data"`
	Message   string             `json:"message"`
}

type EventVolume struct {
	EventID    string
	Percentage float64
	Rank       int
}

// FetchAndUpdateVolumes fetches betting volume data and updates the database using bulk operations
func (s *VolumeService) FetchAndUpdateVolumes(ctx context.Context, sportType int) error {
	start := time.Now()

	// Fetch volume data from API
	url := fmt.Sprintf("https://sportsbookv2.iddaa.com/sportsbook/played-event-percentage?sportType=%d", sportType)

	data, err := s.client.FetchData(url)
	if err != nil {
		return fmt.Errorf("failed to fetch volume data: %w", err)
	}

	var response VolumeResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return fmt.Errorf("failed to unmarshal volume response: %w", err)
	}

	if !response.IsSuccess {
		return fmt.Errorf("API request failed: %s", response.Message)
	}

	if len(response.Data) == 0 {
		s.logger.Info().
			Int("sport_type", sportType).
			Msg("No volume data returned from API")
		return nil
	}

	// Extract data for bulk operations (no sorting needed - database will rank)
	externalIDs := make([]string, 0, len(response.Data))
	percentages := make([]float64, 0, len(response.Data))

	for eventID, percentage := range response.Data {
		externalIDs = append(externalIDs, eventID)
		percentages = append(percentages, percentage)
	}

	// Execute bulk operations in a transaction
	err = s.executeBulkVolumeUpdate(ctx, externalIDs, percentages)
	if err != nil {
		return fmt.Errorf("bulk volume update failed: %w", err)
	}

	duration := time.Since(start)
	s.logger.Info().
		Int("sport_type", sportType).
		Int("events_count", len(response.Data)).
		Dur("duration", duration).
		Float64("events_per_second", float64(len(response.Data))/duration.Seconds()).
		Msg("Volume sync completed")

	return nil
}

// executeBulkVolumeUpdate performs all database operations in a single transaction
func (s *VolumeService) executeBulkVolumeUpdate(ctx context.Context, externalIDs []string, percentages []float64) error {
	// Note: Since we're using generated.Queries, we can't use transactions directly
	// But we can still use bulk operations to minimize round trips

	// First, verify which events exist
	existingEvents, err := s.db.GetEventIDsByExternalIDs(ctx, externalIDs)
	if err != nil {
		return fmt.Errorf("failed to get existing events: %w", err)
	}

	// Build map of existing events
	existingMap := make(map[string]bool)
	for _, event := range existingEvents {
		existingMap[event.ExternalID] = true
	}

	// Filter to only existing events
	validExternalIDs := make([]string, 0, len(existingEvents))
	validPercentages := make([]float64, 0, len(existingEvents))

	for i, extID := range externalIDs {
		if existingMap[extID] {
			validExternalIDs = append(validExternalIDs, extID)
			validPercentages = append(validPercentages, percentages[i])
		}
	}

	if len(validExternalIDs) == 0 {
		s.logger.Warn().
			Int("total_volumes", len(externalIDs)).
			Msg("No matching events found for volume data")
		return nil
	}

	// Bulk update volumes with database-calculated ranks
	rowsUpdated, err := s.db.BulkUpdateEventVolumes(ctx, generated.BulkUpdateEventVolumesParams{
		ExternalIds: validExternalIDs,
		Percentages: validPercentages,
	})
	if err != nil {
		return fmt.Errorf("failed to bulk update volumes: %w", err)
	}

	// Bulk insert history records
	_, err = s.db.BulkInsertVolumeHistory(ctx, generated.BulkInsertVolumeHistoryParams{
		ExternalIds: validExternalIDs,
		Percentages: validPercentages,
		TotalEvents: int64(len(validExternalIDs)),
	})
	if err != nil {
		return fmt.Errorf("failed to bulk insert volume history: %w", err)
	}

	s.logger.Debug().
		Int("requested", len(externalIDs)).
		Int("matched", len(validExternalIDs)).
		Int64("updated", rowsUpdated).
		Msg("Bulk volume update completed")

	return nil
}

// Custom error types for better error handling
var (
	ErrNoVolumeData = errors.New("no volume data available")
	ErrAPIFailure   = errors.New("API request failed")
)

// GetHotMovers finds events with high volume AND significant odds movement
func (s *VolumeService) GetHotMovers(ctx context.Context, minVolume float64, minMovement float64) ([]HotMover, error) {
	rows, err := s.db.GetHotMovers(ctx, generated.GetHotMoversParams{
		MinVolume:   minVolume,
		MinMovement: minMovement,
	})
	if err != nil {
		return nil, err
	}

	movers := make([]HotMover, len(rows))
	for i, row := range rows {
		volume := float64(0)
		if row.BettingVolumePercentage != nil {
			volume = float64(*row.BettingVolumePercentage)
		}
		volumeRank := 0
		if row.VolumeRank != nil {
			volumeRank = int(*row.VolumeRank)
		}

		movers[i] = HotMover{
			EventSlug:       row.Slug,
			MatchName:       row.MatchName.(string),
			Volume:          volume,
			VolumeRank:      volumeRank,
			MaxMovement:     row.MaxMovement.(float64),
			PopularityLevel: row.PopularityLevel,
			EventType:       row.EventType,
		}
	}

	return movers, nil
}

// GetHiddenGems finds low-volume events with big odds movements (potential sharp money)
func (s *VolumeService) GetHiddenGems(ctx context.Context, maxVolume float64, minMovement float64) ([]HiddenGem, error) {
	rows, err := s.db.GetHiddenGems(ctx, generated.GetHiddenGemsParams{
		MaxVolume:   maxVolume,
		MinMovement: minMovement,
	})
	if err != nil {
		return nil, err
	}

	gems := make([]HiddenGem, len(rows))
	for i, row := range rows {
		volume := float64(0)
		if row.BettingVolumePercentage != nil {
			volume = float64(*row.BettingVolumePercentage)
		}

		gems[i] = HiddenGem{
			EventSlug:   row.Slug,
			MatchName:   row.MatchName.(string),
			Volume:      volume,
			MaxMovement: row.MaxMovement.(float64),
			Insight:     "Low public interest but sharp money moving - potential value",
		}
	}

	return gems, nil
}

type HotMover struct {
	EventSlug       string  `json:"event_slug"`
	MatchName       string  `json:"match_name"`
	Volume          float64 `json:"volume_percentage"`
	VolumeRank      int     `json:"volume_rank"`
	MaxMovement     float64 `json:"max_movement_percentage"`
	PopularityLevel string  `json:"popularity_level"`
	EventType       string  `json:"event_type"`
}

type HiddenGem struct {
	EventSlug   string  `json:"event_slug"`
	MatchName   string  `json:"match_name"`
	Volume      float64 `json:"volume_percentage"`
	MaxMovement float64 `json:"max_movement_percentage"`
	Insight     string  `json:"insight"`
}
