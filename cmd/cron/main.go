package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/iddaa-lens/core/internal/config"
	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/database/pool"
	"github.com/iddaa-lens/core/pkg/jobs"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/services"
)

func main() {
	// Load .env file if it exists
	envPath := filepath.Join(".", ".env")
	if _, err := os.Stat(envPath); err == nil {
		if err := godotenv.Load(envPath); err != nil {
			// Log but don't fail - env vars might be set elsewhere
			logger.New("cron-service").Warn().
				Err(err).
				Str("path", envPath).
				Msg("Failed to load .env file")
		}
	}
	// Parse command line flags
	var (
		jobName           = flag.String("job", "", "Run specific job once (config, sports, events, volume, distribution, analytics, market_config, statistics, leagues, detailed_odds, api_football_league_matching, api_football_team_matching, api_football_league_enrichment, api_football_team_enrichment, smart_money_processor)")
		once              = flag.Bool("once", false, "Run job once and exit")
		healthCheck       = flag.Bool("health-check", false, "Perform health check and exit")
		useProductionMode = flag.Bool("production-mode", false, "Use production job manager with distributed locking")
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
	// Use Azure config for cron to be conservative with connections
	poolConfig := pool.AzureConfig()
	db, err := pool.New(context.Background(), cfg.DatabaseURL(), poolConfig)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("action", "db_connect_failed").
			Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize services
	queries := generated.New(db)
	iddaaClient := services.NewIddaaClient(cfg)
	configService := services.NewConfigService(queries, iddaaClient)
	sportsService := services.NewSportService(queries, iddaaClient)
	eventsService := services.NewEventsService(queries, iddaaClient)
	volumeService := services.NewVolumeService(queries, iddaaClient)
	distributionService := services.NewDistributionService(queries, iddaaClient)
	marketConfigService := services.NewMarketConfigService(queries, iddaaClient)
	statisticsService := services.NewStatisticsService(queries, iddaaClient)
	smartMoneyTracker := services.NewSmartMoneyTracker(queries)

	// Create job manager (production or standard based on flag)
	var jobManager jobs.JobManager
	if *useProductionMode {
		log.Info().
			Str("action", "production_mode_enabled").
			Msg("Using production job manager with distributed locking")

		// Create production job manager with distributed locking
		jobManager = jobs.NewProductionJobManager(db, &jobs.ProductionJobManagerConfig{
			EnableLocking: true,
			DefaultConfig: jobs.DefaultProductionJobConfig(),
		})
	} else {
		log.Info().
			Str("action", "standard_mode").
			Msg("Using standard job manager (no distributed locking)")
		jobManager = jobs.NewJobManager()
	}

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

	// Register API-Football league matching job (optimized version)
	apiFootballLeagueMatchingJob := jobs.NewAPIFootballLeagueMatchingJobV2(queries)
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
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		// Find the job by name from registered jobs
		var targetJob jobs.Job
		jobNameMapping := map[string]string{
			"config":                         "Config Sync (WEB)",
			"sports":                         "sports_sync",
			"events":                         "events_sync",
			"volume":                         "volume_sync",
			"distribution":                   "distribution_sync",
			"analytics":                      "analytics_refresh",
			"market_config":                  "market_config_sync",
			"statistics":                     "statistics_sync",
			"leagues":                        "leagues_sync",
			"detailed_odds":                  "detailed_odds",
			"api_football_league_matching":   "api_football_league_matching",
			"api_football_team_matching":     "api_football_team_matching",
			"api_football_league_enrichment": "api_football_league_enrichment",
			"api_football_team_enrichment":   "api_football_team_enrichment",
			"smart_money_processor":          "smart_money_processor",
		}

		actualJobName, exists := jobNameMapping[*jobName]
		if !exists {
			log.Fatalf("Unknown job name: %s", *jobName)
		}

		// Find the registered job
		for _, job := range jobManager.GetJobs() {
			if job.Name() == actualJobName {
				targetJob = job
				break
			}
		}

		if targetJob == nil {
			log.Fatalf("Job not found: %s (mapped to %s)", *jobName, actualJobName)
		}

		log.Printf("Running %s job once...", *jobName)
		if err := targetJob.Execute(ctx); err != nil {
			log.Fatalf("Failed to execute %s job: %v", *jobName, err)
		}
		log.Printf("%s completed successfully", *jobName)
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
