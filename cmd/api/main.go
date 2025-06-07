package main

import (
	"github.com/iddaa-lens/core/internal/config"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/server"
)

func main() {
	// Setup structured logging
	logger.SetupLogger()
	log := logger.New("api-service")

	// Load configuration
	cfg := config.Load()

	// Create and configure server
	srv, err := server.New(cfg, log)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("action", "server_creation_failed").
			Msg("Failed to create server")
	}
	defer srv.Close()

	// Start server
	if err := srv.Start(); err != nil {
		log.Fatal().
			Err(err).
			Str("action", "server_failed").
			Msg("Server failed to start")
	}
}
