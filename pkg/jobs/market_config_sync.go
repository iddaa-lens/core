package jobs

import (
	"context"
	"time"

	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/services"
)

type MarketConfigSyncJob struct {
	marketConfigService *services.MarketConfigService
}

func NewMarketConfigSyncJob(marketConfigService *services.MarketConfigService) Job {
	return &MarketConfigSyncJob{
		marketConfigService: marketConfigService,
	}
}

func (j *MarketConfigSyncJob) Name() string {
	return "market_config_sync"
}

func (j *MarketConfigSyncJob) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx, "market-config-sync")
	start := time.Now()

	log.Info().
		Str("action", "sync_start").
		Msg("Starting market config sync job")

	err := j.marketConfigService.SyncMarketConfigs(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Str("action", "sync_failed").
			Dur("duration", time.Since(start)).
			Msg("Market config sync failed")
		return err
	}

	duration := time.Since(start)
	log.LogJobComplete("market_config_sync", duration, 1, 0)
	return nil
}

func (j *MarketConfigSyncJob) Schedule() string {
	// Run every 15 minutes to sync market configurations
	return "*/15 * * * *"
}
