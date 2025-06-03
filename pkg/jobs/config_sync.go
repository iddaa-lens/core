package jobs

import (
	"context"

	"github.com/betslib/iddaa-core/pkg/services"
)

type ConfigSyncJob struct {
	configService *services.ConfigService
	platform      string
}

// NewConfigSyncJob creates a new config sync job
func NewConfigSyncJob(configService *services.ConfigService, platform string) Job {
	return &ConfigSyncJob{
		configService: configService,
		platform:      platform,
	}
}

func (j *ConfigSyncJob) Execute(ctx context.Context) error {
	return j.configService.SyncConfig(ctx, j.platform)
}

func (j *ConfigSyncJob) Name() string {
	return "Config Sync (" + j.platform + ")"
}

func (j *ConfigSyncJob) Schedule() string {
	// Run weekly on Mondays at 6 AM - config changes very infrequently
	return "0 6 * * 1"
}
