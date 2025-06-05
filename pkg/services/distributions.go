package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/betslib/iddaa-core/pkg/database"
	"github.com/jackc/pgx/v5/pgtype"
)

type DistributionService struct {
	db     *database.Queries
	client *IddaaClient
}

func NewDistributionService(db *database.Queries, client *IddaaClient) *DistributionService {
	return &DistributionService{
		db:     db,
		client: client,
	}
}

type OutcomeDistributionResponse struct {
	IsSuccess bool                                     `json:"isSuccess"`
	Data      map[string]map[string]map[string]float64 `json:"data"` // event_id -> market_id -> outcome -> percentage
	Message   string                                   `json:"message"`
}

// FetchAndUpdateDistributions fetches outcome betting distribution data
func (s *DistributionService) FetchAndUpdateDistributions(ctx context.Context, sportType int) error {
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

	// Process each event
	for eventIDStr, markets := range response.Data {
		eventID, err := strconv.Atoi(eventIDStr)
		if err != nil {
			fmt.Printf("Invalid event ID %s: %v\n", eventIDStr, err)
			continue
		}

		// Check if event exists in our database
		event, err := s.db.GetEventByExternalIDSimple(ctx, eventIDStr)
		if err != nil {
			// Event not found, skip
			continue
		}

		// Process each market for this event
		for marketIDStr, outcomes := range markets {
			marketID, err := strconv.Atoi(marketIDStr)
			if err != nil {
				fmt.Printf("Invalid market ID %s: %v\n", marketIDStr, err)
				continue
			}

			// Process each outcome
			for outcome, percentage := range outcomes {
				err := s.updateOutcomeDistribution(ctx, event.ID, marketID, outcome, percentage)
				if err != nil {
					fmt.Printf("Failed to update distribution for event %d, market %d, outcome %s: %v\n",
						eventID, marketID, outcome, err)
				}
			}
		}
	}

	return nil
}

func (s *DistributionService) updateOutcomeDistribution(ctx context.Context, eventID int32, marketID int, outcome string, percentage float64) error {
	// Get current distribution if exists
	current, err := s.db.GetOutcomeDistribution(ctx, database.GetOutcomeDistributionParams{
		EventID:  pgtype.Int4{Int32: eventID, Valid: true},
		MarketID: int32(marketID),
		Outcome:  outcome,
	})

	var previousPercentage pgtype.Numeric
	if err == nil {
		// Record exists, track the change
		currentFloat, _ := current.BetPercentage.Float64Value()
		previousPercentage = pgtype.Numeric{}
		prevStr := fmt.Sprintf("%.2f", currentFloat.Float64)
		if err := previousPercentage.Scan(prevStr); err != nil {
			return fmt.Errorf("failed to convert previous percentage %.2f: %w", currentFloat.Float64, err)
		}

		// Record history if percentage changed
		if currentFloat.Float64 != percentage {
			newPercentageHist := pgtype.Numeric{}
			newStr := fmt.Sprintf("%.2f", percentage)
			if err := newPercentageHist.Scan(newStr); err != nil {
				return fmt.Errorf("failed to convert new percentage %.2f: %w", percentage, err)
			}

			_, err = s.db.CreateDistributionHistory(ctx, database.CreateDistributionHistoryParams{
				EventID:            pgtype.Int4{Int32: eventID, Valid: true},
				MarketID:           int32(marketID),
				Outcome:            outcome,
				BetPercentage:      newPercentageHist,
				PreviousPercentage: previousPercentage,
			})
			if err != nil {
				return fmt.Errorf("failed to create distribution history: %w", err)
			}
		}
	}

	// Calculate implied probability from current odds
	impliedProb, err := s.getImpliedProbability(ctx, eventID, outcome)
	if err != nil {
		impliedProb = pgtype.Numeric{Valid: false}
	}

	// Upsert the distribution
	newPercentage := pgtype.Numeric{}
	percentageStr := fmt.Sprintf("%.2f", percentage)
	if err := newPercentage.Scan(percentageStr); err != nil {
		return fmt.Errorf("failed to convert percentage %.2f: %w", percentage, err)
	}

	_, err = s.db.UpsertOutcomeDistribution(ctx, database.UpsertOutcomeDistributionParams{
		EventID:            pgtype.Int4{Int32: eventID, Valid: true},
		MarketID:           int32(marketID),
		Outcome:            outcome,
		BetPercentage:      newPercentage,
		ImpliedProbability: impliedProb,
	})

	return err
}

func (s *DistributionService) getImpliedProbability(ctx context.Context, eventID int32, outcome string) (pgtype.Numeric, error) {
	// Get current odds for this outcome
	odds, err := s.db.GetCurrentOddsForOutcome(ctx, database.GetCurrentOddsForOutcomeParams{
		EventID: pgtype.Int4{Int32: eventID, Valid: true},
		Outcome: outcome,
	})
	if err != nil {
		return pgtype.Numeric{Valid: false}, err
	}

	if len(odds) == 0 {
		return pgtype.Numeric{Valid: false}, nil
	}

	// Convert odds to implied probability
	// Implied probability = 1 / decimal_odds * 100
	oddsFloat, err := odds[0].OddsValue.Float64Value()
	if err != nil || !oddsFloat.Valid || oddsFloat.Float64 <= 0 {
		return pgtype.Numeric{Valid: false}, nil
	}

	impliedProb := (1.0 / oddsFloat.Float64) * 100
	result := pgtype.Numeric{}
	impliedStr := fmt.Sprintf("%.2f", impliedProb)
	if err := result.Scan(impliedStr); err != nil {
		return pgtype.Numeric{Valid: false}, fmt.Errorf("failed to convert implied probability %.2f: %w", impliedProb, err)
	}
	return result, nil
}

// GetValueBets finds outcomes where public betting doesn't match implied probabilities
func (s *DistributionService) GetValueBets(ctx context.Context, minBias float64) ([]ValueBet, error) {
	// For now, return empty slice since we simplified the query
	// This can be implemented later with a more complex approach
	return []ValueBet{}, nil
}

// GetContrarianOpportunities finds heavily backed favorites to fade
func (s *DistributionService) GetContrarianOpportunities(ctx context.Context) ([]ContrarianBet, error) {
	// Refresh materialized view first
	err := s.db.RefreshContrarianBets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh contrarian bets: %w", err)
	}

	rows, err := s.db.GetContrarianBets(ctx)
	if err != nil {
		return nil, err
	}

	bets := make([]ContrarianBet, len(rows))
	for i, row := range rows {
		oddsFloat, _ := row.Odds.Float64Value()
		bets[i] = ContrarianBet{
			EventSlug:     row.Slug,
			MatchName:     row.MatchName.(string),
			Market:        row.Market.(string),
			PublicChoice:  row.PublicChoice,
			PublicBacking: row.PublicBacking.(string),
			Odds:          oddsFloat.Float64,
			OverbetBy:     row.OverbetBy.(string),
			Strategy:      row.Strategy,
		}
	}

	return bets, nil
}

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
