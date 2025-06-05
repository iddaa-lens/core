package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/models"
)

type ConfigService struct {
	db     *database.Queries
	client *IddaaClient
}

func NewConfigService(db *database.Queries, client *IddaaClient) *ConfigService {
	return &ConfigService{
		db:     db,
		client: client,
	}
}

func (s *ConfigService) SyncConfig(ctx context.Context, platform string) error {
	log.Printf("Starting config sync for platform: %s", platform)

	resp, err := s.client.GetAppConfig(platform)
	if err != nil {
		return fmt.Errorf("failed to fetch config: %w", err)
	}

	if err := s.saveConfig(ctx, resp); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	log.Printf("Config sync completed for platform: %s", platform)
	return nil
}

func (s *ConfigService) saveConfig(ctx context.Context, config *models.IddaaConfigResponse) error {
	// Marshal the entire config data as JSON
	configDataBytes, err := json.Marshal(config.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal config data: %w", err)
	}

	params := database.UpsertConfigParams{
		Platform:   config.Data.Platform,
		ConfigData: configDataBytes,
	}

	// Handle optional sportoto program name
	if config.Data.SportotoProgramName != "" {
		params.SportotoProgramName = pgtype.Text{String: config.Data.SportotoProgramName, Valid: true}
	}

	// Handle optional payin end date
	if config.Data.PayinEndDate != "" {
		payinEndDate, err := time.Parse("2006-01-02T15:04:05", config.Data.PayinEndDate)
		if err != nil {
			log.Printf("Failed to parse payin end date %s: %v", config.Data.PayinEndDate, err)
		} else {
			params.PayinEndDate = pgtype.Timestamp{Time: payinEndDate, Valid: true}
		}
	}

	// Handle next draw expected win
	if config.Data.NextDrawExpectedWin != 0 {
		params.NextDrawExpectedWin = pgtype.Numeric{Valid: true}
		// Convert float64 to string for Numeric type
		if err := params.NextDrawExpectedWin.Scan(fmt.Sprintf("%.2f", config.Data.NextDrawExpectedWin)); err != nil {
			log.Printf("Failed to scan next draw expected win: %v", err)
			params.NextDrawExpectedWin = pgtype.Numeric{Valid: false}
		}
	}

	_, err = s.db.UpsertConfig(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to upsert config: %w", err)
	}

	return nil
}

func (s *ConfigService) GetLatestConfig(ctx context.Context, platform string) (*database.AppConfig, error) {
	config, err := s.db.GetLatestConfig(ctx, platform)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("config for platform %s not found", platform)
		}
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return &config, nil
}
