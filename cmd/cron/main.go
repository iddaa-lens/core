package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"

	"github.com/betslib/iddaa-core/internal/config"
	"github.com/betslib/iddaa-core/pkg/database"
	"github.com/betslib/iddaa-core/pkg/jobs"
	"github.com/betslib/iddaa-core/pkg/services"
)

func main() {
	// Parse command line flags
	var (
		jobName = flag.String("job", "", "Run specific job once (competitions, config, events, volume, distribution, analytics, market_config, statistics)")
		once    = flag.Bool("once", false, "Run job once and exit")
	)
	flag.Parse()

	cfg := config.Load()

	// Connect to database
	db, err := pgxpool.New(context.Background(), cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize services
	queries := database.New(db)
	iddaaClient := services.NewIddaaClient(cfg)
	competitionService := services.NewCompetitionService(queries, iddaaClient)
	configService := services.NewConfigService(queries, iddaaClient)
	eventsService := services.NewEventsService(queries, iddaaClient)
	volumeService := services.NewVolumeService(queries, iddaaClient)
	distributionService := services.NewDistributionService(queries, iddaaClient)
	marketConfigService := services.NewMarketConfigService(queries, iddaaClient)
	statisticsService := services.NewStatisticsService(queries, iddaaClient)

	// Create job manager
	jobManager := jobs.NewJobManager()

	// Register jobs
	competitionsJob := jobs.NewCompetitionsSyncJob(competitionService)
	if err := jobManager.RegisterJob(competitionsJob); err != nil {
		log.Fatalf("Failed to register competitions sync job: %v", err)
	}

	configJob := jobs.NewConfigSyncJob(configService, "WEB")
	if err := jobManager.RegisterJob(configJob); err != nil {
		log.Fatalf("Failed to register config sync job: %v", err)
	}

	eventsJob := jobs.NewEventsSyncJob(iddaaClient, eventsService)
	if err := jobManager.RegisterJob(eventsJob); err != nil {
		log.Fatalf("Failed to register events sync job: %v", err)
	}

	// Register volume sync job for football (sport type 1)
	volumeJob := jobs.NewVolumeSyncJob(volumeService, 1)
	if err := jobManager.RegisterJob(volumeJob); err != nil {
		log.Fatalf("Failed to register volume sync job: %v", err)
	}

	// Register distribution sync job for football (sport type 1)
	distributionJob := jobs.NewDistributionSyncJob(distributionService, 1)
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

	// Handle single job execution
	if *once && *jobName != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		switch *jobName {
		case "competitions":
			log.Println("Running competitions sync job once...")
			if err := competitionsJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute competitions job: %v", err)
			}
			log.Println("Competitions sync completed successfully")
		case "config":
			log.Println("Running config sync job once...")
			if err := configJob.Execute(ctx); err != nil {
				log.Fatalf("Failed to execute config job: %v", err)
			}
			log.Println("Config sync completed successfully")
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
		default:
			log.Fatalf("Unknown job: %s. Available jobs: competitions, config, events, volume, distribution, analytics, market_config, statistics", *jobName)
		}
		return
	}

	// Run initial competitions sync on startup
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		log.Println("Running initial competitions sync...")
		if err := competitionsJob.Execute(ctx); err != nil {
			log.Printf("Failed to sync competitions on startup: %v", err)
		} else {
			log.Println("Initial competitions sync completed")
		}
	}()

	// Start job manager
	jobManager.Start()
	log.Printf("Cron job service started with %d jobs", len(jobManager.GetJobs()))

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down cron job service...")
	jobManager.Stop()
	log.Println("Cron job service stopped")
}
