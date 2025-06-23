package smart_money

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models/api"
	"github.com/iddaa-lens/core/pkg/services"
)

// Handler handles smart money tracker API endpoints
type Handler struct {
	queries *generated.Queries
	tracker *services.SmartMoneyTracker
	logger  *logger.Logger
}

// NewHandler creates a new smart money tracker handler
func NewHandler(queries *generated.Queries, tracker *services.SmartMoneyTracker) *Handler {
	return &Handler{
		queries: queries,
		tracker: tracker,
		logger:  logger.New("smart-money-handler"),
	}
}

// BigMoversResponse represents the response for big movers endpoint
type BigMoversResponse struct {
	Movements []BigMoverData `json:"movements"`
	Total     int            `json:"total"`
	TimeRange string         `json:"time_range"`
}

// BigMoverData represents a single big mover
type BigMoverData struct {
	ID               int64     `json:"id"`
	EventID          int       `json:"event_id"`
	EventExternalID  string    `json:"event_external_id"`
	HomeTeam         string    `json:"home_team"`
	AwayTeam         string    `json:"away_team"`
	MarketName       string    `json:"market_name"`
	Outcome          string    `json:"outcome"`
	PreviousOdds     float64   `json:"previous_odds"`
	CurrentOdds      float64   `json:"current_odds"`
	ChangePercent    float64   `json:"change_percent"`
	Multiplier       float64   `json:"multiplier"`
	MinutesToKickoff int       `json:"minutes_to_kickoff"`
	RecordedAt       time.Time `json:"recorded_at"`
	IsLive           bool      `json:"is_live"`
	AlertMessage     string    `json:"alert_message"`
}

