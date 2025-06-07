package main

import (
	"fmt"
	"os"

	"github.com/iddaa-lens/core/internal/config"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/server"
)

func main() {
	// Handle health check flag for Docker health checks
	if len(os.Args) > 1 && os.Args[1] == "--health-check" {
		// Simple health check - just exit with 0 if the binary can run
		fmt.Println("OK")
		os.Exit(0)
	}

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
