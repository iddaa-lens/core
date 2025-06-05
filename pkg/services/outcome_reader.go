package services

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/iddaa-lens/core/pkg/database"
)

type OutcomeReaderService struct {
	db *database.Queries
}

func NewOutcomeReaderService(db *database.Queries) *OutcomeReaderService {
	return &OutcomeReaderService{db: db}
}

// OutcomeMapping represents a readable interpretation of an outcome
type OutcomeMapping struct {
	MarketID    int32   `json:"market_id"`
	Outcome     string  `json:"outcome"`
	Description string  `json:"description"`
	BetPercent  float64 `json:"bet_percentage"`
	ImpliedProb float64 `json:"implied_probability,omitempty"`
}

// EventDistribution represents all betting distributions for an event
type EventDistribution struct {
	EventID    int32               `json:"event_id"`
	ExternalID string              `json:"external_id"`
	MatchName  string              `json:"match_name"`
	EventDate  string              `json:"event_date"`
	Outcomes   []OutcomeMapping    `json:"outcomes"`
	Summary    DistributionSummary `json:"summary"`
}

// DistributionSummary provides high-level insights
type DistributionSummary struct {
	TotalMarkets     int      `json:"total_markets"`
	TotalOutcomes    int      `json:"total_outcomes"`
	MostBackedChoice string   `json:"most_backed_choice"`
	MostBackedPct    float64  `json:"most_backed_percentage"`
	MarketTypes      []string `json:"market_types"`
}

