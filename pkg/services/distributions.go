package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
)

type DistributionService struct {
	db     *generated.Queries
	client *IddaaClient
	logger *logger.Logger
}

func NewDistributionService(db *generated.Queries, client *IddaaClient) *DistributionService {
	return &DistributionService{
		db:     db,
		client: client,
		logger: logger.New("distribution-service"),
	}
}

// FlatDistribution represents a single distribution entry for bulk processing
type FlatDistribution struct {
	EventExternalID    string
	MarketID           int32
	Outcome            string
	BetPercentage      float32
	ImpliedProbability float64
	// For history tracking
	PreviousPercentage *float32
	HasChanged         bool
}

type OutcomeDistributionResponse struct {
	IsSuccess bool                                     `json:"isSuccess"`
	Data      map[string]map[string]map[string]float64 `json:"data"` // event_id -> market_id -> outcome -> percentage
	Message   string                                   `json:"message"`
}

// FetchAndUpdateDistributions fetches outcome betting distribution data using bulk operations
func (s *DistributionService) FetchAndUpdateDistributions(ctx context.Context, sportType int) error {
	start := time.Now()

	url := fmt.Sprintf("https://sportsbookv2.iddaa.com/sportsbook/outcome-play-percentages?sportType=%d", sportType)

	data, err := s.client.FetchData(url)
	if err != nil {
		return fmt.Errorf("failed to fetch distribution data: %w", err)
	}

	var response OutcomeDistributionResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return fmt.Errorf("failed to unmarshal distribution response: %w", err)
	}

	if !response.IsSuccess {
		return fmt.Errorf("API request failed: %s", response.Message)
	}

	if len(response.Data) == 0 {
		s.logger.Info().
			Int("sport_type", sportType).
			Msg("No distribution data returned from API")
		return nil
	}

	// Step 1: Flatten the nested structure into a slice for easier processing
	flatDistributions := s.flattenDistributions(response.Data)
	if len(flatDistributions) == 0 {
		return nil
	}

	// Step 2: Execute bulk operations
	err = s.executeBulkDistributionUpdate(ctx, flatDistributions)
	if err != nil {
		return fmt.Errorf("bulk distribution update failed: %w", err)
	}

	duration := time.Since(start)
	s.logger.Info().
		Int("sport_type", sportType).
		Int("events_count", len(response.Data)).
		Int("distributions_count", len(flatDistributions)).
		Dur("duration", duration).
		Float64("distributions_per_second", float64(len(flatDistributions))/duration.Seconds()).
		Msg("Distribution sync completed")

	return nil
}

// flattenDistributions converts nested map structure to flat slice
func (s *DistributionService) flattenDistributions(data map[string]map[string]map[string]float64) []FlatDistribution {
	// Pre-calculate capacity to avoid reallocations
	capacity := 0
	for _, markets := range data {
		for _, outcomes := range markets {
			capacity += len(outcomes)
		}
	}

	distributions := make([]FlatDistribution, 0, capacity)

	// Flatten the structure - maintaining order is critical here
	for eventIDStr, markets := range data {
		for marketIDStr, outcomes := range markets {
			marketID, err := strconv.Atoi(marketIDStr)
			if err != nil {
				s.logger.Warn().
					Str("market_id", marketIDStr).
					Err(err).
					Msg("Invalid market ID, skipping")
				continue
			}

			for outcome, percentage := range outcomes {
				distributions = append(distributions, FlatDistribution{
					EventExternalID:    eventIDStr,
					MarketID:           int32(marketID),
					Outcome:            outcome,
					BetPercentage:      float32(percentage),
					ImpliedProbability: 0, // Will be calculated later
				})
			}
		}
	}

	return distributions
}

