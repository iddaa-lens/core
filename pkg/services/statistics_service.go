package services

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models"
)

type StatisticsService struct {
	db     *database.Queries
	client *IddaaClient
	logger *logger.Logger
}

func NewStatisticsService(db *database.Queries, client *IddaaClient) *StatisticsService {
	return &StatisticsService{
		db:     db,
		client: client,
		logger: logger.New("statistics-service"),
	}
}

func (s *StatisticsService) SyncEventStatistics(ctx context.Context, sportID int, searchDate string) error {
	log := logger.WithContext(ctx, "statistics-sync")

	log.Info().
		Int("sport_id", sportID).
		Str("search_date", searchDate).
		Str("action", "sync_start").
		Msg("Starting statistics sync")

	stats, err := s.client.GetEventStatistics(sportID, searchDate)
	if err != nil {
		log.Error().
			Err(err).
			Int("sport_id", sportID).
			Str("search_date", searchDate).
			Str("action", "api_failed").
			Msg("Failed to fetch event statistics")
		return fmt.Errorf("failed to fetch event statistics: %w", err)
	}

	log.Info().
		Int("events_count", len(stats)).
		Str("action", "api_response").
		Msg("Fetched statistics from API")

	// Keep track of successes and failures
	successCount := 0
	errorCount := 0

	for _, stat := range stats {
		if err := s.saveEventStatistics(ctx, stat); err != nil {
			errorCount++
			log.Error().
				Err(err).
				Int("event_id", stat.EventID).
				Str("action", "save_failed").
				Msg("Failed to save event statistics")
			continue
		}
		successCount++
	}

	log.Info().
		Int("success_count", successCount).
		Int("error_count", errorCount).
		Str("action", "sync_complete").
		Msg("Statistics sync completed")
	return nil
}

