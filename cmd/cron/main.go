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

	"github.com/betslib/iddaa-core/internal/config"
	"github.com/betslib/iddaa-core/pkg/database"
	"github.com/betslib/iddaa-core/pkg/jobs"
	"github.com/betslib/iddaa-core/pkg/logger"
	"github.com/betslib/iddaa-core/pkg/services"
)

func main() {
	// Parse command line flags
	var (
		jobName = flag.String("job", "", "Run specific job once (config, sports, events, volume, distribution, analytics, market_config, statistics, leagues, detailed_odds)")
		once    = flag.Bool("once", false, "Run job once and exit")
	)
	flag.Parse()

	// Setup structured logging
	logger.SetupLogger()
	log := logger.New("cron-service")

	cfg := config.Load()

	// Connect to database
	db, err := pgxpool.New(context.Background(), cfg.DatabaseURL())
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
		default:
			log.Fatalf("Unknown job: %s. Available jobs: config, sports, events, volume, distribution, analytics, market_config, statistics, leagues, detailed_odds", *jobName)
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