// executeBulkDistributionUpdate performs all database operations efficiently
func (s *DistributionService) executeBulkDistributionUpdate(ctx context.Context, distributions []FlatDistribution) error {
	// Step 1: Extract unique event IDs and bulk fetch existing events
	eventIDMap := make(map[string]bool)
	for _, dist := range distributions {
		eventIDMap[dist.EventExternalID] = true
	}

	eventIDs := make([]string, 0, len(eventIDMap))
	for id := range eventIDMap {
		eventIDs = append(eventIDs, id)
	}

	// Bulk fetch events
	events, err := s.db.GetEventsByExternalIDs(ctx, eventIDs)
	if err != nil {
		return fmt.Errorf("failed to fetch events: %w", err)
	}

	// Create event lookup map
	eventMap := make(map[string]int32)
	for _, event := range events {
		eventMap[event.ExternalID] = event.ID
	}

	// Step 2: Filter distributions to only those with valid events
	validDistributions := make([]FlatDistribution, 0, len(distributions))
	for _, dist := range distributions {
		if _, exists := eventMap[dist.EventExternalID]; exists {
			validDistributions = append(validDistributions, dist)
		}
	}

	if len(validDistributions) == 0 {
		s.logger.Warn().
			Int("total_distributions", len(distributions)).
			Int("valid_events", len(eventMap)).
			Msg("No valid distributions after filtering")
		return nil
	}

	// Step 3: Bulk fetch current distributions for comparison
	currentDistributions, err := s.db.GetAllDistributionsForEvents(ctx, eventIDs)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to fetch current distributions")
	}

	// Build lookup map for current distributions
	type distKey struct {
		eventExtID string
		marketID   int32
		outcome    string
	}
	currentMap := make(map[distKey]float32)
	for _, curr := range currentDistributions {
		key := distKey{
			eventExtID: curr.EventExternalID,
			marketID:   curr.MarketID,
			outcome:    curr.Outcome,
		}
		currentMap[key] = curr.BetPercentage
	}

	// Step 4: Calculate implied probabilities from odds (bulk fetch)
	oddsMap, err := s.getImpliedProbabilitiesInBulk(ctx, eventIDs)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to fetch odds for implied probabilities")
	}

	// Step 5: Identify changes and prepare bulk operations
	// CRITICAL: Maintain order by using slices, not maps
	var (
		// For bulk upsert
		upsertExtIDs      []string
		upsertMarketIDs   []int64 // Changed to int64 to match generated code
		upsertOutcomes    []string
		upsertPercentages []float64 // Changed to float64 to match generated code
		upsertImpliedProb []float64

		// For history (only changed values)
		historyExtIDs      []string
		historyMarketIDs   []int64 // Changed to int64 to match generated code
		historyOutcomes    []string
		historyPercentages []float64 // Changed to float64 to match generated code
		historyPrevPercent []float64 // Changed to float64 to match generated code
	)

	const significantChangeThreshold = 0.01 // 0.01% change

	for _, dist := range validDistributions {
		// Calculate implied probability
		impliedProb := 0.0
		if oddsValue, exists := oddsMap[oddsKey{
			eventExtID: dist.EventExternalID,
			outcome:    dist.Outcome,
		}]; exists && oddsValue > 0 {
			impliedProb = (1.0 / oddsValue) * 100
		}

		// Always add to upsert arrays
		upsertExtIDs = append(upsertExtIDs, dist.EventExternalID)
		upsertMarketIDs = append(upsertMarketIDs, int64(dist.MarketID))
		upsertOutcomes = append(upsertOutcomes, dist.Outcome)
		upsertPercentages = append(upsertPercentages, float64(dist.BetPercentage))
		upsertImpliedProb = append(upsertImpliedProb, impliedProb)

		// Check if this is a change worth recording
		key := distKey{
			eventExtID: dist.EventExternalID,
			marketID:   dist.MarketID,
			outcome:    dist.Outcome,
		}
		if prevPercentage, exists := currentMap[key]; exists {
			if math.Abs(float64(prevPercentage-dist.BetPercentage)) > significantChangeThreshold {
				// Add to history arrays
				historyExtIDs = append(historyExtIDs, dist.EventExternalID)
				historyMarketIDs = append(historyMarketIDs, int64(dist.MarketID))
				historyOutcomes = append(historyOutcomes, dist.Outcome)
				historyPercentages = append(historyPercentages, float64(dist.BetPercentage))
				historyPrevPercent = append(historyPrevPercent, float64(prevPercentage))
			}
		}
	}

	// Step 6: Execute bulk operations
	// Bulk upsert distributions
	if len(upsertExtIDs) > 0 {
		_, err = s.db.BulkUpsertDistributions(ctx, generated.BulkUpsertDistributionsParams{
			ExternalIds:          upsertExtIDs,
			MarketIds:            upsertMarketIDs,
			Outcomes:             upsertOutcomes,
			BetPercentages:       upsertPercentages,
			ImpliedProbabilities: upsertImpliedProb,
		})
		if err != nil {
			return fmt.Errorf("failed to bulk upsert distributions: %w", err)
		}
	}

	// Bulk insert history (only if there are changes)
	if len(historyExtIDs) > 0 {
		_, err = s.db.BulkInsertDistributionHistory(ctx, generated.BulkInsertDistributionHistoryParams{
			ExternalIds:         historyExtIDs,
			MarketIds:           historyMarketIDs,
			Outcomes:            historyOutcomes,
			BetPercentages:      historyPercentages,
			PreviousPercentages: historyPrevPercent,
		})
		if err != nil {
			return fmt.Errorf("failed to bulk insert distribution history: %w", err)
		}
	}

	s.logger.Debug().
		Int("total_distributions", len(distributions)).
		Int("valid_distributions", len(validDistributions)).
		Int("upserted", len(upsertExtIDs)).
		Int("history_records", len(historyExtIDs)).
		Msg("Bulk distribution update completed")

	return nil
}

// oddsKey for looking up odds values
type oddsKey struct {
	eventExtID string
	outcome    string
}

// getImpliedProbabilitiesInBulk fetches odds for all events and returns a lookup map
func (s *DistributionService) getImpliedProbabilitiesInBulk(ctx context.Context, eventExtIDs []string) (map[oddsKey]float64, error) {
	odds, err := s.db.GetCurrentOddsForEvents(ctx, eventExtIDs)
	if err != nil {
		return nil, err
	}

	oddsMap := make(map[oddsKey]float64)
	for _, odd := range odds {
		key := oddsKey{
			eventExtID: odd.ExternalID,
			outcome:    odd.Outcome,
		}
		oddsMap[key] = odd.OddsValue
	}

	return oddsMap, nil
}

// Legacy methods kept for compatibility but should not be used
// Use FetchAndUpdateDistributions instead

type ValueBet struct {
	EventSlug          string    `json:"event_slug"`
	MatchName          string    `json:"match_name"`
	EventDate          time.Time `json:"event_date"`
	MarketName         string    `json:"market_name"`
	Outcome            string    `json:"outcome"`
	CurrentOdds        float64   `json:"current_odds"`
	ImpliedProbability string    `json:"implied_probability"`
	PublicBetPercent   string    `json:"public_bet_percentage"`
	BiasPercentage     float64   `json:"bias_percentage"`
	BetAssessment      string    `json:"assessment"`
	Recommendation     string    `json:"recommendation"`
}

type ContrarianBet struct {
	EventSlug     string  `json:"event_slug"`
	MatchName     string  `json:"match_name"`
	Market        string  `json:"market"`
	PublicChoice  string  `json:"public_choice"`
	PublicBacking string  `json:"public_backing"`
	Odds          float64 `json:"odds"`
	OverbetBy     string  `json:"overbet_by"`
	Strategy      string  `json:"strategy"`
}
