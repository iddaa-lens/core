package jobs

import (
	"context"
	"time"

	"github.com/betslib/iddaa-core/pkg/logger"
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
	log := logger.WithContext(ctx, "config-sync")
	start := time.Now()

	log.Info().
		Str("action", "sync_start").
		Str("platform", j.platform).
		Msg("Starting config sync job")

	err := j.configService.SyncConfig(ctx, j.platform)
	duration := time.Since(start)

	if err != nil {
		log.Error().
			Err(err).
			Str("action", "sync_failed").
			Str("platform", j.platform).
			Dur("duration", duration).
			Msg("Config sync failed")
		return err
	}

	log.LogJobComplete("config_sync", duration, 1, 0)
	return nil
}

func (j *ConfigSyncJob) Name() string {
	return "Config Sync (" + j.platform + ")"
}

func (j *ConfigSyncJob) Schedule() string {
	// Run weekly on Mondays at 6 AM - config changes very infrequently
	return "0 6 * * 1"
}
