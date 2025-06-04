package jobs

import (
	"context"
	"fmt"
	"log"

	"github.com/betslib/iddaa-core/pkg/database"
	"github.com/betslib/iddaa-core/pkg/services"
)

type DistributionSyncJob struct {
	distributionService *services.DistributionService
	db                  *database.Queries
}

func NewDistributionSyncJob(distributionService *services.DistributionService, db *database.Queries) Job {
	return &DistributionSyncJob{
		distributionService: distributionService,
		db:                  db,
	}
}

func (j *DistributionSyncJob) Name() string {
	return "distribution_sync"
}

func (j *DistributionSyncJob) Execute(ctx context.Context) error {
	log.Printf("Starting distribution sync job...")

	// Fetch all active sports from database
	sports, err := j.db.ListSports(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch sports from database: %w", err)
	}

	if len(sports) == 0 {
		log.Printf("No active sports found in database")
		return nil
	}

	log.Printf("Found %d active sports to sync distribution data for", len(sports))
	totalProcessed := 0

	for _, sport := range sports {
		log.Printf("Fetching distribution data for sport %s (ID: %d)...", sport.Name, sport.ID)

		err := j.distributionService.FetchAndUpdateDistributions(ctx, int(sport.ID))
		if err != nil {
			log.Printf("Failed to fetch distribution data for sport %s (ID: %d): %v", sport.Name, sport.ID, err)
			continue // Continue with other sports
		}

		log.Printf("Distribution sync completed for sport %s (ID: %d)", sport.Name, sport.ID)
		totalProcessed++
	}

	log.Printf("Distribution sync completed successfully for %d/%d sports", totalProcessed, len(sports))
	return nil
}

func (j *DistributionSyncJob) Schedule() string {
	// Run every hour to track betting distribution changes
	return "0 * * * *"
}
