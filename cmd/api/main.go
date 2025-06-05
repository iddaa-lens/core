package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/betslib/iddaa-core/pkg/logger"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	// Setup structured logging
	logger.SetupLogger()
	log := logger.New("api-service")

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Simple HTTP server
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		response := HealthResponse{
			Status:    "ok",
			Timestamp: time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error().
				Err(err).
				Str("action", "health_check_failed").
				Str("endpoint", "/health").
				Msg("Failed to encode health response")
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}

		log.Debug().
			Str("action", "health_check").
			Str("endpoint", "/health").
			Str("method", r.Method).
			Str("remote_addr", r.RemoteAddr).
			Int("status_code", 200).
			Dur("duration", time.Since(start)).
			Msg("Health check completed")
	})

	// Simple root endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintf(w, "Iddaa API Service - OK"); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
	})

	log.Info().
		Str("action", "server_start").
		Str("port", port).
		Msg("Starting API server")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal().
			Err(err).
			Str("action", "server_failed").
			Str("port", port).
			Msg("Server failed to start")
	}
}
