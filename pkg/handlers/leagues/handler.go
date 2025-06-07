package leagues

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models/api"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	queries *database.Queries
	logger  *logger.Logger
}

func NewHandler(queries *database.Queries, logger *logger.Logger) *Handler {
	return &Handler{
		queries: queries,
		logger:  logger,
	}
}

type LeagueWithMapping struct {
	database.League
	IsMapped bool `json:"is_mapped"`
}

// List handles GET /api/leagues
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	notMapped := r.URL.Query().Get("not_mapped") == "true"
	
	var leagues []database.League
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
			IsMapped: league.ApiFootballID.Valid,
		}
		response = append(response, leagueWithMapping)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Data:    response,
		Meta: map[string]interface{}{
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
	err = h.queries.UpdateLeagueApiFootballID(ctx, database.UpdateLeagueApiFootballIDParams{
		ID: int32(leagueID),
		ApiFootballID: pgtype.Int4{
			Int32: req.ApiFootballID,
			Valid: true,
		},
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to update league mapping")
		http.Error(w, "Failed to update league mapping", http.StatusInternalServerError)
		return
	}

	// Also create/update league mapping record
	_, err = h.queries.UpsertLeagueMapping(ctx, database.UpsertLeagueMappingParams{
		InternalLeagueID:     int32(leagueID),
		FootballApiLeagueID:  req.ApiFootballID,
		Confidence: pgtype.Numeric{
			Int: nil, // Will be set to a default value by the database
			Exp: 0,
			NaN: false,
			Valid: true,
		},
		MappingMethod: "manual",
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create league mapping record")
		// Don't fail the request, just log the error
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Message: "League mapping updated successfully",
	})
}