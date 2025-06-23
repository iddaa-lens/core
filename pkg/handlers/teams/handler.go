package teams

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

type TeamWithMapping struct {
	generated.Team
	IsMapped bool `json:"is_mapped"`
}

// List handles GET /api/teams
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	notMapped := r.URL.Query().Get("not_mapped") == "true"

	var teams []generated.Team
	var err error

	if notMapped {
		// Get teams that don't have mappings
		teams, err = h.queries.ListUnmappedTeams(ctx)
	} else {
		// Get all teams (you may want to implement pagination here)
		emptyString := ""
		teams, err = h.queries.SearchTeams(ctx, generated.SearchTeamsParams{
			SearchTerm: &emptyString,
			LimitCount: 1000, // Set a reasonable default limit
		})
	}

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to fetch teams")
		http.Error(w, "Failed to fetch teams", http.StatusInternalServerError)
		return
	}

	// Convert to response format with mapping status
	var response []TeamWithMapping
	for _, team := range teams {
		teamWithMapping := TeamWithMapping{
			Team:     team,
			IsMapped: team.ApiFootballID != nil,
		}
		response = append(response, teamWithMapping)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Data:    response,
		Meta: map[string]any{
			"total": len(response),
		},
	}); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode teams response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// UpdateMapping handles PUT /api/teams/{id}/mapping
func (h *Handler) UpdateMapping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract team ID from path
	teamIDStr := r.URL.Path[len("/api/teams/"):]
	if idx := len(teamIDStr) - len("/mapping"); idx > 0 && teamIDStr[idx:] == "/mapping" {
		teamIDStr = teamIDStr[:idx]
	}

	teamID, err := strconv.Atoi(teamIDStr)
	if err != nil {
		http.Error(w, "Invalid team ID", http.StatusBadRequest)
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

	// Update team with API Football ID
	err = h.queries.UpdateTeamApiFootballID(ctx, generated.UpdateTeamApiFootballIDParams{
		ID:            int32(teamID),
		ApiFootballID: &req.ApiFootballID,
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to update team mapping")
		http.Error(w, "Failed to update team mapping", http.StatusInternalServerError)
		return
	}

	// Also create/update team mapping record
	_, err = h.queries.UpsertTeamMapping(ctx, generated.UpsertTeamMappingParams{
		InternalTeamID:    int32(teamID),
		FootballApiTeamID: req.ApiFootballID,
		Confidence:        1.0,
		MappingMethod:     "manual",
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create team mapping record")
		// Don't fail the request, just log the error
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Message: "Team mapping updated successfully",
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
