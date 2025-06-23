package odds

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models/api"
)

// Handler handles odds-related requests
type Handler struct {
	queries *generated.Queries
	logger  *logger.Logger
}

// NewHandler creates a new odds handler
func NewHandler(queries *generated.Queries, log *logger.Logger) *Handler {
	return &Handler{
		queries: queries,
		logger:  log,
	}
}

// BigMovers handles the /api/odds/big-movers endpoint
func (h *Handler) BigMovers(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters with simple defaults
	hours := 24
	if hoursStr := r.URL.Query().Get("hours"); hoursStr != "" {
		if parsed, err := strconv.Atoi(hoursStr); err == nil && parsed >= 1 && parsed <= 168 {
			hours = parsed
		}
	}

	threshold := 50.0
	if thresholdStr := r.URL.Query().Get("threshold"); thresholdStr != "" {
		if parsed, err := strconv.ParseFloat(thresholdStr, 64); err == nil && parsed >= 0 {
			threshold = parsed
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Query database
	params := generated.GetRecentMovementsParams{
		SinceTime: pgtype.Timestamp{
			Time:  time.Now().Add(-time.Duration(hours) * time.Hour),
			Valid: true,
		},
		MinChangePercentage: threshold,
		LimitCount:          100,
	}

	dbMovements, err := h.queries.GetRecentMovements(ctx, params)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to query odds movements")
		// Return empty array on error
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]api.BigMoverResponse{}); err != nil {
			h.logger.Error().Err(err).Msg("Failed to encode empty response")
		}
		return
	}

	// Convert to API response
	movers := make([]api.BigMoverResponse, 0, len(dbMovements))

	for _, m := range dbMovements {
		// Skip invalid data
		if m.OddsValue <= 1.0 || (m.PreviousValue != nil && *m.PreviousValue <= 1.0) {
			continue
		}

		// Get change percentage - skip if nil or too small
		if m.ChangePercentage == nil || math.Abs(float64(*m.ChangePercentage)) < 5.0 {
			continue
		}

		// Convert pointers to values with defaults
		changePercentage := float64(*m.ChangePercentage)

		previousValue := m.OddsValue // default to current if nil
		if m.PreviousValue != nil {
			previousValue = *m.PreviousValue
		}

		multiplier := 1.0
		if m.Multiplier != nil {
			multiplier = *m.Multiplier
		}

		// Determine direction
		direction := "DRIFTING"
		if changePercentage < 0 {
			direction = "SHORTENING"
		}

		// Convert int32 pointers to int64 pointers for API response
		var homeScore, awayScore, minuteOfMatch *int64
		if m.HomeScore != nil {
			val := int64(*m.HomeScore)
			homeScore = &val
		}
		if m.AwayScore != nil {
			val := int64(*m.AwayScore)
			awayScore = &val
		}
		if m.MinuteOfMatch != nil {
			val := int64(*m.MinuteOfMatch)
			minuteOfMatch = &val
		}

		// Convert float32 to float64 for betting volume
		var bettingVolumePercent *float64
		if m.BettingVolumePercentage != nil {
			val := float64(*m.BettingVolumePercentage)
			bettingVolumePercent = &val
		}

		// Get event time
		eventTime := time.Now().Add(24 * time.Hour) // default
		if m.EventDate.Valid {
			eventTime = m.EventDate.Time
		}

		// Handle nullable strings
		leagueCountry := ""
		if m.LeagueCountry != nil {
			leagueCountry = *m.LeagueCountry
		}

		marketDescription := ""
		if m.MarketDescription != nil {
			marketDescription = *m.MarketDescription
		}

		homeTeamCountry := ""
		if m.HomeTeamCountry != nil {
			homeTeamCountry = *m.HomeTeamCountry
		}

		awayTeamCountry := ""
		if m.AwayTeamCountry != nil {
			awayTeamCountry = *m.AwayTeamCountry
		}

		isLive := false
		if m.IsLive != nil {
			isLive = *m.IsLive
		}

		mover := api.BigMoverResponse{
			EventSlug:            m.EventSlug,
			Match:                fmt.Sprintf("%s vs %s", m.HomeTeam, m.AwayTeam),
			Sport:                m.SportName,
			SportCode:            m.SportCode,
			League:               m.LeagueName,
			LeagueCountry:        leagueCountry,
			Market:               m.MarketName,
			MarketDescription:    marketDescription,
			Outcome:              m.Outcome,
			OpeningOdds:          previousValue,
			CurrentOdds:          m.OddsValue,
			ChangePercentage:     changePercentage,
			Multiplier:           multiplier,
			Direction:            direction,
			LastUpdated:          m.RecordedAt.Time,
			EventTime:            eventTime,
			EventStatus:          m.EventStatus,
			IsLive:               isLive,
			HomeScore:            homeScore,
			AwayScore:            awayScore,
			MinuteOfMatch:        minuteOfMatch,
			BettingVolumePercent: bettingVolumePercent,
			HomeTeamCountry:      homeTeamCountry,
			AwayTeamCountry:      awayTeamCountry,
		}

		movers = append(movers, mover)
	}

	h.logger.Info().
		Int("hours", hours).
		Float64("threshold", threshold).
		Int("count", len(movers)).
		Msg("Returning big movers")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(movers); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
