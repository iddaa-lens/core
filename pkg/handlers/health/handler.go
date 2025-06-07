package health

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models/api"
)

// Handler handles health check requests
type Handler struct {
	logger *logger.Logger
}

// NewHandler creates a new health handler
func NewHandler(log *logger.Logger) *Handler {
	return &Handler{
		logger: log,
	}
}

// HealthCheck handles the /health endpoint
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	response := api.HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error().
			Err(err).
			Str("action", "health_check_failed").
			Str("endpoint", "/health").
			Msg("Failed to encode health response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	h.logger.Debug().
		Str("action", "health_check").
		Str("endpoint", "/health").
		Str("method", r.Method).
		Str("remote_addr", r.RemoteAddr).
		Int("status_code", 200).
		Dur("duration", time.Since(start)).
		Msg("Health check completed")
}
