package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"

	"github.com/iddaa-lens/core/internal/config"
	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/jobs"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/services"
)

func main() {
	// Parse command line flags
	var (
		jobName     = flag.String("job", "", "Run specific job once (config, sports, events, volume, distribution, analytics, market_config, statistics, leagues, detailed_odds, api_football_league_matching, api_football_team_matching, api_football_league_enrichment, api_football_team_enrichment, smart_money_processor)")
		once        = flag.Bool("once", false, "Run job once and exit")
		healthCheck = flag.Bool("health-check", false, "Perform health check and exit")
	)
	flag.Parse()

	// Handle health check flag for Docker health checks
	if *healthCheck {
		// Simple health check - just exit with 0 if the binary can run
		log := logger.New("health-check")
		log.Info().Msg("Health check OK")
		os.Exit(0)
	}

	// Setup structured logging
	logger.SetupLogger()
	log := logger.New("cron-service")

	cfg := config.Load()

	// Connect to database with optimized pool configuration
	dbConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL())
	if err != nil {
		log.Fatal().
			Err(err).
			Str("action", "db_config_parse_failed").
			Msg("Failed to parse database config")
	}

	// Configure connection pool for better performance
	dbConfig.MaxConns = 20 // Maximum number of connections
	dbConfig.MinConns = 5  // Minimum number of connections
	dbConfig.MaxConnLifetime = time.Hour
	dbConfig.MaxConnIdleTime = time.Minute * 30
	dbConfig.HealthCheckPeriod = time.Minute
	dbConfig.ConnConfig.ConnectTimeout = time.Second * 10

	db, err := pgxpool.NewWithConfig(context.Background(), dbConfig)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("action", "db_connect_failed").
			Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize services
	queries := database.New(db)
	iddaaClient := services.NewIddaaClient(cfg)
	configService := services.NewConfigService(queries, iddaaClient)
	sportsService := services.NewSportService(queries, iddaaClient)
	eventsService := services.NewEventsService(queries, iddaaClient)
	volumeService := services.NewVolumeService(queries, iddaaClient)
	distributionService := services.NewDistributionService(queries, iddaaClient)
	marketConfigService := services.NewMarketConfigService(queries, iddaaClient)
	statisticsService := services.NewStatisticsService(queries, iddaaClient)
	smartMoneyTracker := services.NewSmartMoneyTracker(queries)

	// Create job manager
	jobManager := jobs.NewJobManager()

	// Register jobs
	configJob := jobs.NewConfigSyncJob(configService, "WEB")
	if err := jobManager.RegisterJob(configJob); err != nil {
		log.Fatalf("Failed to register config sync job: %v", err)
	}

	sportsJob := jobs.NewSportsSyncJob(sportsService)
	if err := jobManager.RegisterJob(sportsJob); err != nil {
		log.Fatalf("Failed to register sports sync job: %v", err)
	}

	eventsJob := jobs.NewEventsSyncJob(iddaaClient, eventsService)
	if err := jobManager.RegisterJob(eventsJob); err != nil {
		log.Fatalf("Failed to register events sync job: %v", err)
	}

	// Register volume sync job for all sports
	volumeJob := jobs.NewVolumeSyncJob(volumeService, queries)
	if err := jobManager.RegisterJob(volumeJob); err != nil {
		log.Fatalf("Failed to register volume sync job: %v", err)
	}

	// Register distribution sync job for all sports
	distributionJob := jobs.NewDistributionSyncJob(distributionService, queries)
	if err := jobManager.RegisterJob(distributionJob); err != nil {
		log.Fatalf("Failed to register distribution sync job: %v", err)
	}

	// Register analytics refresh job
	analyticsJob := jobs.NewAnalyticsRefreshJob(queries)
	if err := jobManager.RegisterJob(analyticsJob); err != nil {
		log.Fatalf("Failed to register analytics refresh job: %v", err)
	}

	// Register market config sync job
	marketConfigJob := jobs.NewMarketConfigSyncJob(marketConfigService)
	if err := jobManager.RegisterJob(marketConfigJob); err != nil {
		log.Fatalf("Failed to register market config sync job: %v", err)
	}

	// Register statistics sync job for football (sport type 1)
	statisticsJob := jobs.NewStatisticsSyncJob(statisticsService, 1)
	if err := jobManager.RegisterJob(statisticsJob); err != nil {
		log.Fatalf("Failed to register statistics sync job: %v", err)
	}

	// Register leagues sync job for Iddaa and Football API integration
	leaguesJob := jobs.NewLeaguesSyncJob(queries, iddaaClient)
	if err := jobManager.RegisterJob(leaguesJob); err != nil {
		log.Fatalf("Failed to register leagues sync job: %v", err)
	}

	// Register detailed odds sync job for high-frequency odds tracking
	detailedOddsJob := jobs.NewDetailedOddsSyncJob(queries, iddaaClient, eventsService)
	if err := jobManager.RegisterJob(detailedOddsJob); err != nil {
		log.Fatalf("Failed to register detailed odds sync job: %v", err)
	}

	// Register API-Football league matching job (independent of Iddaa sync)
	apiFootballLeagueMatchingJob := jobs.NewAPIFootballLeagueMatchingJob(queries)
	if err := jobManager.RegisterJob(apiFootballLeagueMatchingJob); err != nil {
		log.Fatalf("Failed to register API-Football league matching job: %v", err)
	}

	// Register API-Football team matching job (independent of Iddaa sync)
	apiFootballTeamMatchingJob := jobs.NewAPIFootballTeamMatchingJob(queries)
	if err := jobManager.RegisterJob(apiFootballTeamMatchingJob); err != nil {
		log.Fatalf("Failed to register API-Football team matching job: %v", err)
	}

	// Register API-Football league enrichment job
	apiFootballLeagueEnrichmentJob := jobs.NewAPIFootballLeagueEnrichmentJob(queries)
	if err := jobManager.RegisterJob(apiFootballLeagueEnrichmentJob); err != nil {
		log.Fatalf("Failed to register API-Football league enrichment job: %v", err)
	}

	// Register API-Football team enrichment job
	apiFootballTeamEnrichmentJob := jobs.NewAPIFootballTeamEnrichmentJob(queries)
	if err := jobManager.RegisterJob(apiFootballTeamEnrichmentJob); err != nil {
		log.Fatalf("Failed to register API-Football team enrichment job: %v", err)
	}

	// Register Smart Money Processor job
	smartMoneyProcessorJob := jobs.NewSmartMoneyProcessorJob(queries, smartMoneyTracker)
	if err := jobManager.RegisterJob(smartMoneyProcessorJob); err != nil {
		log.Fatalf("Failed to register Smart Money Processor job: %v", err)
	}

	// Handle single job execution
	if *once && *jobName != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		switch *jobName {
		case "config":
			log.Println("Running config sync job once...")
			if err := configJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute config job: %v", err)
			}
			log.Println("Config sync completed successfully")
		case "sports":
			log.Println("Running sports sync job once...")
			if err := sportsJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute sports job: %v", err)
			}
			log.Println("Sports sync completed successfully")
		case "events":
			log.Println("Running events sync job once...")
			if err := eventsJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute events job: %v", err)
			}
			log.Println("Events sync completed successfully")
		case "volume":
			log.Println("Running volume sync job once...")
			if err := volumeJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute volume job: %v", err)
			}
			log.Println("Volume sync completed successfully")
		case "distribution":
			log.Println("Running distribution sync job once...")
			if err := distributionJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute distribution job: %v", err)
			}
			log.Println("Distribution sync completed successfully")
		case "analytics":
			log.Println("Running analytics refresh job once...")
			if err := analyticsJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute analytics job: %v", err)
			}
			log.Println("Analytics refresh completed successfully")
		case "market_config":
			log.Println("Running market config sync job once...")
			if err := marketConfigJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute market config job: %v", err)
			}
			log.Println("Market config sync completed successfully")
		case "statistics":
			log.Println("Running statistics sync job once...")
			if err := statisticsJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute statistics job: %v", err)
			}
			log.Println("Statistics sync completed successfully")
		case "leagues":
			log.Println("Running leagues sync job once...")
			if err := leaguesJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute leagues job: %v", err)
			}
			log.Println("Leagues sync completed successfully")
		case "detailed_odds":
			log.Println("Running detailed odds sync job once...")
			if err := detailedOddsJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute detailed odds job: %v", err)
			}
			log.Println("Detailed odds sync completed successfully")
		case "api_football_league_matching":
			log.Println("Running API-Football league matching job once...")
			if err := apiFootballLeagueMatchingJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute API-Football league matching job: %v", err)
			}
			log.Println("API-Football league matching completed successfully")
		case "api_football_team_matching":
			log.Println("Running API-Football team matching job once...")
			if err := apiFootballTeamMatchingJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute API-Football team matching job: %v", err)
			}
			log.Println("API-Football team matching completed successfully")
		case "api_football_league_enrichment":
			log.Println("Running API-Football league enrichment job once...")
			if err := apiFootballLeagueEnrichmentJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute API-Football league enrichment job: %v", err)
			}
			log.Println("API-Football league enrichment completed successfully")
		case "api_football_team_enrichment":
			log.Println("Running API-Football team enrichment job once...")
			if err := apiFootballTeamEnrichmentJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute API-Football team enrichment job: %v", err)
			}
			log.Println("API-Football team enrichment completed successfully")
		case "smart_money_processor":
			log.Println("Running Smart Money Processor job once...")
			if err := smartMoneyProcessorJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute Smart Money Processor job: %v", err)
			}
			log.Println("Smart Money Processor completed successfully")
		default:
			log.Fatalf("Unknown job: %s. Available jobs: config, sports, events, volume, distribution, analytics, market_config, statistics, leagues, detailed_odds, api_football_league_matching, api_football_team_matching, api_football_league_enrichment, api_football_team_enrichment, smart_money_processor", *jobName)
		}
		return
	}

	// Start job manager
	jobManager.Start()
	log.Info().
		Str("action", "service_started").
		Int("job_count", len(jobManager.GetJobs())).
		Msg("Cron job service started")

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().
		Str("action", "shutdown_initiated").
		Msg("Shutting down cron job service")

	jobManager.Stop()

	log.Info().
		Str("action", "service_stopped").
		Msg("Cron job service stopped")
}