func (s *StatisticsService) saveEventStatistics(ctx context.Context, stat models.IddaaEventStatistics) error {
	// First, try to find the event by external ID
	event, err := s.db.GetEventByExternalID(ctx, strconv.Itoa(stat.EventID))
	if err != nil {
		// Event not found in our database, skip
		s.logger.Debug().
			Int("event_id", stat.EventID).
			Str("action", "event_not_found").
			Msg("Event not found in database, skipping statistics")
		return nil
	}

	// Update event with live data
	_, err = s.db.UpdateEventLiveData(ctx, database.UpdateEventLiveDataParams{
		ID:            event.ID,
		IsLive:        pgtype.Bool{Bool: stat.IsLive, Valid: true},
		Status:        strconv.Itoa(stat.Status),
		HomeScore:     pgtype.Int4{Int32: int32(stat.HomeScore), Valid: true},
		AwayScore:     pgtype.Int4{Int32: int32(stat.AwayScore), Valid: true},
		MinuteOfMatch: pgtype.Int4{Int32: int32(stat.MinuteOfMatch), Valid: true},
		Half:          pgtype.Int4{Int32: int32(stat.Half), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update event live data: %w", err)
	}

	// Save match statistics if available
	if stat.HasStatistics {
		err = s.saveMatchStatistics(ctx, event.ID, stat.Statistics)
		if err != nil {
			s.logger.Error().
				Err(err).
				Int("event_id", stat.EventID).
				Str("action", "match_stats_failed").
				Msg("Failed to save match statistics")
		}
	}

	// Save match events
	for _, matchEvent := range stat.Events {
		err = s.saveMatchEvent(ctx, event.ID, matchEvent)
		if err != nil {
			s.logger.Error().
				Err(err).
				Int("event_id", stat.EventID).
				Str("action", "match_event_failed").
				Msg("Failed to save match event")
		}
	}

	return nil
}

func (s *StatisticsService) saveMatchStatistics(ctx context.Context, eventID int32, stats models.IddaaMatchStatistics) error {
	// Upsert home team statistics
	_, err := s.db.UpsertMatchStatistics(ctx, database.UpsertMatchStatisticsParams{
		EventID:       pgtype.Int4{Int32: eventID, Valid: true},
		IsHome:        true,
		Shots:         pgtype.Int4{Int32: int32(stats.HomeStats.Shots), Valid: true},
		ShotsOnTarget: pgtype.Int4{Int32: int32(stats.HomeStats.ShotsOnTarget), Valid: true},
		Possession:    pgtype.Int4{Int32: int32(stats.HomeStats.Possession), Valid: true},
		Corners:       pgtype.Int4{Int32: int32(stats.HomeStats.Corners), Valid: true},
		YellowCards:   pgtype.Int4{Int32: int32(stats.HomeStats.YellowCards), Valid: true},
		RedCards:      pgtype.Int4{Int32: int32(stats.HomeStats.RedCards), Valid: true},
		Fouls:         pgtype.Int4{Int32: int32(stats.HomeStats.Fouls), Valid: true},
		Offsides:      pgtype.Int4{Int32: int32(stats.HomeStats.Offsides), Valid: true},
		FreeKicks:     pgtype.Int4{Int32: int32(stats.HomeStats.FreeKicks), Valid: true},
		ThrowIns:      pgtype.Int4{Int32: int32(stats.HomeStats.ThrowIns), Valid: true},
		GoalKicks:     pgtype.Int4{Int32: int32(stats.HomeStats.GoalKicks), Valid: true},
		Saves:         pgtype.Int4{Int32: int32(stats.HomeStats.Saves), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to save home team statistics: %w", err)
	}

	// Upsert away team statistics
	_, err = s.db.UpsertMatchStatistics(ctx, database.UpsertMatchStatisticsParams{
		EventID:       pgtype.Int4{Int32: eventID, Valid: true},
		IsHome:        false,
		Shots:         pgtype.Int4{Int32: int32(stats.AwayStats.Shots), Valid: true},
		ShotsOnTarget: pgtype.Int4{Int32: int32(stats.AwayStats.ShotsOnTarget), Valid: true},
		Possession:    pgtype.Int4{Int32: int32(stats.AwayStats.Possession), Valid: true},
		Corners:       pgtype.Int4{Int32: int32(stats.AwayStats.Corners), Valid: true},
		YellowCards:   pgtype.Int4{Int32: int32(stats.AwayStats.YellowCards), Valid: true},
		RedCards:      pgtype.Int4{Int32: int32(stats.AwayStats.RedCards), Valid: true},
		Fouls:         pgtype.Int4{Int32: int32(stats.AwayStats.Fouls), Valid: true},
		Offsides:      pgtype.Int4{Int32: int32(stats.AwayStats.Offsides), Valid: true},
		FreeKicks:     pgtype.Int4{Int32: int32(stats.AwayStats.FreeKicks), Valid: true},
		ThrowIns:      pgtype.Int4{Int32: int32(stats.AwayStats.ThrowIns), Valid: true},
		GoalKicks:     pgtype.Int4{Int32: int32(stats.AwayStats.GoalKicks), Valid: true},
		Saves:         pgtype.Int4{Int32: int32(stats.AwayStats.Saves), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to save away team statistics: %w", err)
	}

	return nil
}

func (s *StatisticsService) saveMatchEvent(ctx context.Context, eventID int32, matchEvent models.IddaaMatchEvent) error {
	_, err := s.db.CreateMatchEvent(ctx, database.CreateMatchEventParams{
		EventID:     pgtype.Int4{Int32: eventID, Valid: true},
		Minute:      int32(matchEvent.Minute),
		EventType:   matchEvent.EventType,
		Team:        matchEvent.Team,
		Player:      pgtype.Text{String: matchEvent.Player, Valid: matchEvent.Player != ""},
		Description: matchEvent.Description,
		IsHome:      matchEvent.IsHome,
	})
	if err != nil {
		return fmt.Errorf("failed to create match event: %w", err)
	}

	return nil
}

// GetLiveEvents returns events that are currently live
func (s *StatisticsService) GetLiveEvents(ctx context.Context) ([]LiveEvent, error) {
	rows, err := s.db.GetLiveEvents(ctx)
	if err != nil {
		return nil, err
	}

	events := make([]LiveEvent, len(rows))
	for i, row := range rows {
		events[i] = LiveEvent{
			EventSlug:     row.Slug,
			HomeTeam:      row.HomeTeam,
			AwayTeam:      row.AwayTeam,
			HomeScore:     int(row.HomeScore.Int32),
			AwayScore:     int(row.AwayScore.Int32),
			MinuteOfMatch: int(row.MinuteOfMatch.Int32),
			Half:          int(row.Half.Int32),
			Status:        row.Status,
		}
	}

	return events, nil
}

type LiveEvent struct {
	EventSlug     string `json:"event_slug"`
	HomeTeam      string `json:"home_team"`
	AwayTeam      string `json:"away_team"`
	HomeScore     int    `json:"home_score"`
	AwayScore     int    `json:"away_score"`
	MinuteOfMatch int    `json:"minute_of_match"`
	Half          int    `json:"half"`
	Status        string `json:"status"`
}
