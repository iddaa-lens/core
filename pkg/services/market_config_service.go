package services

import (
	"context"
	"fmt"

	"github.com/gosimple/slug"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
)

type MarketConfigService struct {
	db     *generated.Queries
	client *IddaaClient
}

func NewMarketConfigService(db *generated.Queries, client *IddaaClient) *MarketConfigService {
	return &MarketConfigService{
		db:     db,
		client: client,
	}
}

func (s *MarketConfigService) SyncMarketConfigs(ctx context.Context) error {
	log := logger.WithContext(ctx, "market-config-sync")

	log.Info().
		Str("action", "sync_start").
		Msg("Starting market config sync")

	resp, err := s.client.GetMarketConfig()
	if err != nil {
		log.Error().
			Err(err).
			Str("action", "fetch_failed").
			Msg("Failed to fetch market configs")
		return fmt.Errorf("failed to fetch market configs: %w", err)
	}

	log.Info().
		Int("config_count", len(resp.Data.Markets)).
		Str("action", "configs_fetched").
		Msg("Fetched market configs from API")

	// Prepare batch arrays
	var (
		codes                  []string
		names                  []string
		slugs                  []string
		descriptions           []string
		iddaaMarketIDs         []int32
		isLives                []bool
		marketTypes            []int32
		minMarketDefaultValues []int32
		maxMarketLimitValues   []int32
		priorities             []int32
		sportTypes             []int32
		marketSubTypes         []int32
		minDefaultValues       []int32
		maxLimitValues         []int32
		isActives              []bool
	)

	// Collect all market configs into arrays
	for marketKey, config := range resp.Data.Markets {
		// Use the market key as the code (e.g., "2_821") to match the format used in events processing
		marketTypeCode := marketKey

		// Generate slug from the Turkish name and make it unique with market key
		marketSlug := slug.Make(fmt.Sprintf("%s-%s", config.Name, marketKey))

		// Add to arrays
		codes = append(codes, marketTypeCode)
		names = append(names, config.Name)
		slugs = append(slugs, marketSlug)
		descriptions = append(descriptions, config.Description)
		iddaaMarketIDs = append(iddaaMarketIDs, int32(config.ID))
		isLives = append(isLives, config.IsLive)
		marketTypes = append(marketTypes, int32(config.MarketType))
		minMarketDefaultValues = append(minMarketDefaultValues, int32(config.MinMarketValue))
		maxMarketLimitValues = append(maxMarketLimitValues, int32(config.MaxMarketValue))
		priorities = append(priorities, int32(config.Priority))
		sportTypes = append(sportTypes, int32(config.SportType))
		marketSubTypes = append(marketSubTypes, int32(config.MarketSubType))
		minDefaultValues = append(minDefaultValues, int32(config.MinValue))
		maxLimitValues = append(maxLimitValues, int32(config.MaxValue))
		isActives = append(isActives, config.IsActive)
	}

	// Perform bulk upsert
	if len(codes) > 0 {
		err = s.db.BulkUpsertMarketTypes(ctx, generated.BulkUpsertMarketTypesParams{
			Codes:                  codes,
			Names:                  names,
			Slugs:                  slugs,
			Descriptions:           descriptions,
			IddaaMarketIds:         iddaaMarketIDs,
			IsLives:                isLives,
			MarketTypes:            marketTypes,
			MinMarketDefaultValues: minMarketDefaultValues,
			MaxMarketLimitValues:   maxMarketLimitValues,
			Priorities:             priorities,
			SportTypes:             sportTypes,
			MarketSubTypes:         marketSubTypes,
			MinDefaultValues:       minDefaultValues,
			MaxLimitValues:         maxLimitValues,
			IsActives:              isActives,
		})

		if err != nil {
			log.Error().
				Err(err).
				Str("action", "bulk_upsert_failed").
				Int("market_count", len(codes)).
				Msg("Failed to bulk upsert market configs")
			return fmt.Errorf("failed to bulk upsert market configs: %w", err)
		}
	}

	log.Info().
		Int("synced_count", len(codes)).
		Int("total_count", len(resp.Data.Markets)).
		Str("action", "sync_complete").
		Msg("Market config sync completed with bulk upsert")
	return nil
}
