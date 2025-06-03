package jobs

import (
	"context"
	"log"

	"github.com/betslib/iddaa-core/pkg/services"
)

type DistributionSyncJob struct {
	distributionService *services.DistributionService
	sportType           int
}

func NewDistributionSyncJob(distributionService *services.DistributionService, sportType int) Job {
	return &DistributionSyncJob{
		distributionService: distributionService,
		sportType:           sportType,
	}
}

func (j *DistributionSyncJob) Name() string {
	return "distribution_sync"
}

func (j *DistributionSyncJob) Execute(ctx context.Context) error {
	log.Printf("Starting distribution sync job for sport type %d...", j.sportType)

	err := j.distributionService.FetchAndUpdateDistributions(ctx, j.sportType)
	if err != nil {
		return err
	}

	log.Printf("Distribution sync completed successfully")
	return nil
}

func (j *DistributionSyncJob) Schedule() string {
	// Run every hour to track betting distribution changes
	return "0 * * * *"
}
