package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/iddaa-lens/core/pkg/database"
	"github.com/jackc/pgx/v5/pgtype"
)

type VolumeService struct {
	db     *database.Queries
	client *IddaaClient
}

func NewVolumeService(db *database.Queries, client *IddaaClient) *VolumeService {
	return &VolumeService{
		db:     db,
		client: client,
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

// FetchAndUpdateVolumes fetches betting volume data and updates the database
func (s *VolumeService) FetchAndUpdateVolumes(ctx context.Context, sportType int) error {
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

	// Convert to sorted slice for ranking
	volumes := make([]EventVolume, 0, len(response.Data))
	for eventID, percentage := range response.Data {
		volumes = append(volumes, EventVolume{
			EventID:    eventID,
			Percentage: percentage,
		})
	}

	// Sort by percentage descending
	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Percentage > volumes[j].Percentage
	})

	// Assign ranks
	for i := range volumes {
		volumes[i].Rank = i + 1
	}

	// Update database
	totalEvents := len(volumes)
	for _, vol := range volumes {
		if err := s.updateEventVolume(ctx, vol, totalEvents); err != nil {
			// Log error but continue with other events
			fmt.Printf("Failed to update volume for event %s: %v\n", vol.EventID, err)
			continue
		}
	}

	return nil
}

func (s *VolumeService) updateEventVolume(ctx context.Context, vol EventVolume, totalEvents int) error {
	// First, check if event exists
	event, err := s.db.GetEventByExternalIDSimple(ctx, vol.EventID)
	if err != nil {
		return fmt.Errorf("event %s not found: %w", vol.EventID, err)
	}

	// Create volume percentage numeric value
	var volumePercentageNumeric pgtype.Numeric
	volumeStr := fmt.Sprintf("%.2f", vol.Percentage)
	if err := volumePercentageNumeric.Scan(volumeStr); err != nil {
		return fmt.Errorf("failed to convert volume percentage %.2f: %w", vol.Percentage, err)
	}

	// Update event with volume data
	_, err = s.db.UpdateEventVolume(ctx, database.UpdateEventVolumeParams{
		ID:                      event.ID,
		BettingVolumePercentage: volumePercentageNumeric,
		VolumeRank:              pgtype.Int4{Int32: int32(vol.Rank), Valid: true},
		VolumeUpdatedAt:         pgtype.Timestamp{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update event volume: %w", err)
	}

	// Record in history with the same volume percentage value
	_, err = s.db.CreateVolumeHistory(ctx, database.CreateVolumeHistoryParams{
		EventID:            pgtype.Int4{Int32: event.ID, Valid: true},
		VolumePercentage:   volumePercentageNumeric,
		RankPosition:       pgtype.Int4{Int32: int32(vol.Rank), Valid: true},
		TotalEventsTracked: pgtype.Int4{Int32: int32(totalEvents), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create volume history: %w", err)
	}

	return nil
}

// GetHotMovers finds events with high volume AND significant odds movement
func (s *VolumeService) GetHotMovers(ctx context.Context, minVolume float64, minMovement float64) ([]HotMover, error) {
	rows, err := s.db.GetHotMovers(ctx, database.GetHotMoversParams{
		MinVolume:   func() pgtype.Numeric { n := pgtype.Numeric{}; _ = n.Scan(minVolume); return n }(),
		MinMovement: func() pgtype.Numeric { n := pgtype.Numeric{}; _ = n.Scan(minMovement); return n }(),
	})
	if err != nil {
		return nil, err
	}

	movers := make([]HotMover, len(rows))
	for i, row := range rows {
		volumeFloat, _ := row.BettingVolumePercentage.Float64Value()
		movers[i] = HotMover{
			EventSlug:       row.Slug,
			MatchName:       row.MatchName.(string),
			Volume:          volumeFloat.Float64,
			VolumeRank:      int(row.VolumeRank.Int32),
			MaxMovement:     row.MaxMovement.(float64),
			PopularityLevel: row.PopularityLevel,
			EventType:       row.EventType,
		}
	}

	return movers, nil
}

// GetHiddenGems finds low-volume events with big odds movements (potential sharp money)
func (s *VolumeService) GetHiddenGems(ctx context.Context, maxVolume float64, minMovement float64) ([]HiddenGem, error) {
	rows, err := s.db.GetHiddenGems(ctx, database.GetHiddenGemsParams{
		MaxVolume:   func() pgtype.Numeric { n := pgtype.Numeric{}; _ = n.Scan(maxVolume); return n }(),
		MinMovement: func() pgtype.Numeric { n := pgtype.Numeric{}; _ = n.Scan(minMovement); return n }(),
	})
	if err != nil {
		return nil, err
	}

	gems := make([]HiddenGem, len(rows))
	for i, row := range rows {
		gems[i] = HiddenGem{
			EventSlug:   row.Slug,
			MatchName:   row.MatchName.(string),
			Volume:      func() float64 { v, _ := row.BettingVolumePercentage.Float64Value(); return v.Float64 }(),
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
