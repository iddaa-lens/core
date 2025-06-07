package smart_money

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models/api"
	"github.com/iddaa-lens/core/pkg/services"
	"github.com/jackc/pgx/v5/pgtype"
)

// Handler handles smart money tracker API endpoints
type Handler struct {
	queries *database.Queries
	tracker *services.SmartMoneyTracker
	logger  *logger.Logger
}

// NewHandler creates a new smart money tracker handler
func NewHandler(queries *database.Queries, tracker *services.SmartMoneyTracker) *Handler {
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

	// Parse query parameters
	hoursBack := 24 // default
	if h := r.URL.Query().Get("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil && parsed > 0 && parsed <= 168 {
			hoursBack = parsed
		}
	}

	minChangePercent := 20.0 // default
	if p := r.URL.Query().Get("min_change"); p != "" {
		if parsed, err := strconv.ParseFloat(p, 64); err == nil && parsed >= 5.0 {
			minChangePercent = parsed
		}
	}

	minMultiplier := 2.0 // default
	if m := r.URL.Query().Get("min_multiplier"); m != "" {
		if parsed, err := strconv.ParseFloat(m, 64); err == nil && parsed >= 1.1 {
			minMultiplier = parsed
		}
	}

	limit := 50 // default
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	// Get big movers from database
	since := time.Now().Add(-time.Duration(hoursBack) * time.Hour)
	var minChangePctNumeric, minMultiplierNumeric pgtype.Numeric
	if err := minChangePctNumeric.Scan(fmt.Sprintf("%.2f", minChangePercent)); err != nil {
		h.logger.Error().Err(err).Msg("Failed to scan min change percentage")
		return
	}
	if err := minMultiplierNumeric.Scan(fmt.Sprintf("%.2f", minMultiplier)); err != nil {
		h.logger.Error().Err(err).Msg("Failed to scan min multiplier")
		return
	}

	bigMovers, err := h.queries.GetRecentBigMovers(ctx, database.GetRecentBigMoversParams{
		MinChangePct:  minChangePctNumeric,
		MinMultiplier: minMultiplierNumeric,
		SinceTime:     pgtype.Timestamp{Time: since, Valid: true},
		LimitCount:    int32(limit),
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get big movers")
		http.Error(w, "Failed to retrieve big movers", http.StatusInternalServerError)
		return
	}

	// Convert database records to response format
	movements := make([]BigMoverData, len(bigMovers))
	for i, mover := range bigMovers {
		changePercent := 0.0
		if mover.ChangePercentage.Valid {
			if changeFloat, err := mover.ChangePercentage.Float64Value(); err == nil && changeFloat.Valid {
				changePercent = changeFloat.Float64
			}
		}

		multiplier := 1.0
		if mover.Multiplier.Valid {
			if multiplierFloat, err := mover.Multiplier.Float64Value(); err == nil && multiplierFloat.Valid {
				multiplier = multiplierFloat.Float64
			}
		}

		previousOdds := 0.0
		if mover.PreviousValue.Valid {
			if prevFloat, err := mover.PreviousValue.Float64Value(); err == nil && prevFloat.Valid {
				previousOdds = prevFloat.Float64
			}
		}

		currentOdds := 0.0
		if mover.OddsValue.Valid {
			if currentFloat, err := mover.OddsValue.Float64Value(); err == nil && currentFloat.Valid {
				currentOdds = currentFloat.Float64
			}
		}

		homeTeam := "TBD"
		if mover.HomeTeamName.Valid {
			homeTeam = mover.HomeTeamName.String
		}

		awayTeam := "TBD"
		if mover.AwayTeamName.Valid {
			awayTeam = mover.AwayTeamName.String
		}

		movements[i] = BigMoverData{
			ID:              int64(mover.ID),
			EventID:         int(mover.EventID.Int32),
			EventExternalID: mover.EventExternalID,
			HomeTeam:        homeTeam,
			AwayTeam:        awayTeam,
			MarketName:      mover.MarketName,
			Outcome:         mover.Outcome,
			PreviousOdds:    previousOdds,
			CurrentOdds:     currentOdds,
			ChangePercent:   changePercent,
			Multiplier:      multiplier,
			RecordedAt:      mover.RecordedAt.Time,
			IsLive:          mover.IsLive.Bool,
			AlertMessage:    fmt.Sprintf("%.1f%% movement (%.2fx)", changePercent, multiplier),
		}

		// Calculate minutes to kickoff
		if mover.EventDate.Valid {
			minutesToKickoff := time.Until(mover.EventDate.Time).Minutes()
			if minutesToKickoff > 0 {
				movements[i].MinutesToKickoff = int(minutesToKickoff)
			}
		}
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
	ChangePercent    float64   `json:"change_percent"`
	Multiplier       float64   `json:"multiplier"`
	ConfidenceScore  float64   `json:"confidence_score"`
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
	alertType := r.URL.Query().Get("type")       // 'big_mover', 'reverse_line', 'sharp_money', 'value_spot'
	minSeverity := r.URL.Query().Get("severity") // 'low', 'medium', 'high', 'critical'

	limit := 50 // default
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	// Get active alerts from database
	var alertTypePtr *string
	if alertType != "" {
		alertTypePtr = &alertType
	}

	var minSeverityPtr *string
	if minSeverity != "" {
		minSeverityPtr = &minSeverity
	}

	// Convert pointers to strings for database query
	alertTypeStr := ""
	if alertTypePtr != nil {
		alertTypeStr = *alertTypePtr
	}

	minSeverityStr := ""
	if minSeverityPtr != nil {
		minSeverityStr = *minSeverityPtr
	}

	alerts, err := h.queries.GetActiveAlerts(ctx, database.GetActiveAlertsParams{
		AlertType:   alertTypeStr,
		MinSeverity: minSeverityStr,
		LimitCount:  int32(limit),
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get active alerts")
		http.Error(w, "Failed to retrieve alerts", http.StatusInternalServerError)
		return
	}

	// Convert database records to response format
	alertData := make([]AlertData, len(alerts))
	for i, alert := range alerts {
		changePercent := 0.0
		if alert.ChangePercentage.Valid {
			if changeFloat, err := alert.ChangePercentage.Float64Value(); err == nil && changeFloat.Valid {
				changePercent = changeFloat.Float64
			}
		}

		multiplier := 1.0
		if alert.Multiplier.Valid {
			if multiplierFloat, err := alert.Multiplier.Float64Value(); err == nil && multiplierFloat.Valid {
				multiplier = multiplierFloat.Float64
			}
		}

		confidenceScore := 0.0
		if alert.ConfidenceScore.Valid {
			if confFloat, err := alert.ConfidenceScore.Float64Value(); err == nil && confFloat.Valid {
				confidenceScore = confFloat.Float64
			}
		}

		minutesToKickoff := 0
		if alert.MinutesToKickoff.Valid {
			minutesToKickoff = int(alert.MinutesToKickoff.Int32)
		}

		alertData[i] = AlertData{
			ID:               int64(alert.ID),
			AlertType:        alert.AlertType,
			Severity:         alert.Severity,
			Title:            alert.Title,
			Message:          alert.Message,
			ChangePercent:    changePercent,
			Multiplier:       multiplier,
			ConfidenceScore:  confidenceScore,
			MinutesToKickoff: minutesToKickoff,
			EventExternalID:  alert.EventExternalID,
			HomeTeam:         alert.HomeTeamName.String,
			AwayTeam:         alert.AwayTeamName.String,
			MarketName:       alert.MarketName,
			Outcome:          alert.Outcome,
			CreatedAt:        alert.CreatedAt.Time,
			ExpiresAt:        alert.ExpiresAt.Time,
			Views: func() int {
				if alert.Views.Valid {
					return int(alert.Views.Int32)
				} else {
					return 0
				}
			}(),
			Clicks: func() int {
				if alert.Clicks.Valid {
					return int(alert.Clicks.Int32)
				} else {
					return 0
				}
			}(),
		}
	}

	response := AlertsResponse{
		Alerts:  alertData,
		Total:   len(alertData),
		HasMore: len(alertData) == limit, // Simple pagination check
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Data:    response,
		Message: "Alerts retrieved successfully",
	}); err != nil {
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
	minBias := 10.0 // default minimum public bias percentage
	if b := r.URL.Query().Get("min_bias"); b != "" {
		if parsed, err := strconv.ParseFloat(b, 64); err == nil && parsed >= 5.0 {
			minBias = parsed
		}
	}

	limit := 30 // default
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Get value spots from database
	since := time.Now().Add(-24 * time.Hour) // Last 24 hours
	var minBiasNumeric, minMovementNumeric pgtype.Numeric
	if err := minBiasNumeric.Scan(fmt.Sprintf("%.2f", minBias)); err != nil {
		h.logger.Error().Err(err).Msg("Failed to scan min bias")
		http.Error(w, "Invalid bias parameter", http.StatusBadRequest)
		return
	}
	if err := minMovementNumeric.Scan("5.0"); err != nil {
		h.logger.Error().Err(err).Msg("Failed to scan min movement")
		http.Error(w, "Invalid movement parameter", http.StatusBadRequest)
		return
	}

	valueSpots, err := h.queries.GetValueSpots(ctx, database.GetValueSpotsParams{
		SinceTime:      pgtype.Timestamp{Time: since, Valid: true},
		MinBiasPct:     minBiasNumeric,
		MinMovementPct: minMovementNumeric,
		LimitCount:     int32(limit),
	})

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get value spots")
		http.Error(w, "Failed to retrieve value spots", http.StatusInternalServerError)
		return
	}

	// Convert database records to response format
	valueSpotData := make([]ValueSpotData, len(valueSpots))
	for i, spot := range valueSpots {
		currentOdds := 0.0
		if spot.OddsValue.Valid {
			if oddsFloat, err := spot.OddsValue.Float64Value(); err == nil && oddsFloat.Valid {
				currentOdds = oddsFloat.Float64
			}
		}

		impliedProb := 0.0
		if spot.ImpliedProbability.Valid {
			if impliedFloat, err := spot.ImpliedProbability.Float64Value(); err == nil && impliedFloat.Valid {
				impliedProb = impliedFloat.Float64
			}
		}

		publicPercent := 0.0
		if spot.BetPercentage.Valid {
			if publicFloat, err := spot.BetPercentage.Float64Value(); err == nil && publicFloat.Valid {
				publicPercent = publicFloat.Float64
			}
		}

		publicBias := float64(spot.PublicBias)

		valueSpotData[i] = ValueSpotData{
			ID:              int64(spot.ID),
			EventExternalID: spot.EventExternalID,
			HomeTeam: func() string {
				if spot.HomeTeamName.Valid {
					return spot.HomeTeamName.String
				} else {
					return "TBD"
				}
			}(),
			AwayTeam: func() string {
				if spot.AwayTeamName.Valid {
					return spot.AwayTeamName.String
				} else {
					return "TBD"
				}
			}(),
			MarketName:    spot.MarketName,
			Outcome:       spot.Outcome,
			CurrentOdds:   currentOdds,
			ImpliedProb:   impliedProb,
			PublicPercent: publicPercent,
			PublicBias:    publicBias,
			ValueScore:    publicBias, // Use bias as value score
			RecordedAt:    spot.RecordedAt.Time,
			AlertMessage:  fmt.Sprintf("%.1f%% public bias - potential value", publicBias),
		}

		// Calculate minutes to kickoff
		if spot.EventDate.Valid {
			minutesToKickoff := time.Until(spot.EventDate.Time).Minutes()
			if minutesToKickoff > 0 {
				valueSpotData[i].MinutesToKickoff = int(minutesToKickoff)
			}
		}
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
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// MarkAlertViewed handles POST /api/smart-money/alerts/{id}/view
func (h *Handler) MarkAlertViewed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract alert ID from URL path
	alertIDStr := path.Base(path.Dir(r.URL.Path)) // Gets {id} from /alerts/{id}/view
	alertID, err := strconv.ParseInt(alertIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid alert ID", http.StatusBadRequest)
		return
	}

	// Mark alert as viewed in database
	err = h.queries.MarkAlertViewed(ctx, int32(alertID))
	if err != nil {
		h.logger.Error().Err(err).Int64("alert_id", alertID).Msg("Failed to mark alert as viewed")
		http.Error(w, "Failed to mark alert as viewed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Message: "Alert marked as viewed",
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// MarkAlertClicked handles POST /api/smart-money/alerts/{id}/click
func (h *Handler) MarkAlertClicked(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract alert ID from URL path
	alertIDStr := path.Base(path.Dir(r.URL.Path)) // Gets {id} from /alerts/{id}/click
	alertID, err := strconv.ParseInt(alertIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid alert ID", http.StatusBadRequest)
		return
	}

	// Mark alert as clicked in database
	err = h.queries.MarkAlertClicked(ctx, int32(alertID))
	if err != nil {
		h.logger.Error().Err(err).Int64("alert_id", alertID).Msg("Failed to mark alert as clicked")
		http.Error(w, "Failed to mark alert as clicked", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(api.Response{
		Success: true,
		Message: "Alert marked as clicked",
	}); err != nil {
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
	activeAlerts, err := h.queries.GetActiveAlerts(ctx, database.GetActiveAlertsParams{
		AlertType:   "",
		MinSeverity: "",
		LimitCount:  5, // Top 5 for dashboard
	})
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get active alerts for dashboard")
		activeAlerts = []database.GetActiveAlertsRow{} // Continue with empty data
	}

	// Get recent big movers
	var minChangePctNumeric, minMultiplierNumeric pgtype.Numeric
	var bigMovers []database.GetRecentBigMoversRow

	if err := minChangePctNumeric.Scan("20.0"); err != nil {
		h.logger.Error().Err(err).Msg("Failed to scan min change percentage")
		bigMovers = []database.GetRecentBigMoversRow{} // Continue with empty data
	} else if err := minMultiplierNumeric.Scan("2.0"); err != nil {
		h.logger.Error().Err(err).Msg("Failed to scan min multiplier")
		bigMovers = []database.GetRecentBigMoversRow{} // Continue with empty data
	} else {
		bigMovers, err = h.queries.GetRecentBigMovers(ctx, database.GetRecentBigMoversParams{
			MinChangePct:  minChangePctNumeric,
			MinMultiplier: minMultiplierNumeric,
			SinceTime:     pgtype.Timestamp{Time: since24h, Valid: true},
			LimitCount:    5, // Top 5 for dashboard
		})
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to get big movers for dashboard")
			bigMovers = []database.GetRecentBigMoversRow{} // Continue with empty data
		}
	}

	// Get value spots
	var minBiasNumeric, minMovementNumeric pgtype.Numeric
	var valueSpots []database.GetValueSpotsRow

	if err := minBiasNumeric.Scan("10.0"); err != nil {
		h.logger.Error().Err(err).Msg("Failed to scan min bias")
		valueSpots = []database.GetValueSpotsRow{} // Continue with empty data
	} else if err := minMovementNumeric.Scan("5.0"); err != nil {
		h.logger.Error().Err(err).Msg("Failed to scan min movement")
		valueSpots = []database.GetValueSpotsRow{} // Continue with empty data
	} else {
		valueSpots, err = h.queries.GetValueSpots(ctx, database.GetValueSpotsParams{
			SinceTime:      pgtype.Timestamp{Time: since24h, Valid: true},
			MinBiasPct:     minBiasNumeric,
			MinMovementPct: minMovementNumeric,
			LimitCount:     5, // Top 5 for dashboard
		})
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to get value spots for dashboard")
			valueSpots = []database.GetValueSpotsRow{} // Continue with empty data
		}
	}

	// Convert to response format (reusing existing conversion logic)
	dashboardAlerts := make([]AlertData, len(activeAlerts))
	for i, alert := range activeAlerts {
		// Simplified conversion for dashboard
		dashboardAlerts[i] = AlertData{
			ID:        int64(alert.ID),
			AlertType: alert.AlertType,
			Severity:  alert.Severity,
			Title:     alert.Title,
			Message:   alert.Message,
			CreatedAt: alert.CreatedAt.Time,
		}
	}

	dashboardMovers := make([]BigMoverData, len(bigMovers))
	for i, mover := range bigMovers {
		dashboardMovers[i] = BigMoverData{
			ID:              int64(mover.ID),
			EventExternalID: mover.EventExternalID,
			MarketName:      mover.MarketName,
			Outcome:         mover.Outcome,
			RecordedAt:      mover.RecordedAt.Time,
		}
	}

	dashboardValueSpots := make([]ValueSpotData, len(valueSpots))
	for i, spot := range valueSpots {
		dashboardValueSpots[i] = ValueSpotData{
			ID:              int64(spot.ID),
			EventExternalID: spot.EventExternalID,
			MarketName:      spot.MarketName,
			Outcome:         spot.Outcome,
			RecordedAt:      spot.RecordedAt.Time,
		}
	}

	// Count reverse movements (alerts with type "reverse_line")
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
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
