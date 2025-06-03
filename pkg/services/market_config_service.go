package services

import (
	"context"
	"fmt"
	"log"
	"strconv"

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

	log.Printf("Fetched %d market configs from API", len(resp.Data))

	for _, config := range resp.Data {
		if err := s.saveMarketConfig(ctx, config); err != nil {
			log.Printf("Failed to save market config %d: %v", config.ID, err)
			continue
		}
	}

	log.Println("Market config sync completed")
	return nil
}

func (s *MarketConfigService) saveMarketConfig(ctx context.Context, config models.IddaaMarketConfig) error {
	// Use external ID as code and name as description
	params := database.UpsertMarketTypeByExternalIDParams{
		ExternalID:  strconv.Itoa(config.ID),
		Name:        config.Name,
		Description: pgtype.Text{String: config.Description, Valid: config.Description != ""},
	}

	_, err := s.db.UpsertMarketTypeByExternalID(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to upsert market config: %w", err)
	}

	return nil
}
