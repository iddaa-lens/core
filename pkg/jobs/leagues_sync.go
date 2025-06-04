package jobs

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/betslib/iddaa-core/pkg/database"
	"github.com/betslib/iddaa-core/pkg/services"
)

type LeaguesSyncJob struct {
	db             *database.Queries
	leaguesService *services.LeaguesService
}

func NewLeaguesSyncJob(db *database.Queries, iddaaClient *services.IddaaClient) *LeaguesSyncJob {
	// Get Football API key from environment
	apiKey := os.Getenv("FOOTBALL_API_KEY")
	if apiKey == "" {
		log.Printf("Warning: FOOTBALL_API_KEY not set, Football API sync will be disabled")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create leagues service
	leaguesService := services.NewLeaguesService(db, client, apiKey, iddaaClient)

	return &LeaguesSyncJob{
		db:             db,
		leaguesService: leaguesService,
	}
}

func (j *LeaguesSyncJob) Name() string {
	return "leagues_sync"
}

func (j *LeaguesSyncJob) Description() string {
	return "Syncs leagues and teams with Football API"
}

func (j *LeaguesSyncJob) Schedule() string {
	return "0 2 * * *" // Daily at 2 AM
}

func (j *LeaguesSyncJob) Execute(ctx context.Context) error {
	log.Printf("Starting leagues sync from Iddaa and Football API")
	startTime := time.Now()

	// Step 1: Sync leagues from Iddaa competitions endpoint first
	log.Printf("Syncing leagues from Iddaa...")
	if err := j.leaguesService.SyncLeaguesFromIddaa(ctx); err != nil {
		log.Printf("Error syncing leagues from Iddaa: %v", err)
		return fmt.Errorf("failed to sync Iddaa leagues: %w", err)
	}

	// Step 2: Check if Football API key is available for additional enrichment
	apiKey := os.Getenv("FOOTBALL_API_KEY")
	if apiKey == "" {
		log.Printf("FOOTBALL_API_KEY not set, skipping Football API sync")
		duration := time.Since(startTime)
		log.Printf("Iddaa leagues sync completed successfully in %v", duration)
		return nil
	}

	// Step 3: Sync leagues with Football API for mapping
	log.Printf("Syncing leagues with Football API...")
	if err := j.leaguesService.SyncLeaguesWithFootballAPI(ctx); err != nil {
		log.Printf("Error syncing leagues with Football API: %v", err)
		// Don't fail the entire job, Iddaa sync was successful
	}

	// Step 4: Sync teams (only for leagues that are already mapped)
	log.Printf("Syncing teams with Football API...")
	if err := j.leaguesService.SyncTeamsWithFootballAPI(ctx); err != nil {
		log.Printf("Error syncing teams with Football API: %v", err)
		// Don't fail the entire job, Iddaa sync was successful
	}

	duration := time.Since(startTime)
	log.Printf("Complete leagues sync completed successfully in %v", duration)

	return nil
}

func (j *LeaguesSyncJob) Timeout() time.Duration {
	return 10 * time.Minute // Allow up to 10 minutes for sync
}
