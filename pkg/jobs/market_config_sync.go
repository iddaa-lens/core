package jobs

import (
	"context"
	"log"

	"github.com/betslib/iddaa-core/pkg/services"
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
	log.Printf("Starting market config sync job...")

	err := j.marketConfigService.SyncMarketConfigs(ctx)
	if err != nil {
		return err
	}

	log.Printf("Market config sync completed successfully")
	return nil
}

func (j *MarketConfigSyncJob) Schedule() string {
	// Run daily at 6 AM to sync market configurations
	return "0 6 * * *"
}
