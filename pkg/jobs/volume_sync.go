package jobs

import (
	"context"
	"fmt"
	"log"

	"github.com/betslib/iddaa-core/pkg/database"
	"github.com/betslib/iddaa-core/pkg/services"
)

type VolumeSyncJob struct {
	volumeService *services.VolumeService
	db            *database.Queries
}

func NewVolumeSyncJob(volumeService *services.VolumeService, db *database.Queries) Job {
	return &VolumeSyncJob{
		volumeService: volumeService,
		db:            db,
	}
}

func (j *VolumeSyncJob) Name() string {
	return "volume_sync"
}

func (j *VolumeSyncJob) Execute(ctx context.Context) error {
	log.Printf("Starting volume sync job...")

	// Fetch all active sports from database
	sports, err := j.db.ListSports(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch sports from database: %w", err)
	}

	if len(sports) == 0 {
		log.Printf("No active sports found in database")
		return nil
	}

	log.Printf("Found %d active sports to sync volume data for", len(sports))
	totalProcessed := 0

	for _, sport := range sports {
		log.Printf("Fetching volume data for sport %s (ID: %d)...", sport.Name, sport.ID)

		err := j.volumeService.FetchAndUpdateVolumes(ctx, int(sport.ID))
		if err != nil {
			log.Printf("Failed to fetch volume data for sport %s (ID: %d): %v", sport.Name, sport.ID, err)
			continue // Continue with other sports
		}

		log.Printf("Volume sync completed for sport %s (ID: %d)", sport.Name, sport.ID)
		totalProcessed++
	}

	log.Printf("Volume sync completed successfully for %d/%d sports", totalProcessed, len(sports))
	return nil
}

func (j *VolumeSyncJob) Schedule() string {
	// Run every 20 minutes to track volume changes
	return "*/20 * * * *"
}
