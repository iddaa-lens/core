package jobs

import (
	"context"

	"github.com/betslib/iddaa-core/pkg/services"
)

type CompetitionsSyncJob struct {
	competitionService services.CompetitionSyncer
}

// NewCompetitionsSyncJob creates a new competitions sync job
func NewCompetitionsSyncJob(competitionService services.CompetitionSyncer) Job {
	return &CompetitionsSyncJob{
		competitionService: competitionService,
	}
}

func (j *CompetitionsSyncJob) Execute(ctx context.Context) error {
	return j.competitionService.SyncCompetitions(ctx)
}

func (j *CompetitionsSyncJob) Name() string {
	return "Competitions Sync"
}

func (j *CompetitionsSyncJob) Schedule() string {
	// Run daily at 8 AM
	return "0 8 * * *"
}
