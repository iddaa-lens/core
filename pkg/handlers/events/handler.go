package events

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models/api"
	"github.com/jackc/pgx/v5/pgtype"
)

// Handler handles event-related requests
type Handler struct {
	queries *database.Queries
	logger  *logger.Logger
}

// NewHandler creates a new events handler
func NewHandler(queries *database.Queries, log *logger.Logger) *Handler {
	return &Handler{
		queries: queries,
		logger:  log,
	}
}

// List handles the /api/events endpoint with pagination
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	hoursBeforeStr := r.URL.Query().Get("hours_before")
	if hoursBeforeStr == "" {
		hoursBeforeStr = "24"
	}
	hoursBefore, err := strconv.Atoi(hoursBeforeStr)
	if err != nil || hoursBefore < 0 || hoursBefore > 168 {
		hoursBefore = 24
	}

	hoursAfterStr := r.URL.Query().Get("hours_after")
	if hoursAfterStr == "" {
		hoursAfterStr = "24"
	}
	hoursAfter, err := strconv.Atoi(hoursAfterStr)
	if err != nil || hoursAfter < 0 || hoursAfter > 168 {
		hoursAfter = 24
	}

	sport := r.URL.Query().Get("sport")
	league := r.URL.Query().Get("league")
	status := r.URL.Query().Get("status")

	// Parse pagination parameters
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		pageStr = "1"
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	perPageStr := r.URL.Query().Get("per_page")
	if perPageStr == "" {
		perPageStr = "20"
	}
	perPage, err := strconv.Atoi(perPageStr)
	if err != nil || perPage < 1 || perPage > 100 {
		perPage = 20
	}

	// Calculate offset
	offset := (page - 1) * perPage

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Calculate time range
	now := time.Now()
	timeAfter := pgtype.Timestamp{
		Time:  now.Add(-time.Duration(hoursBefore) * time.Hour),
		Valid: true,
	}
	timeBefore := pgtype.Timestamp{
		Time:  now.Add(time.Duration(hoursAfter) * time.Hour),
		Valid: true,
	}

	// Prepare filter parameters - use empty strings instead of nil to match SQL query expectations
	sportCode := ""
	leagueName := ""
	statusFilter := ""

	if sport != "" {
		sportCode = sport
	}
	if league != "" {
		leagueName = league
	}
	if status != "" {
		statusFilter = status
	}

	// First, get the total count for pagination
	countParams := database.CountEventsFilteredParams{
		TimeAfter:  timeAfter,
		TimeBefore: timeBefore,
		SportCode:  sportCode,
		LeagueName: leagueName,
		Status:     statusFilter,
	}

	totalCount, err := h.queries.CountEventsFiltered(ctx, countParams)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("action", "count_events_failed").
			Msg("Failed to count events")

		http.Error(w, "Failed to count events", http.StatusInternalServerError)
		return
	}

	// Query database for events with pagination
	params := database.ListEventsFilteredParams{
		TimeAfter:   timeAfter,
		TimeBefore:  timeBefore,
		SportCode:   sportCode,
		LeagueName:  leagueName,
		Status:      statusFilter,
		OffsetCount: int32(offset),
		LimitCount:  int32(perPage),
	}

	dbEvents, err := h.queries.ListEventsFiltered(ctx, params)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("action", "query_events_failed").
			Msg("Failed to query events")

		http.Error(w, "Failed to query events", http.StatusInternalServerError)
		return
	}

	// Convert database results to API response
	events := h.convertEventsToResponse(dbEvents)

	// Calculate pagination metadata
	totalPages := int((totalCount + int64(perPage) - 1) / int64(perPage)) // Ceiling division
	hasNext := page < totalPages
	hasPrevious := page > 1

	pagination := api.PaginationInfo{
		Page:        page,
		PerPage:     perPage,
		Total:       int(totalCount),
		TotalPages:  totalPages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
	}

	response := api.PaginatedEventsResponse{
		Data:       events,
		Pagination: pagination,
	}

	// Log response info
	h.logger.Info().
		Str("action", "events_response").
		Int("hours_before", hoursBefore).
		Int("hours_after", hoursAfter).
		Str("sport", sport).
		Str("league", league).
		Str("status", status).
		Int("page", page).
		Int("per_page", perPage).
		Int64("total", totalCount).
		Int("count", len(events)).
		Msg("Returning paginated events")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// Upcoming handles the /api/events/upcoming endpoint
func (h *Handler) Upcoming(w http.ResponseWriter, r *http.Request) {
	// Parse timeframe parameter (default: 6h)
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "6h"
	}

	// Parse timeframe into hours
	var hours int
	switch timeframe {
	case "1h":
		hours = 1
	case "3h":
		hours = 3
	case "6h":
		hours = 6
	case "12h":
		hours = 12
	case "24h":
		hours = 24
	case "48h":
		hours = 48
	default:
		hours = 6 // default to 6 hours
	}

	// Parse limit parameter (default: 10 for upcoming events)
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = "10"
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	// Override request query to use calculated parameters (non-paginated for upcoming events)
	r.URL.RawQuery = fmt.Sprintf("hours_before=0&hours_after=%d&status=scheduled&per_page=%d&page=1", hours, limit)

	// Call the non-paginated handler for backwards compatibility
	h.ListNonPaginated(w, r)
}

