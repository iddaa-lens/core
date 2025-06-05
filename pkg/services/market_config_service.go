package services

import (
	"context"
	"fmt"

	"github.com/betslib/iddaa-core/pkg/database"
	"github.com/betslib/iddaa-core/pkg/logger"
	"github.com/betslib/iddaa-core/pkg/models"
	"github.com/jackc/pgx/v5/pgtype"
)

type MarketConfigService struct {
	db     *database.Queries
	client *IddaaClient
}

func NewMarketConfigService(db *database.Queries, client *IddaaClient) *MarketConfigService {
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

	synced := 0
	errors := 0
	for marketKey, config := range resp.Data.Markets {
		if err := s.saveMarketConfig(ctx, marketKey, config); err != nil {
			errors++
			log.Error().
				Err(err).
				Str("action", "save_failed").
				Str("market_key", marketKey).
				Int("market_id", config.ID).
				Msg("Failed to save market config")
			continue
		}
		synced++
	}

	log.Info().
		Int("synced_count", synced).
		Int("total_count", len(resp.Data.Markets)).
		Int("error_count", errors).
		Str("action", "sync_complete").
		Msg("Market config sync completed")
	return nil
}

func (s *MarketConfigService) saveMarketConfig(ctx context.Context, marketKey string, config models.IddaaMarketConfig) error {
	// Generate market type code from market key and subtype
	// Market key format is like "2_821" where 2 is market type and 821 is subtype
	marketTypeCode := fmt.Sprintf("MST_%d", config.MarketSubType)

	// Use all fields from the API
	params := database.UpsertMarketTypeParams{
		Code:                  marketTypeCode,
		Name:                  config.Name,
		Description:           pgtype.Text{String: config.Description, Valid: config.Description != ""},
		IddaaMarketID:         pgtype.Int4{Int32: int32(config.ID), Valid: true},
		IsLive:                pgtype.Bool{Bool: config.IsLive, Valid: true},
		MarketType:            pgtype.Int4{Int32: int32(config.MarketType), Valid: true},
		MinMarketDefaultValue: pgtype.Int4{Int32: int32(config.MinMarketValue), Valid: true},
		MaxMarketLimitValue:   pgtype.Int4{Int32: int32(config.MaxMarketValue), Valid: true},
		Priority:              pgtype.Int4{Int32: int32(config.Priority), Valid: true},
		SportType:             pgtype.Int4{Int32: int32(config.SportType), Valid: true},
		MarketSubType:         pgtype.Int4{Int32: int32(config.MarketSubType), Valid: true},
		MinDefaultValue:       pgtype.Int4{Int32: int32(config.MinValue), Valid: true},
		MaxLimitValue:         pgtype.Int4{Int32: int32(config.MaxValue), Valid: true},
		IsActive:              pgtype.Bool{Bool: config.IsActive, Valid: true},
	}

	_, err := s.db.UpsertMarketType(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to upsert market config: %w", err)
	}

	log := logger.WithContext(ctx, "market-config-sync")
	log.Debug().
		Str("action", "config_saved").
		Str("market_key", marketKey).
		Str("market_name", config.Name).
		Int("market_id", config.ID).
		Int("market_subtype", config.MarketSubType).
		Bool("is_active", config.IsActive).
		Msg("Synced market config")
	return nil
}