// GetEventDistribution returns readable distribution data for an event
func (s *OutcomeReaderService) GetEventDistribution(ctx context.Context, eventID int32) (*EventDistribution, error) {
	// Get event details
	event, err := s.db.GetEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	// Get team names for match name
	homeTeam, _ := s.db.GetTeam(ctx, event.HomeTeamID.Int32)
	awayTeam, _ := s.db.GetTeam(ctx, event.AwayTeamID.Int32)

	matchName := "Unknown Match"
	if homeTeam.ID > 0 && awayTeam.ID > 0 {
		matchName = fmt.Sprintf("%s vs %s", homeTeam.Name, awayTeam.Name)
	}

	// Get distribution data
	distributions, err := s.db.GetEventDistributions(ctx, pgtype.Int4{Int32: eventID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get distributions: %w", err)
	}

	outcomes := make([]OutcomeMapping, 0, len(distributions))
	marketTypes := make(map[string]bool)
	var mostBackedChoice string
	var mostBackedPct float64

	for _, dist := range distributions {
		betPct, _ := dist.BetPercentage.Float64Value()
		impliedPct := 0.0
		if dist.ImpliedProbability.Valid {
			impliedFloat, _ := dist.ImpliedProbability.Float64Value()
			impliedPct = impliedFloat.Float64
		}

		outcome := OutcomeMapping{
			MarketID:    dist.MarketID,
			Outcome:     dist.Outcome,
			Description: s.interpretOutcome(dist.MarketID, dist.Outcome),
			BetPercent:  betPct.Float64,
			ImpliedProb: impliedPct,
		}

		outcomes = append(outcomes, outcome)

		// Track market types
		marketType := s.getMarketType(dist.MarketID)
		marketTypes[marketType] = true

		// Track most backed choice
		if betPct.Float64 > mostBackedPct {
			mostBackedPct = betPct.Float64
			mostBackedChoice = outcome.Description
		}
	}

	// Convert market types to slice
	marketTypesList := make([]string, 0, len(marketTypes))
	for mt := range marketTypes {
		marketTypesList = append(marketTypesList, mt)
	}

	return &EventDistribution{
		EventID:    eventID,
		ExternalID: event.ExternalID,
		MatchName:  matchName,
		EventDate:  event.EventDate.Time.Format("2006-01-02 15:04"),
		Outcomes:   outcomes,
		Summary: DistributionSummary{
			TotalMarkets:     len(marketTypes),
			TotalOutcomes:    len(outcomes),
			MostBackedChoice: mostBackedChoice,
			MostBackedPct:    mostBackedPct,
			MarketTypes:      marketTypesList,
		},
	}, nil
}

// interpretOutcome converts outcome numbers to readable descriptions
func (s *OutcomeReaderService) interpretOutcome(marketID int32, outcome string) string {
	// Convert outcome to number for pattern matching
	outcomeNum, err := strconv.Atoi(outcome)
	if err != nil {
		return fmt.Sprintf("Outcome %s", outcome)
	}

	// Common outcome patterns based on Turkish betting (Iddaa) conventions
	switch outcomeNum {
	case 1:
		return "Home Win / Player 1 Win / Yes / Over"
	case 2:
		return "Away Win / Player 2 Win / No / Under"
	case 3:
		return "Draw / Tie"
	case 4:
		return "1X (Home Win or Draw)"
	case 5:
		return "X2 (Draw or Away Win)"
	case 6:
		return "12 (Home or Away Win)"
	case 7:
		return "Home Win & Over 2.5"
	case 8:
		return "Home Win & Under 2.5"
	case 9:
		return "Away Win & Over 2.5"
	case 10:
		return "Away Win & Under 2.5"
	case 11:
		return "Draw & Over 2.5"
	case 12:
		return "Draw & Under 2.5"
	case 15:
		return "0-0 Correct Score"
	case 16:
		return "1-0 Correct Score"
	case 17:
		return "0-1 Correct Score"
	case 18:
		return "1-1 Correct Score"
	case 19:
		return "2-0 Correct Score"
	case 20:
		return "0-2 Correct Score"
	case 21:
		return "2-1 Correct Score"
	case 22:
		return "1-2 Correct Score"
	case 23:
		return "2-2 Correct Score"
	case 24:
		return "3-0 Correct Score"
	case 25:
		return "0-3 Correct Score"
	case 26:
		return "3-1 Correct Score"
	case 27:
		return "1-3 Correct Score"
	case 28:
		return "3-2 Correct Score"
	case 29:
		return "2-3 Correct Score"
	case 30:
		return "3-3 Correct Score"
	case 31:
		return "4-0 Correct Score"
	case 32:
		return "0-4 Correct Score"
	case 33:
		return "Other Score"
	default:
		// For handicap, total goals, and other special markets
		if outcomeNum >= 100 && outcomeNum <= 199 {
			return fmt.Sprintf("Handicap Outcome %d", outcomeNum-100)
		} else if outcomeNum >= 200 && outcomeNum <= 299 {
			return fmt.Sprintf("Total Goals Outcome %d", outcomeNum-200)
		} else if outcomeNum >= 300 && outcomeNum <= 399 {
			return fmt.Sprintf("Special Market %d", outcomeNum-300)
		}

		return fmt.Sprintf("Outcome %s", outcome)
	}
}

// getMarketType identifies the type of betting market
func (s *OutcomeReaderService) getMarketType(marketID int32) string {
	// This is a simplified approach - in a real system you'd query the market_types table
	// For now, we'll infer from the market ID patterns
	idStr := fmt.Sprintf("%d", marketID)

	// Common patterns in Iddaa market IDs
	if len(idStr) >= 8 {
		// Most Iddaa markets are 8+ digits
		return "Standard Market"
	}

	return "Unknown Market"
}

// GetTopDistributions returns the most interesting distribution patterns
func (s *OutcomeReaderService) GetTopDistributions(ctx context.Context, limit int) ([]OutcomeMapping, error) {
	// Get top distributions by betting percentage
	distributions, err := s.db.GetTopDistributions(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get top distributions: %w", err)
	}

	outcomes := make([]OutcomeMapping, 0, len(distributions))
	for _, dist := range distributions {
		betPct, _ := dist.BetPercentage.Float64Value()
		impliedPct := 0.0
		if dist.ImpliedProbability.Valid {
			impliedFloat, _ := dist.ImpliedProbability.Float64Value()
			impliedPct = impliedFloat.Float64
		}

		outcome := OutcomeMapping{
			MarketID:    dist.MarketID,
			Outcome:     dist.Outcome,
			Description: s.interpretOutcome(dist.MarketID, dist.Outcome),
			BetPercent:  betPct.Float64,
			ImpliedProb: impliedPct,
		}
		outcomes = append(outcomes, outcome)
	}

	return outcomes, nil
}

// GetDistributionHistory returns historical changes for an outcome
func (s *OutcomeReaderService) GetDistributionHistory(ctx context.Context, eventID int32, marketID int32, outcome string) ([]DistributionChange, error) {
	// Get history from database
	history, err := s.db.GetDistributionHistory(ctx, database.GetDistributionHistoryParams{
		EventID:  pgtype.Int4{Int32: eventID, Valid: true},
		MarketID: marketID,
		Outcome:  outcome,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get distribution history: %w", err)
	}

	changes := make([]DistributionChange, 0, len(history))
	for _, hist := range history {
		currentPct, _ := hist.BetPercentage.Float64Value()
		previousPct := 0.0
		if hist.PreviousPercentage.Valid {
			prevFloat, _ := hist.PreviousPercentage.Float64Value()
			previousPct = prevFloat.Float64
		}

		change := DistributionChange{
			Timestamp:          hist.RecordedAt.Time.Format("2006-01-02 15:04:05"),
			CurrentPercentage:  currentPct.Float64,
			PreviousPercentage: previousPct,
			Change:             currentPct.Float64 - previousPct,
			Description:        s.interpretOutcome(marketID, outcome),
		}
		changes = append(changes, change)
	}

	return changes, nil
}

type DistributionChange struct {
	Timestamp          string  `json:"timestamp"`
	CurrentPercentage  float64 `json:"current_percentage"`
	PreviousPercentage float64 `json:"previous_percentage"`
	Change             float64 `json:"change"`
	Description        string  `json:"description"`
}
