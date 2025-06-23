package services

import (
	"context"
	"fmt"
	"strconv"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models"
)

type StatisticsService struct {
	db     *generated.Queries
	client *IddaaClient
	logger *logger.Logger
}

func NewStatisticsService(db *generated.Queries, client *IddaaClient) *StatisticsService {
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
	isLive := stat.IsLive
	homeScore := int32(stat.HomeScore)
	awayScore := int32(stat.AwayScore)
	minuteOfMatch := int32(stat.MinuteOfMatch)
	half := int32(stat.Half)

	_, err = s.db.UpdateEventLiveData(ctx, generated.UpdateEventLiveDataParams{
		ID:            event.ID,
		IsLive:        &isLive,
		Status:        strconv.Itoa(stat.Status),
		HomeScore:     &homeScore,
		AwayScore:     &awayScore,
		MinuteOfMatch: &minuteOfMatch,
		Half:          &half,
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
	shots := int32(stats.HomeStats.Shots)
	shotsOnTarget := int32(stats.HomeStats.ShotsOnTarget)
	possession := int32(stats.HomeStats.Possession)
	corners := int32(stats.HomeStats.Corners)
	yellowCards := int32(stats.HomeStats.YellowCards)
	redCards := int32(stats.HomeStats.RedCards)
	fouls := int32(stats.HomeStats.Fouls)
	offsides := int32(stats.HomeStats.Offsides)
	freeKicks := int32(stats.HomeStats.FreeKicks)
	throwIns := int32(stats.HomeStats.ThrowIns)
	goalKicks := int32(stats.HomeStats.GoalKicks)
	saves := int32(stats.HomeStats.Saves)

	_, err := s.db.UpsertMatchStatistics(ctx, generated.UpsertMatchStatisticsParams{
		EventID:       &eventID,
		IsHome:        true,
		Shots:         &shots,
		ShotsOnTarget: &shotsOnTarget,
		Possession:    &possession,
		Corners:       &corners,
		YellowCards:   &yellowCards,
		RedCards:      &redCards,
		Fouls:         &fouls,
		Offsides:      &offsides,
		FreeKicks:     &freeKicks,
		ThrowIns:      &throwIns,
		GoalKicks:     &goalKicks,
		Saves:         &saves,
	})
	if err != nil {
		return fmt.Errorf("failed to save home team statistics: %w", err)
	}

	// Upsert away team statistics
	awayShots := int32(stats.AwayStats.Shots)
	awayShotsOnTarget := int32(stats.AwayStats.ShotsOnTarget)
	awayPossession := int32(stats.AwayStats.Possession)
	awayCorners := int32(stats.AwayStats.Corners)
	awayYellowCards := int32(stats.AwayStats.YellowCards)
	awayRedCards := int32(stats.AwayStats.RedCards)
	awayFouls := int32(stats.AwayStats.Fouls)
	awayOffsides := int32(stats.AwayStats.Offsides)
	awayFreeKicks := int32(stats.AwayStats.FreeKicks)
	awayThrowIns := int32(stats.AwayStats.ThrowIns)
	awayGoalKicks := int32(stats.AwayStats.GoalKicks)
	awaySaves := int32(stats.AwayStats.Saves)

	_, err = s.db.UpsertMatchStatistics(ctx, generated.UpsertMatchStatisticsParams{
		EventID:       &eventID,
		IsHome:        false,
		Shots:         &awayShots,
		ShotsOnTarget: &awayShotsOnTarget,
		Possession:    &awayPossession,
		Corners:       &awayCorners,
		YellowCards:   &awayYellowCards,
		RedCards:      &awayRedCards,
		Fouls:         &awayFouls,
		Offsides:      &awayOffsides,
		FreeKicks:     &awayFreeKicks,
		ThrowIns:      &awayThrowIns,
		GoalKicks:     &awayGoalKicks,
		Saves:         &awaySaves,
	})
	if err != nil {
		return fmt.Errorf("failed to save away team statistics: %w", err)
	}

	return nil
}

func (s *StatisticsService) saveMatchEvent(ctx context.Context, eventID int32, matchEvent models.IddaaMatchEvent) error {
	var player *string
	if matchEvent.Player != "" {
		player = &matchEvent.Player
	}

	_, err := s.db.CreateMatchEvent(ctx, generated.CreateMatchEventParams{
		EventID:     &eventID,
		Minute:      int32(matchEvent.Minute),
		EventType:   matchEvent.EventType,
		Team:        matchEvent.Team,
		Player:      player,
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
		homeScore := 0
		if row.HomeScore != nil {
			homeScore = int(*row.HomeScore)
		}
		awayScore := 0
		if row.AwayScore != nil {
			awayScore = int(*row.AwayScore)
		}
		minuteOfMatch := 0
		if row.MinuteOfMatch != nil {
			minuteOfMatch = int(*row.MinuteOfMatch)
		}
		half := 0
		if row.Half != nil {
			half = int(*row.Half)
		}

		events[i] = LiveEvent{
			EventSlug:     row.Slug,
			HomeTeam:      row.HomeTeam,
			AwayTeam:      row.AwayTeam,
			HomeScore:     homeScore,
			AwayScore:     awayScore,
			MinuteOfMatch: minuteOfMatch,
			Half:          half,
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
