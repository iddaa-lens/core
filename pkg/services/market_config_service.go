package services

import (
	"context"
	"fmt"
	"log"

	"github.com/betslib/iddaa-core/pkg/database"
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
	log.Println("Starting market config sync...")

	resp, err := s.client.GetMarketConfig()
	if err != nil {
		return fmt.Errorf("failed to fetch market configs: %w", err)
	}

	log.Printf("Fetched %d market configs from API", len(resp.Data.Markets))

	synced := 0
	for marketKey, config := range resp.Data.Markets {
		if err := s.saveMarketConfig(ctx, marketKey, config); err != nil {
			log.Printf("Failed to save market config %s (ID: %d): %v", marketKey, config.ID, err)
			continue
		}
		synced++
	}

	log.Printf("Market config sync completed. Synced %d out of %d configs", synced, len(resp.Data.Markets))
	return nil
}

func (s *MarketConfigService) saveMarketConfig(ctx context.Context, marketKey string, config models.IddaaMarketConfig) error {
	// Generate market type code from market key and subtype
	// Market key format is like "2_821" where 2 is market type and 821 is subtype
	marketTypeCode := fmt.Sprintf("MST_%d", config.MarketSubType)

	// Use the Turkish name and description from the API
	params := database.UpsertMarketTypeParams{
		Code:        marketTypeCode,
		Name:        config.Name, // Turkish name
		Description: pgtype.Text{String: config.Description, Valid: config.Description != ""},
	}

	_, err := s.db.UpsertMarketType(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to upsert market config: %w", err)
	}

	log.Printf("Synced market config: %s - %s (SubType: %d)", marketKey, config.Name, config.MarketSubType)
	return nil
}
