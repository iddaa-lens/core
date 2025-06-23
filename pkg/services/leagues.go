package services

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
)

type LeaguesService struct {
	db           *generated.Queries
	client       *http.Client
	apiKey       string
	iddaaClient  *IddaaClient
	aiTranslator *AITranslationService
	logger       *logger.Logger
}

func NewLeaguesService(db *generated.Queries, client *http.Client, apiKey string, iddaaClient *IddaaClient, openaiKey string) *LeaguesService {
	var aiTranslator *AITranslationService
	if openaiKey != "" {
		aiTranslator = NewAITranslationService(openaiKey)
	}

	return &LeaguesService{
		db:           db,
		client:       client,
		apiKey:       apiKey,
		iddaaClient:  iddaaClient,
		aiTranslator: aiTranslator,
		logger:       logger.New("leagues-service"),
	}
}

// SyncLeaguesFromIddaa fetches and syncs leagues from Iddaa competitions endpoint
func (s *LeaguesService) SyncLeaguesFromIddaa(ctx context.Context) error {
	s.logger.Info().
		Str("action", "sync_start").
		Msg("Starting Iddaa leagues sync")

	// Get list of valid sports from database
	validSports, err := s.db.ListSports(ctx)
	if err != nil {
		return fmt.Errorf("failed to get sports: %w", err)
	}

	// Create map for faster sport ID lookup
	sportIDMap := make(map[int32]bool)
	for _, sport := range validSports {
		sportIDMap[sport.ID] = true
	}

	// Fetch competitions from Iddaa API
	response, err := s.iddaaClient.GetCompetitions()
	if err != nil {
		return fmt.Errorf("failed to fetch competitions: %w", err)
	}

	if len(response.Data) == 0 {
		s.logger.Warn().
			Str("action", "no_data").
			Msg("No competitions data returned from Iddaa")
		return nil
	}

	// Prepare bulk data
	var (
		externalIds []string
		names       []string
		countries   []string
		sportIds    []int32
		isActives   []bool
	)

	successCount := 0
	skippedCount := 0

	for _, comp := range response.Data {
		// Skip invalid competitions
		if comp.ID == 0 || comp.Name == "" {
			skippedCount++
			s.logger.Debug().
				Int("comp_id", comp.ID).
				Str("comp_name", comp.Name).
				Str("action", "skipped_invalid").
				Msg("Skipping invalid competition")
			continue
		}

		// Extract and validate sport ID
		sportIDInt, err := strconv.Atoi(comp.SportID)
		if err != nil {
			skippedCount++
			s.logger.Debug().
				Int("comp_id", comp.ID).
				Str("sport_id", comp.SportID).
				Str("action", "skipped_invalid_sport").
				Msg("Skipping competition with invalid sport ID")
			continue
		}

		// Skip if sport doesn't exist in our database
		if !sportIDMap[int32(sportIDInt)] {
			skippedCount++
			s.logger.Debug().
				Int("comp_id", comp.ID).
				Int("sport_id", sportIDInt).
				Str("action", "skipped_unknown_sport").
				Msg("Skipping competition with unknown sport")
			continue
		}

		// Use country ID as country name
		country := comp.CountryID

		// Create external ID as string from the integer ID
		externalID := strconv.Itoa(comp.ID)

		// Add to bulk arrays
		externalIds = append(externalIds, externalID)
		names = append(names, comp.Name)
		countries = append(countries, country)
		sportIds = append(sportIds, int32(sportIDInt))
		isActives = append(isActives, true) // Assume all competitions are active

		successCount++

		s.logger.Debug().
			Int("comp_id", comp.ID).
			Str("comp_name", comp.Name).
			Str("country", country).
			Int("sport_id", sportIDInt).
			Bool("is_active", true).
			Str("action", "prepared").
			Msg("Prepared competition for bulk insert")
	}

	// Perform bulk upsert if we have data
	if len(externalIds) > 0 {
		rowsAffected, err := s.db.BulkUpsertLeagues(ctx, generated.BulkUpsertLeaguesParams{
			ExternalIds: externalIds,
			Names:       names,
			Countries:   countries,
			SportIds:    sportIds,
			IsActives:   isActives,
		})
		if err != nil {
			return fmt.Errorf("failed to bulk upsert leagues: %w", err)
		}

		s.logger.Info().
			Int64("rows_affected", rowsAffected).
			Int("success_count", successCount).
			Int("skipped_count", skippedCount).
			Int("total_count", len(response.Data)).
			Str("action", "sync_complete").
			Msg("Iddaa leagues sync completed")
	} else {
		s.logger.Warn().
			Int("success_count", successCount).
			Int("skipped_count", skippedCount).
			Int("total_count", len(response.Data)).
			Str("action", "sync_complete").
			Msg("No valid leagues to sync")
	}

	return nil
}
