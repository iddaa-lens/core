package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iddaa-lens/core/internal/config"
	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/database/pool"
	"github.com/iddaa-lens/core/pkg/handlers/events"
	"github.com/iddaa-lens/core/pkg/handlers/health"
	"github.com/iddaa-lens/core/pkg/handlers/leagues"
	"github.com/iddaa-lens/core/pkg/handlers/odds"
	"github.com/iddaa-lens/core/pkg/handlers/smart_money"
	"github.com/iddaa-lens/core/pkg/handlers/sports"
	"github.com/iddaa-lens/core/pkg/handlers/teams"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/middleware"
	"github.com/iddaa-lens/core/pkg/services"
)

// Server represents the API server
type Server struct {
	router   *http.ServeMux
	port     string
	logger   *logger.Logger
	dbPool   *pgxpool.Pool
	queries  *generated.Queries
	handlers struct {
		health     *health.Handler
		events     *events.Handler
		odds       *odds.Handler
		sports     *sports.Handler
		teams      *teams.Handler
		leagues    *leagues.Handler
		smartMoney *smart_money.Handler
	}
}

// New creates a new server instance
func New(cfg *config.Config, log *logger.Logger) (*Server, error) {
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize database connection pool with optimized production settings
	poolConfig := pool.DefaultConfig()
	dbPool, err := pool.New(context.Background(), cfg.DatabaseURL(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	// Test database connection with retry logic
	if err := testDatabaseConnection(dbPool, log); err != nil {
		dbPool.Close()
		return nil, err
	}

	// Create queries instance
	queries := generated.New(dbPool)

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
	server.handlers.sports = sports.NewHandler(queries, log)
	server.handlers.teams = teams.NewHandler(queries, log)
	server.handlers.leagues = leagues.NewHandler(queries, log)

	// Initialize smart money tracker service and handler
	smartMoneyTracker := services.NewSmartMoneyTracker(queries)
	server.handlers.smartMoney = smart_money.NewHandler(queries, smartMoneyTracker)

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

	// Smart Money endpoints
	s.router.HandleFunc("/api/smart-money/big-movers", middleware.CORS(s.handlers.smartMoney.GetBigMovers))
	s.router.HandleFunc("/api/smart-money/alerts", middleware.CORS(s.handlers.smartMoney.GetAlerts))
	s.router.HandleFunc("/api/smart-money/value-spots", middleware.CORS(s.handlers.smartMoney.GetValueSpots))
	s.router.HandleFunc("/api/smart-money/dashboard", middleware.CORS(s.handlers.smartMoney.GetDashboard))
	s.router.HandleFunc("/api/smart-money/alerts/", middleware.CORS(func(w http.ResponseWriter, r *http.Request) {
		// Handle both /alerts/{id}/view and /alerts/{id}/click
		if r.Method == "POST" {
			if r.URL.Path[len(r.URL.Path)-5:] == "/view" {
				s.handlers.smartMoney.MarkAlertViewed(w, r)
			} else if r.URL.Path[len(r.URL.Path)-6:] == "/click" {
				s.handlers.smartMoney.MarkAlertClicked(w, r)
			} else {
				http.Error(w, "Not Found", http.StatusNotFound)
			}
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}))

	// Events endpoints
	s.router.HandleFunc("/api/events", middleware.CORS(s.handlers.events.List))
	s.router.HandleFunc("/api/events/upcoming", middleware.CORS(s.handlers.events.Upcoming))
	s.router.HandleFunc("/api/events/daily", middleware.CORS(s.handlers.events.Daily))
	s.router.HandleFunc("/api/events/live", middleware.CORS(s.handlers.events.Live))

	// Sports endpoints
	s.router.HandleFunc("/api/sports", middleware.CORS(s.handlers.sports.List))

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
