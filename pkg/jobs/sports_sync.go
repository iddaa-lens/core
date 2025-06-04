package jobs

import (
	"context"

	"github.com/betslib/iddaa-core/pkg/services"
)

type SportsSyncJob struct {
	sportsService *services.SportService
}

// NewSportsSyncJob creates a new sports sync job
func NewSportsSyncJob(sportsService *services.SportService) Job {
	return &SportsSyncJob{
		sportsService: sportsService,
	}
}

func (j *SportsSyncJob) Execute(ctx context.Context) error {
	return j.sportsService.SyncSports(ctx)
}

func (j *SportsSyncJob) Name() string {
	return "sports_sync"
}

func (j *SportsSyncJob) Schedule() string {
	// Run every 30 minutes to keep sports info up to date
	return "*/30 * * * *"
}
