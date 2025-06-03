package jobs

import (
	"context"
	"log"

	"github.com/betslib/iddaa-core/pkg/services"
)

type VolumeSyncJob struct {
	volumeService *services.VolumeService
	sportType     int
}

func NewVolumeSyncJob(volumeService *services.VolumeService, sportType int) Job {
	return &VolumeSyncJob{
		volumeService: volumeService,
		sportType:     sportType,
	}
}

func (j *VolumeSyncJob) Name() string {
	return "volume_sync"
}

func (j *VolumeSyncJob) Execute(ctx context.Context) error {
	log.Printf("Starting volume sync job for sport type %d...", j.sportType)

	err := j.volumeService.FetchAndUpdateVolumes(ctx, j.sportType)
	if err != nil {
		return err
	}

	log.Printf("Volume sync completed successfully")
	return nil
}

func (j *VolumeSyncJob) Schedule() string {
	// Run every 20 minutes to track volume changes
	return "*/20 * * * *"
}
