package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/iddaa-lens/core/internal/config"
	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/handlers/events"
	"github.com/iddaa-lens/core/pkg/handlers/health"
	"github.com/iddaa-lens/core/pkg/handlers/leagues"
	"github.com/iddaa-lens/core/pkg/handlers/odds"
	"github.com/iddaa-lens/core/pkg/handlers/teams"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server represents the API server
type Server struct {
	router   *http.ServeMux
	port     string
	logger   *logger.Logger
	dbPool   *pgxpool.Pool
	queries  *database.Queries
	handlers struct {
		health  *health.Handler
		events  *events.Handler
		odds    *odds.Handler
		teams   *teams.Handler
		leagues *leagues.Handler
	}
}

// New creates a new server instance
func New(cfg *config.Config, log *logger.Logger) (*Server, error) {
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize database connection pool with production settings
	dbPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test database connection with retry logic
	if err := testDatabaseConnection(dbPool, log); err != nil {
		dbPool.Close()
		return nil, err
	}

	// Create queries instance
	queries := database.New(dbPool)

	// Create server instance
	server := &Server{
		router:  http.NewServeMux(),
		port:    port,
		logger:  log,
		dbPool:  dbPool,
		queries: queries,
	}

	// Initialize handlers
	server.handlers.health = health.NewHandler(log)
	server.handlers.events = events.NewHandler(queries, log)
	server.handlers.odds = odds.NewHandler(queries, log)
	server.handlers.teams = teams.NewHandler(queries, log)
	server.handlers.leagues = leagues.NewHandler(queries, log)

	// Setup routes
	server.setupRoutes()

	log.Info().
		Str("action", "db_connected").
		Msg("Database connection pool established with production settings")

	return server, nil
}

// setupRoutes configures all the API routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.HandleFunc("/health", middleware.CORS(s.handlers.health.HealthCheck))

	// Simple root endpoint
	s.router.HandleFunc("/", middleware.CORS(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintf(w, "Iddaa API Service - OK (Database Connected)"); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
	}))

	// Odds endpoints
	s.router.HandleFunc("/api/odds/big-movers", middleware.CORS(s.handlers.odds.BigMovers))

	// Events endpoints
	s.router.HandleFunc("/api/events", middleware.CORS(s.handlers.events.List))
	s.router.HandleFunc("/api/events/upcoming", middleware.CORS(s.handlers.events.Upcoming))
	s.router.HandleFunc("/api/events/daily", middleware.CORS(s.handlers.events.Daily))
	s.router.HandleFunc("/api/events/live", middleware.CORS(s.handlers.events.Live))

	// Teams endpoints
	s.router.HandleFunc("/api/teams", middleware.CORS(s.handlers.teams.List))
	s.router.HandleFunc("/api/teams/", middleware.CORS(s.handlers.teams.UpdateMapping)) // handles /api/teams/{id}/mapping

	// Leagues endpoints
	s.router.HandleFunc("/api/leagues", middleware.CORS(s.handlers.leagues.List))
	s.router.HandleFunc("/api/leagues/", middleware.CORS(s.handlers.leagues.UpdateMapping)) // handles /api/leagues/{id}/mapping
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info().
		Str("action", "server_start").
		Str("port", s.port).
		Msg("Starting API server with database connection")

	if err := http.ListenAndServe(":"+s.port, s.router); err != nil {
		return fmt.Errorf("server failed to start on port %s: %w", s.port, err)
	}

	return nil
}

// Close gracefully shuts down the server and closes database connections
func (s *Server) Close() {
	if s.dbPool != nil {
		s.dbPool.Close()
		s.logger.Info().Msg("Database connection pool closed")
	}
}

// testDatabaseConnection tests the database connection with retry logic
func testDatabaseConnection(dbPool *pgxpool.Pool, log *logger.Logger) error {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := dbPool.Ping(ctx)
		cancel()

		if err == nil {
			return nil
		}

		if i == maxRetries-1 {
			return fmt.Errorf("failed to ping database after %d retries: %w", maxRetries, err)
		}

		log.Warn().
			Err(err).
			Int("attempt", i+1).
			Str("action", "db_ping_retry").
			Msg("Retrying database connection")
		time.Sleep(2 * time.Second)
	}

	return nil
}
