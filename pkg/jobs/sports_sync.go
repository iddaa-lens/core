package jobs

import (
	"context"

	"github.com/iddaa-lens/core/pkg/services"
)

type SportsSyncJob struct {
	sportsService *services.SportService
}

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
	// Run every hour - sports data doesn't change frequently
	return "0 * * * *"
}