// ListNonPaginated handles events without pagination (returns array directly)
func (h *Handler) ListNonPaginated(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters (similar to List but return array directly)
	hoursBeforeStr := r.URL.Query().Get("hours_before")
	if hoursBeforeStr == "" {
		hoursBeforeStr = "24"
	}
	hoursBefore, err := strconv.Atoi(hoursBeforeStr)
	if err != nil || hoursBefore < 0 || hoursBefore > 168 {
		hoursBefore = 24
	}

	hoursAfterStr := r.URL.Query().Get("hours_after")
	if hoursAfterStr == "" {
		hoursAfterStr = "24"
	}
	hoursAfter, err := strconv.Atoi(hoursAfterStr)
	if err != nil || hoursAfter < 0 || hoursAfter > 168 {
		hoursAfter = 24
	}

	sport := r.URL.Query().Get("sport")
	league := r.URL.Query().Get("league")
	status := r.URL.Query().Get("status")

	// Parse limit (for backwards compatibility)
	limitStr := r.URL.Query().Get("per_page")
	if limitStr == "" {
		limitStr = r.URL.Query().Get("limit")
		if limitStr == "" {
			limitStr = "20"
		}
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Calculate time range
	now := time.Now()
	timeAfter := pgtype.Timestamp{
		Time:  now.Add(-time.Duration(hoursBefore) * time.Hour),
		Valid: true,
	}
	timeBefore := pgtype.Timestamp{
		Time:  now.Add(time.Duration(hoursAfter) * time.Hour),
		Valid: true,
	}

	// Prepare filter parameters
	sportCode := ""
	leagueName := ""
	statusFilter := ""

	if sport != "" {
		sportCode = sport
	}
	if league != "" {
		leagueName = league
	}
	if status != "" {
		statusFilter = status
	}

	// Query database for events (no pagination)
	params := database.ListEventsFilteredParams{
		TimeAfter:   timeAfter,
		TimeBefore:  timeBefore,
		SportCode:   sportCode,
		LeagueName:  leagueName,
		Status:      statusFilter,
		OffsetCount: 0,
		LimitCount:  int32(limit),
	}

	dbEvents, err := h.queries.ListEventsFiltered(ctx, params)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("action", "query_events_failed").
			Msg("Failed to query events")

		http.Error(w, "Failed to query events", http.StatusInternalServerError)
		return
	}

	// Convert database results to API response
	events := h.convertEventsToResponse(dbEvents)

	// Log response info
	h.logger.Info().
		Str("action", "events_non_paginated_response").
		Int("hours_before", hoursBefore).
		Int("hours_after", hoursAfter).
		Str("sport", sport).
		Str("league", league).
		Str("status", status).
		Int("limit", limit).
		Int("count", len(events)).
		Msg("Returning non-paginated events")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(events); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// Daily handles the /api/events/daily endpoint
func (h *Handler) Daily(w http.ResponseWriter, r *http.Request) {
	// Override query parameters for daily view (-24h to +24h)
	r.URL.RawQuery = "hours_before=24&hours_after=24&limit=100"
	h.List(w, r)
}

// Live handles the /api/events/live endpoint
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	// Override query parameters for live events only
	r.URL.RawQuery = "hours_before=24&hours_after=24&status=live&limit=50"
	h.List(w, r)
}

// convertEventsToResponse converts database events to API response format
func (h *Handler) convertEventsToResponse(dbEvents []database.ListEventsFilteredRow) []api.EventResponse {
	var events []api.EventResponse
	for _, event := range dbEvents {
		// Get event time
		eventTime := time.Now()
		if event.EventDate.Valid {
			eventTime = event.EventDate.Time
		}

		// Create match string
		match := fmt.Sprintf("%s vs %s", event.HomeTeamName, event.AwayTeamName)

		eventResponse := api.EventResponse{
			ID:                      event.ID,
			ExternalID:              event.ExternalID,
			Slug:                    event.Slug,
			EventDate:               eventTime,
			Status:                  event.Status,
			HomeScore:               h.getInt32Ptr(event.HomeScore),
			AwayScore:               h.getInt32Ptr(event.AwayScore),
			IsLive:                  event.IsLive.Bool,
			MinuteOfMatch:           h.getInt32Ptr(event.MinuteOfMatch),
			Half:                    h.getInt32Ptr(event.Half),
			BettingVolumePercentage: h.getFloat64Ptr(event.BettingVolumePercentage),
			VolumeRank:              h.getInt32Ptr(event.VolumeRank),
			HasKingOdd:              event.HasKingOdd.Bool,
			OddsCount:               h.getInt32Ptr(event.OddsCount),
			HasCombine:              event.HasCombine.Bool,
			HomeTeam:                event.HomeTeamName,
			HomeTeamCountry:         h.getStringFromText(event.HomeTeamCountry),
			AwayTeam:                event.AwayTeamName,
			AwayTeamCountry:         h.getStringFromText(event.AwayTeamCountry),
			League:                  event.LeagueName,
			LeagueCountry:           h.getStringFromText(event.LeagueCountry),
			Sport:                   event.SportName,
			SportCode:               event.SportCode,
			Match:                   match,
		}

		events = append(events, eventResponse)
	}
	return events
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
