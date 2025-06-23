package leagues

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models/api"
)

type Handler struct {
	queries *generated.Queries
	logger  *logger.Logger
}

func NewHandler(queries *generated.Queries, logger *logger.Logger) *Handler {
	return &Handler{
		queries: queries,
		logger:  logger,
	}
}

type LeagueWithMapping struct {
	generated.League
	IsMapped bool `json:"is_mapped"`
}

// List handles GET /api/leagues
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	notMapped := r.URL.Query().Get("not_mapped") == "true"

	var leagues []generated.League
	var err error

	if notMapped {
		// Get leagues that don't have mappings
		leagues, err = h.queries.ListUnmappedLeagues(ctx)
	} else {
		// Get all leagues
		leagues, err = h.queries.ListLeagues(ctx)
	}

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to fetch leagues")
		http.Error(w, "Failed to fetch leagues", http.StatusInternalServerError)
		return
	}

	// Convert to response format with mapping status
	var response []LeagueWithMapping
	for _, league := range leagues {
		leagueWithMapping := LeagueWithMapping{
			League:   league,
			IsMapped: league.ApiFootballID != nil,
		}
		response = append(response, leagueWithMapping)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Data:    response,
		Meta: map[string]any{
			"total": len(response),
		},
	}); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode leagues response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// UpdateMapping handles PUT /api/leagues/{id}/mapping
func (h *Handler) UpdateMapping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract league ID from path
	leagueIDStr := r.URL.Path[len("/api/leagues/"):]
	if idx := len(leagueIDStr) - len("/mapping"); idx > 0 && leagueIDStr[idx:] == "/mapping" {
		leagueIDStr = leagueIDStr[:idx]
	}

	leagueID, err := strconv.Atoi(leagueIDStr)
	if err != nil {
		http.Error(w, "Invalid league ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req struct {
		ApiFootballID int32 `json:"api_football_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update league with API Football ID
	err = h.queries.UpdateLeagueApiFootballID(ctx, generated.UpdateLeagueApiFootballIDParams{
		ID:            int32(leagueID),
		ApiFootballID: &req.ApiFootballID,
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to update league mapping")
		http.Error(w, "Failed to update league mapping", http.StatusInternalServerError)
		return
	}

	// Also create/update league mapping record
	_, err = h.queries.UpsertLeagueMapping(ctx, generated.UpsertLeagueMappingParams{
		InternalLeagueID:    int32(leagueID),
		FootballApiLeagueID: req.ApiFootballID,
		Confidence:          1.0, // Set a default confidence value, adjust as needed
		MappingMethod:       "manual",
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create league mapping record")
		// Don't fail the request, just log the error
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Message: "League mapping updated successfully",
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
