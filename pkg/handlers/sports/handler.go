package sports

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models/api"
)

// Handler handles sports-related requests
type Handler struct {
	queries *generated.Queries
	logger  *logger.Logger
}

// NewHandler creates a new sports handler
func NewHandler(queries *generated.Queries, log *logger.Logger) *Handler {
	return &Handler{
		queries: queries,
		logger:  log,
	}
}

// List handles the /api/v1/sports endpoint
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Query database for all sports
	dbSports, err := h.queries.ListSports(ctx)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("action", "query_sports_failed").
			Msg("Failed to query sports")

		http.Error(w, "Failed to query sports", http.StatusInternalServerError)
		return
	}

	// Convert database results to API response
	sports := h.convertSportsToResponse(dbSports)

	// Log response info
	h.logger.Info().
		Str("action", "sports_response").
		Int("count", len(sports)).
		Msg("Returning sports list")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sports); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// convertSportsToResponse converts database sports to API response format
func (h *Handler) convertSportsToResponse(dbSports []generated.Sport) []api.SportResponse {
	var sports []api.SportResponse
	for _, sport := range dbSports {
		// Handle nullable fields with default values
		liveCount := int32(0)
		if sport.LiveCount != nil {
			liveCount = *sport.LiveCount
		}

		upcomingCount := int32(0)
		if sport.UpcomingCount != nil {
			upcomingCount = *sport.UpcomingCount
		}

		eventsCount := int32(0)
		if sport.EventsCount != nil {
			eventsCount = *sport.EventsCount
		}

		oddsCount := int32(0)
		if sport.OddsCount != nil {
			oddsCount = *sport.OddsCount
		}

		hasResults := false
		if sport.HasResults != nil {
			hasResults = *sport.HasResults
		}

		hasKingOdd := false
		if sport.HasKingOdd != nil {
			hasKingOdd = *sport.HasKingOdd
		}

		hasDigitalContent := false
		if sport.HasDigitalContent != nil {
			hasDigitalContent = *sport.HasDigitalContent
		}

		sportResponse := api.SportResponse{
			ID:                sport.ID,
			Name:              sport.Name,
			Code:              sport.Code,
			Slug:              sport.Slug,
			LiveCount:         liveCount,
			UpcomingCount:     upcomingCount,
			EventsCount:       eventsCount,
			OddsCount:         oddsCount,
			HasResults:        hasResults,
			HasKingOdd:        hasKingOdd,
			HasDigitalContent: hasDigitalContent,
		}
		sports = append(sports, sportResponse)
	}
	return sports
}