// GetBigMovers handles GET /api/smart-money/big-movers
func (h *Handler) GetBigMovers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters with simple defaults
	hoursBack := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil && parsed > 0 && parsed <= 168 {
			hoursBack = parsed
		}
	}

	minChangePercent := 20.0
	if p := r.URL.Query().Get("min_change"); p != "" {
		if parsed, err := strconv.ParseFloat(p, 64); err == nil && parsed >= 5.0 {
			minChangePercent = parsed
		}
	}

	minMultiplier := 2.0
	if m := r.URL.Query().Get("min_multiplier"); m != "" {
		if parsed, err := strconv.ParseFloat(m, 64); err == nil && parsed >= 1.1 {
			minMultiplier = parsed
		}
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	// Get big movers from database
	since := time.Now().Add(-time.Duration(hoursBack) * time.Hour)

	bigMovers, err := h.queries.GetRecentBigMovers(ctx, generated.GetRecentBigMoversParams{
		MinChangePct:  minChangePercent,
		MinMultiplier: minMultiplier,
		SinceTime:     pgtype.Timestamp{Time: since, Valid: true},
		LimitCount:    int32(limit),
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get big movers")
		http.Error(w, "Failed to retrieve big movers", http.StatusInternalServerError)
		return
	}

	// Convert database records to response format
	movements := make([]BigMoverData, 0, len(bigMovers))

	for _, mover := range bigMovers {
		// Handle nullable fields with defaults
		changePercent := 0.0
		if mover.ChangePercentage != nil {
			changePercent = float64(*mover.ChangePercentage)
		}

		multiplier := 1.0
		if mover.Multiplier != nil {
			multiplier = *mover.Multiplier
		}

		previousOdds := mover.OddsValue // default to current
		if mover.PreviousValue != nil {
			previousOdds = *mover.PreviousValue
		}

		homeTeam := "TBD"
		if mover.HomeTeamName != nil {
			homeTeam = *mover.HomeTeamName
		}

		awayTeam := "TBD"
		if mover.AwayTeamName != nil {
			awayTeam = *mover.AwayTeamName
		}

		eventID := 0
		if mover.EventID != nil {
			eventID = int(*mover.EventID)
		}

		isLive := false
		if mover.IsLive != nil {
			isLive = *mover.IsLive
		}

		movement := BigMoverData{
			ID:              int64(mover.ID),
			EventID:         eventID,
			EventExternalID: mover.EventExternalID,
			HomeTeam:        homeTeam,
			AwayTeam:        awayTeam,
			MarketName:      mover.MarketName,
			Outcome:         mover.Outcome,
			PreviousOdds:    previousOdds,
			CurrentOdds:     mover.OddsValue,
			ChangePercent:   changePercent,
			Multiplier:      multiplier,
			RecordedAt:      mover.RecordedAt.Time,
			IsLive:          isLive,
			AlertMessage:    fmt.Sprintf("%.1f%% movement (%.2fx)", changePercent, multiplier),
		}

		// Calculate minutes to kickoff
		if mover.EventDate.Valid {
			minutesToKickoff := time.Until(mover.EventDate.Time).Minutes()
			if minutesToKickoff > 0 {
				movement.MinutesToKickoff = int(minutesToKickoff)
			}
		}

		movements = append(movements, movement)
	}

	response := BigMoversResponse{
		Movements: movements,
		Total:     len(movements),
		TimeRange: strconv.Itoa(hoursBack) + " hours",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Data:    response,
		Message: "Big movers retrieved successfully",
	}); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// AlertsResponse represents the response for alerts endpoint
type AlertsResponse struct {
	Alerts  []AlertData `json:"alerts"`
	Total   int         `json:"total"`
	HasMore bool        `json:"has_more"`
}

// AlertData represents a single alert
type AlertData struct {
	ID               int64     `json:"id"`
	AlertType        string    `json:"alert_type"`
	Severity         string    `json:"severity"`
	Title            string    `json:"title"`
	Message          string    `json:"message"`
	ChangePercent    float32   `json:"change_percent"`
	Multiplier       float64   `json:"multiplier"`
	ConfidenceScore  float32   `json:"confidence_score"`
	MinutesToKickoff int       `json:"minutes_to_kickoff"`
	EventExternalID  string    `json:"event_external_id"`
	HomeTeam         string    `json:"home_team"`
	AwayTeam         string    `json:"away_team"`
	MarketName       string    `json:"market_name"`
	Outcome          string    `json:"outcome"`
	CreatedAt        time.Time `json:"created_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	Views            int       `json:"views"`
	Clicks           int       `json:"clicks"`
}

// GetAlerts handles GET /api/smart-money/alerts
func (h *Handler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	alertType := r.URL.Query().Get("type")
	minSeverity := r.URL.Query().Get("severity")

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	// Get active alerts from database
	alerts, err := h.queries.GetActiveAlerts(ctx, generated.GetActiveAlertsParams{
		AlertType:   alertType,
		MinSeverity: minSeverity,
		LimitCount:  int64(limit),
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get active alerts")
		http.Error(w, "Failed to retrieve alerts", http.StatusInternalServerError)
		return
	}

	// Convert database records to response format
	alertData := make([]AlertData, 0, len(alerts))

	for _, alert := range alerts {
		minutesToKickoff := 0
		if alert.MinutesToKickoff != nil {
			minutesToKickoff = int(*alert.MinutesToKickoff)
		}

		homeTeam := ""
		if alert.HomeTeamName != nil {
			homeTeam = *alert.HomeTeamName
		}

		awayTeam := ""
		if alert.AwayTeamName != nil {
			awayTeam = *alert.AwayTeamName
		}

		alertData = append(alertData, AlertData{
			ID:               int64(alert.ID),
			AlertType:        alert.AlertType,
			Severity:         alert.Severity,
			Title:            alert.Title,
			Message:          alert.Message,
			ChangePercent:    alert.ChangePercentage,
			Multiplier:       alert.Multiplier,
			ConfidenceScore:  alert.ConfidenceScore,
			MinutesToKickoff: minutesToKickoff,
			EventExternalID:  alert.EventExternalID,
			HomeTeam:         homeTeam,
			AwayTeam:         awayTeam,
			MarketName:       alert.MarketName,
			Outcome:          alert.Outcome,
			CreatedAt:        alert.CreatedAt.Time,
			ExpiresAt:        alert.ExpiresAt.Time,
			Views:            int(alert.Views),
			Clicks:           int(alert.Clicks),
		})
	}

	response := AlertsResponse{
		Alerts:  alertData,
		Total:   len(alertData),
		HasMore: len(alertData) == limit,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Data:    response,
		Message: "Alerts retrieved successfully",
	}); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// ValueSpotsResponse represents the response for value spots endpoint
type ValueSpotsResponse struct {
	ValueSpots []ValueSpotData `json:"value_spots"`
	Total      int             `json:"total"`
}

// ValueSpotData represents a single value betting opportunity
type ValueSpotData struct {
	ID               int64     `json:"id"`
	EventExternalID  string    `json:"event_external_id"`
	HomeTeam         string    `json:"home_team"`
	AwayTeam         string    `json:"away_team"`
	MarketName       string    `json:"market_name"`
	Outcome          string    `json:"outcome"`
	CurrentOdds      float64   `json:"current_odds"`
	ImpliedProb      float64   `json:"implied_probability"`
	PublicPercent    float64   `json:"public_percent"`
	PublicBias       float64   `json:"public_bias"`
	ValueScore       float64   `json:"value_score"`
	MinutesToKickoff int       `json:"minutes_to_kickoff"`
	RecordedAt       time.Time `json:"recorded_at"`
	AlertMessage     string    `json:"alert_message"`
}

// GetValueSpots handles GET /api/smart-money/value-spots
func (h *Handler) GetValueSpots(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	minBias := 10.0
	if b := r.URL.Query().Get("min_bias"); b != "" {
		if parsed, err := strconv.ParseFloat(b, 64); err == nil && parsed >= 5.0 {
			minBias = parsed
		}
	}

	limit := 30
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Get value spots from database
	since := time.Now().Add(-24 * time.Hour)

	valueSpots, err := h.queries.GetValueSpots(ctx, generated.GetValueSpotsParams{
		SinceTime:      pgtype.Timestamp{Time: since, Valid: true},
		MinBiasPct:     minBias,
		MinMovementPct: 5.0,
		LimitCount:     int64(limit),
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get value spots")
		http.Error(w, "Failed to retrieve value spots", http.StatusInternalServerError)
		return
	}

	// Convert database records to response format
	valueSpotData := make([]ValueSpotData, 0, len(valueSpots))

	for _, spot := range valueSpots {
		impliedProb := 0.0
		if spot.ImpliedProbability != nil {
			impliedProb = float64(*spot.ImpliedProbability)
		}

		publicPercent := 0.0
		if spot.BetPercentage != nil {
			publicPercent = float64(*spot.BetPercentage)
		}

		publicBias := float64(spot.PublicBias)

		homeTeam := "TBD"
		if spot.HomeTeamName != nil {
			homeTeam = *spot.HomeTeamName
		}

		awayTeam := "TBD"
		if spot.AwayTeamName != nil {
			awayTeam = *spot.AwayTeamName
		}

		valueSpot := ValueSpotData{
			ID:              int64(spot.ID),
			EventExternalID: spot.EventExternalID,
			HomeTeam:        homeTeam,
			AwayTeam:        awayTeam,
			MarketName:      spot.MarketName,
			Outcome:         spot.Outcome,
			CurrentOdds:     spot.OddsValue,
			ImpliedProb:     impliedProb,
			PublicPercent:   publicPercent,
			PublicBias:      publicBias,
			ValueScore:      publicBias,
			RecordedAt:      spot.RecordedAt.Time,
			AlertMessage:    fmt.Sprintf("%.1f%% public bias - potential value", publicBias),
		}

		// Calculate minutes to kickoff
		if spot.EventDate.Valid {
			minutesToKickoff := time.Until(spot.EventDate.Time).Minutes()
			if minutesToKickoff > 0 {
				valueSpot.MinutesToKickoff = int(minutesToKickoff)
			}
		}

		valueSpotData = append(valueSpotData, valueSpot)
	}

	response := ValueSpotsResponse{
		ValueSpots: valueSpotData,
		Total:      len(valueSpotData),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Data:    response,
		Message: "Value spots retrieved successfully",
	}); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// MarkAlertViewed handles POST /api/smart-money/alerts/{id}/view
func (h *Handler) MarkAlertViewed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract alert ID from URL path
	alertIDStr := path.Base(path.Dir(r.URL.Path))
	alertID, err := strconv.ParseInt(alertIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid alert ID", http.StatusBadRequest)
		return
	}

	// Mark alert as viewed in database
	if err := h.queries.MarkAlertViewed(ctx, int32(alertID)); err != nil {
		h.logger.Error().Err(err).Int64("alert_id", alertID).Msg("Failed to mark alert as viewed")
		http.Error(w, "Failed to mark alert as viewed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Message: "Alert marked as viewed",
	}); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// MarkAlertClicked handles POST /api/smart-money/alerts/{id}/click
func (h *Handler) MarkAlertClicked(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract alert ID from URL path
	alertIDStr := path.Base(path.Dir(r.URL.Path))
	alertID, err := strconv.ParseInt(alertIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid alert ID", http.StatusBadRequest)
		return
	}

	// Mark alert as clicked in database
	if err := h.queries.MarkAlertClicked(ctx, int32(alertID)); err != nil {
		h.logger.Error().Err(err).Int64("alert_id", alertID).Msg("Failed to mark alert as clicked")
		http.Error(w, "Failed to mark alert as clicked", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Message: "Alert marked as clicked",
	}); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// DashboardResponse represents the response for the dashboard endpoint
type DashboardResponse struct {
	Summary         DashboardSummary `json:"summary"`
	RecentBigMovers []BigMoverData   `json:"recent_big_movers"`
	ActiveAlerts    []AlertData      `json:"active_alerts"`
	TopValueSpots   []ValueSpotData  `json:"top_value_spots"`
	LastUpdated     time.Time        `json:"last_updated"`
}

// DashboardSummary provides overview statistics
type DashboardSummary struct {
	TotalActiveAlerts     int `json:"total_active_alerts"`
	BigMoversLast24h      int `json:"big_movers_last_24h"`
	ReverseMovementsToday int `json:"reverse_movements_today"`
	ValueSpotsAvailable   int `json:"value_spots_available"`
	SharpMoneySignals     int `json:"sharp_money_signals"`
}

// GetDashboard handles GET /api/smart-money/dashboard
func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Aggregate dashboard data from multiple sources
	since24h := time.Now().Add(-24 * time.Hour)

	// Get active alerts count
	activeAlerts, err := h.queries.GetActiveAlerts(ctx, generated.GetActiveAlertsParams{
		AlertType:   "",
		MinSeverity: "",
		LimitCount:  5,
	})
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get active alerts for dashboard")
		activeAlerts = []generated.GetActiveAlertsRow{}
	}

	// Get recent big movers
	bigMovers, err := h.queries.GetRecentBigMovers(ctx, generated.GetRecentBigMoversParams{
		MinChangePct:  20.0,
		MinMultiplier: 2.0,
		SinceTime:     pgtype.Timestamp{Time: since24h, Valid: true},
		LimitCount:    5,
	})
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get big movers for dashboard")
		bigMovers = []generated.GetRecentBigMoversRow{}
	}

	// Get value spots
	valueSpots, err := h.queries.GetValueSpots(ctx, generated.GetValueSpotsParams{
		SinceTime:      pgtype.Timestamp{Time: since24h, Valid: true},
		MinBiasPct:     10.0,
		MinMovementPct: 5.0,
		LimitCount:     5,
	})
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get value spots for dashboard")
		valueSpots = []generated.GetValueSpotsRow{}
	}

	// Convert to response format - simplified for dashboard
	dashboardAlerts := make([]AlertData, 0, len(activeAlerts))
	for _, alert := range activeAlerts {
		dashboardAlerts = append(dashboardAlerts, AlertData{
			ID:        int64(alert.ID),
			AlertType: alert.AlertType,
			Severity:  alert.Severity,
			Title:     alert.Title,
			Message:   alert.Message,
			CreatedAt: alert.CreatedAt.Time,
		})
	}

	dashboardMovers := make([]BigMoverData, 0, len(bigMovers))
	for _, mover := range bigMovers {
		dashboardMovers = append(dashboardMovers, BigMoverData{
			ID:              int64(mover.ID),
			EventExternalID: mover.EventExternalID,
			MarketName:      mover.MarketName,
			Outcome:         mover.Outcome,
			RecordedAt:      mover.RecordedAt.Time,
		})
	}

	dashboardValueSpots := make([]ValueSpotData, 0, len(valueSpots))
	for _, spot := range valueSpots {
		dashboardValueSpots = append(dashboardValueSpots, ValueSpotData{
			ID:              int64(spot.ID),
			EventExternalID: spot.EventExternalID,
			MarketName:      spot.MarketName,
			Outcome:         spot.Outcome,
			RecordedAt:      spot.RecordedAt.Time,
		})
	}

	// Count reverse movements and sharp money signals
	reverseMovements := 0
	sharpMoneySignals := 0
	for _, alert := range activeAlerts {
		switch alert.AlertType {
		case "reverse_line":
			reverseMovements++
		case "sharp_money":
			sharpMoneySignals++
		}
	}

	response := DashboardResponse{
		Summary: DashboardSummary{
			TotalActiveAlerts:     len(activeAlerts),
			BigMoversLast24h:      len(bigMovers),
			ReverseMovementsToday: reverseMovements,
			ValueSpotsAvailable:   len(valueSpots),
			SharpMoneySignals:     sharpMoneySignals,
		},
		RecentBigMovers: dashboardMovers,
		ActiveAlerts:    dashboardAlerts,
		TopValueSpots:   dashboardValueSpots,
		LastUpdated:     time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Data:    response,
		Message: "Dashboard data retrieved successfully",
	}); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
