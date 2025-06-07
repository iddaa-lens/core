package odds

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models/api"
	"github.com/jackc/pgx/v5/pgtype"
)

// Handler handles odds-related requests
type Handler struct {
	queries *database.Queries
	logger  *logger.Logger
}

// NewHandler creates a new odds handler
func NewHandler(queries *database.Queries, log *logger.Logger) *Handler {
	return &Handler{
		queries: queries,
		logger:  log,
	}
}

// BigMovers handles the /api/odds/big-movers endpoint
func (h *Handler) BigMovers(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	hoursStr := r.URL.Query().Get("hours")
	if hoursStr == "" {
		hoursStr = "24"
	}
	hours, err := strconv.Atoi(hoursStr)
	if err != nil || hours < 1 || hours > 168 {
		hours = 24
	}

	thresholdStr := r.URL.Query().Get("threshold")
	if thresholdStr == "" {
		thresholdStr = "50"
	}
	threshold, err := strconv.ParseFloat(thresholdStr, 64)
	if err != nil || threshold < 0 {
		threshold = 50
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Calculate time range
	sinceTime := pgtype.Timestamp{
		Time:  time.Now().Add(-time.Duration(hours) * time.Hour),
		Valid: true,
	}

	// Convert threshold to numeric
	minChangePercentage := pgtype.Numeric{}
	err = minChangePercentage.Scan(fmt.Sprintf("%.2f", threshold))
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to convert threshold")
		http.Error(w, "Invalid threshold parameter", http.StatusBadRequest)
		return
	}

	// Query database for recent movements
	params := database.GetRecentMovementsParams{
		SinceTime:           sinceTime,
		MinChangePercentage: minChangePercentage,
		LimitCount:          100, // Limit to 100 results
	}

	dbMovements, err := h.queries.GetRecentMovements(ctx, params)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("action", "query_movements_failed").
			Msg("Failed to query odds movements")

		// Return empty data when database query fails
		h.logger.Error().
			Err(err).
			Str("action", "returning_empty").
			Msg("Returning empty data due to database query failure")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]api.BigMoverResponse{})
		return
	}

	// Convert database results to API response
	var movers []api.BigMoverResponse
	for _, movement := range dbMovements {
		// Convert pgtype.Numeric to float64 safely
		var oddsValue, previousValue, changePercentage, multiplier float64

		if movement.OddsValue.Valid {
			if val, err := movement.OddsValue.Value(); err == nil {
				if str, ok := val.(string); ok {
					if parsed, err := strconv.ParseFloat(str, 64); err == nil {
						oddsValue = parsed
					}
				}
			}
		}
		if movement.PreviousValue.Valid {
			if val, err := movement.PreviousValue.Value(); err == nil {
				if str, ok := val.(string); ok {
					if parsed, err := strconv.ParseFloat(str, 64); err == nil {
						previousValue = parsed
					}
				}
			}
		}
		if movement.ChangePercentage.Valid {
			if val, err := movement.ChangePercentage.Value(); err == nil {
				if str, ok := val.(string); ok {
					if parsed, err := strconv.ParseFloat(str, 64); err == nil {
						changePercentage = parsed
					}
				}
			}
		}
		if movement.Multiplier.Valid {
			if val, err := movement.Multiplier.Value(); err == nil {
				if str, ok := val.(string); ok {
					if parsed, err := strconv.ParseFloat(str, 64); err == nil {
						multiplier = parsed
					}
				}
			}
		}

		// Skip invalid odds (1.0 or below are usually placeholders/suspended)
		if oddsValue <= 1.0 || previousValue <= 1.0 {
			continue
		}

		// Skip movements that are too small to be meaningful
		if math.Abs(changePercentage) < 5.0 {
			continue
		}

		// Determine direction
		direction := "DRIFTING"
		if changePercentage < 0 {
			direction = "SHORTENING"
		}

		// Create match string
		match := fmt.Sprintf("%s vs %s", movement.HomeTeam, movement.AwayTeam)

		// Get event time
		eventTime := time.Now().Add(24 * time.Hour) // Default
		if movement.EventDate.Valid {
			eventTime = movement.EventDate.Time
		}

		mover := api.BigMoverResponse{
			EventSlug:            movement.EventSlug,
			Match:                match,
			Sport:                movement.SportName,
			SportCode:            movement.SportCode,
			League:               movement.LeagueName,
			LeagueCountry:        h.getStringFromText(movement.LeagueCountry),
			Market:               movement.MarketName,
			MarketDescription:    h.getStringFromText(movement.MarketDescription),
			Outcome:              movement.Outcome,
			OpeningOdds:          previousValue,
			CurrentOdds:          oddsValue,
			ChangePercentage:     changePercentage,
			Multiplier:           multiplier,
			Direction:            direction,
			LastUpdated:          movement.RecordedAt.Time,
			EventTime:            eventTime,
			EventStatus:          movement.EventStatus,
			IsLive:               movement.IsLive.Bool,
			HomeScore:            h.getInt32Ptr(movement.HomeScore),
			AwayScore:            h.getInt32Ptr(movement.AwayScore),
			MinuteOfMatch:        h.getInt32Ptr(movement.MinuteOfMatch),
			BettingVolumePercent: h.getFloat64Ptr(movement.BettingVolumePercentage),
			HomeTeamCountry:      h.getStringFromText(movement.HomeTeamCountry),
			AwayTeamCountry:      h.getStringFromText(movement.AwayTeamCountry),
		}

		movers = append(movers, mover)
	}

	// Log if no data found
	if len(movers) == 0 {
		h.logger.Info().
			Str("action", "no_data_found").
			Msg("No recent movements found in the specified time period")
	}

	// Log response info
	h.logger.Info().
		Str("action", "big_movers_response").
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

// Helper function to get string from pgtype.Text
func (h *Handler) getStringFromText(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

// Helper function to get int32 pointer from pgtype.Int4
func (h *Handler) getInt32Ptr(i pgtype.Int4) *int32 {
	if i.Valid {
		return &i.Int32
	}
	return nil
}

// Helper function to get float64 pointer from pgtype.Numeric
func (h *Handler) getFloat64Ptr(n pgtype.Numeric) *float64 {
	if n.Valid {
		if val, err := n.Value(); err == nil {
			if str, ok := val.(string); ok {
				if parsed, err := strconv.ParseFloat(str, 64); err == nil {
					return &parsed
				}
			}
		}
	}
	return nil
}
