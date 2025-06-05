package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models"
	"github.com/jackc/pgx/v5/pgtype"
)

type SportService struct {
	db     *database.Queries
	client *IddaaClient
	logger *logger.Logger
}

func NewSportService(db *database.Queries, client *IddaaClient) *SportService {
	return &SportService{
		db:     db,
		client: client,
		logger: logger.New("sports-service"),
	}
}

func (s *SportService) SyncSports(ctx context.Context) error {
	log := logger.WithContext(ctx, "sports-sync")

	log.Info().
		Str("action", "sync_start").
		Msg("Starting sports sync")

	resp, err := s.client.GetSportInfo()
	if err != nil {
		return fmt.Errorf("failed to fetch sport info: %w", err)
	}

	log.Info().
		Str("action", "api_response").
		Int("sports_count", len(resp.Data)).
		Msg("Fetched sports from API")

	successCount := 0
	errorCount := 0

	for _, sport := range resp.Data {
		if err := s.saveSport(ctx, sport); err != nil {
			errorCount++
			log.Error().
				Err(err).
				Int("sport_id", sport.SportID).
				Str("action", "save_failed").
				Msg("Failed to save sport")
			continue
		}
		successCount++
	}

	log.Info().
		Str("action", "sync_complete").
		Int("success_count", successCount).
		Int("error_count", errorCount).
		Msg("Sports sync completed")
	return nil
}

func (s *SportService) saveSport(ctx context.Context, sport models.IddaaSportInfo) error {
	// Map sport IDs to names and codes
	sportMapping := map[int]struct {
		name string
		code string
	}{
		1:  {name: "Football", code: "FOOTBALL"},
		2:  {name: "Basketball", code: "BASKETBALL"},
		4:  {name: "Ice Hockey", code: "ICE_HOCKEY"},
		5:  {name: "Tennis", code: "TENNIS"},
		6:  {name: "Handball", code: "HANDBALL"},
		11: {name: "Formula 1", code: "FORMULA1"},
		23: {name: "Other", code: "OTHER"},
	}

	sportInfo, exists := sportMapping[sport.SportID]
	if !exists {
		s.logger.Warn().
			Int("sport_id", sport.SportID).
			Str("action", "unknown_sport").
			Msg("Unknown sport ID, skipping")
		return nil
	}

	params := database.UpsertSportParams{
		ID:                int32(sport.SportID),
		Name:              sportInfo.name,
		Code:              sportInfo.code,
		Slug:              generateSlug(sportInfo.name),
		LiveCount:         pgtype.Int4{Int32: int32(sport.LiveCount), Valid: true},
		UpcomingCount:     pgtype.Int4{Int32: int32(sport.UpcomingCount), Valid: true},
		EventsCount:       pgtype.Int4{Int32: int32(sport.EventsCount), Valid: true},
		OddsCount:         pgtype.Int4{Int32: int32(sport.OddsCount), Valid: true},
		HasResults:        pgtype.Bool{Bool: sport.HasResults, Valid: true},
		HasKingOdd:        pgtype.Bool{Bool: sport.HasKingOdd, Valid: true},
		HasDigitalContent: pgtype.Bool{Bool: sport.HasDigitalContent, Valid: true},
	}

	_, err := s.db.UpsertSport(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to upsert sport: %w", err)
	}

	s.logger.Debug().
		Int("sport_id", sport.SportID).
		Str("sport_name", sportInfo.name).
		Int("live_count", sport.LiveCount).
		Int("upcoming_count", sport.UpcomingCount).
		Int("events_count", sport.EventsCount).
		Str("action", "sport_updated").
		Msg("Updated sport data")

	return nil
}

func (s *SportService) GetSport(ctx context.Context, sportID int32) (*database.Sport, error) {
	sport, err := s.db.GetSport(ctx, sportID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sport: %w", err)
	}
	return &sport, nil
}

func (s *SportService) ListSports(ctx context.Context) ([]database.Sport, error) {
	sports, err := s.db.ListSports(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sports: %w", err)
	}
	return sports, nil
}

// generateSlug creates a URL-friendly slug from a string
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	return slug
}
