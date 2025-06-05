package services

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/betslib/iddaa-core/pkg/database"
	"github.com/betslib/iddaa-core/pkg/logger"
	"github.com/betslib/iddaa-core/pkg/models"
	"github.com/jackc/pgx/v5/pgtype"
)

type EventsService struct {
	db          *database.Queries
	client      *IddaaClient
	rateLimiter *time.Ticker
	mutex       sync.Mutex
	logger      *logger.Logger
}

func NewEventsService(db *database.Queries, client *IddaaClient) *EventsService {
	return &EventsService{
		db:          db,
		client:      client,
		rateLimiter: time.NewTicker(100 * time.Millisecond), // Max 10 requests per second
		logger:      logger.New("events-service"),
	}
}

// ProcessEventsResponse processes the API response and saves events, teams, and odds
func (s *EventsService) ProcessEventsResponse(ctx context.Context, response *models.IddaaEventsResponse) error {
	if !response.IsSuccess || response.Data == nil {
		return fmt.Errorf("invalid API response: isSuccess=%t", response.IsSuccess)
	}

	for _, event := range response.Data.Events {
		// Process home and away teams
		homeTeamID, err := s.upsertTeam(ctx, event.HomeTeam, event.HomeTeam)
		if err != nil {
			return fmt.Errorf("failed to upsert home team %s: %w", event.HomeTeam, err)
		}

		awayTeamID, err := s.upsertTeam(ctx, event.AwayTeam, event.AwayTeam)
		if err != nil {
			return fmt.Errorf("failed to upsert away team %s: %w", event.AwayTeam, err)
		}

		// Get competition ID
		competitionID, err := s.getCompetitionID(ctx, event.CompetitionID)
		if err != nil {
			return fmt.Errorf("failed to get competition %d: %w", event.CompetitionID, err)
		}

		// Convert Unix timestamp to time (iddaa returns seconds, not milliseconds)
		eventDate := time.Unix(event.Date, 0)

		// Convert status to string
		statusStr := s.convertEventStatus(event.Status)

		// Upsert event with all Iddaa fields
		eventRecord, err := s.db.UpsertEvent(ctx, database.UpsertEventParams{
			ExternalID: strconv.Itoa(event.ID),
			LeagueID:   pgtype.Int4{Int32: int32(competitionID), Valid: competitionID > 0},
			HomeTeamID: pgtype.Int4{Int32: int32(homeTeamID), Valid: true},
			AwayTeamID: pgtype.Int4{Int32: int32(awayTeamID), Valid: true},
			EventDate:  pgtype.Timestamp{Time: eventDate, Valid: true},
			Status:     statusStr,
			HomeScore:  pgtype.Int4{Valid: false},
			AwayScore:  pgtype.Int4{Valid: false},
			BulletinID: pgtype.Int8{Int64: int64(event.BulletinID), Valid: true},
			Version:    pgtype.Int8{Int64: int64(event.Version), Valid: true},
			SportID:    pgtype.Int4{Int32: int32(event.SportID), Valid: true},
			BetProgram: pgtype.Int4{Int32: int32(event.BetProgram), Valid: true},
			Mbc:        pgtype.Int4{Int32: int32(event.MBC), Valid: true},
			HasKingOdd: pgtype.Bool{Bool: event.HasKingOdd, Valid: true},
			OddsCount:  pgtype.Int4{Int32: int32(event.OddsCount), Valid: true},
			HasCombine: pgtype.Bool{Bool: event.HasCombine, Valid: true},
			IsLive:     pgtype.Bool{Bool: event.IsLive, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to upsert event %d: %w", event.ID, err)
		}

		// Skip bulk markets processing - use detailed endpoint only for better accuracy
		// Process detailed odds for this specific event (more comprehensive than bulk)
		err = s.fetchAndProcessDetailedOdds(ctx, int(eventRecord.ID), event.ID)
		if err != nil {
			// Fallback to bulk markets if detailed fetch fails
			s.logger.Warn().
				Err(err).
				Int("event_id", event.ID).
				Str("action", "fallback_to_bulk").
				Msg("Failed to fetch detailed odds, using bulk markets")
			err = s.processMarkets(ctx, int(eventRecord.ID), event.Markets, time.Now())
			if err != nil {
				return fmt.Errorf("failed to process fallback markets for event %d: %w", event.ID, err)
			}
		}
	}

	return nil
}

// fetchAndProcessDetailedOdds fetches detailed odds for a specific event and processes them
func (s *EventsService) fetchAndProcessDetailedOdds(ctx context.Context, eventID int, externalEventID int) error {
	// Rate limit API calls to avoid overwhelming the server
	s.mutex.Lock()
	<-s.rateLimiter.C
	s.mutex.Unlock()

	// Fetch detailed event data
	singleEvent, err := s.client.GetSingleEvent(externalEventID)
	if err != nil {
		return fmt.Errorf("failed to fetch single event %d: %w", externalEventID, err)
	}

	// Convert detailed markets to regular markets for processing
	detailedMarkets := make([]models.IddaaMarket, len(singleEvent.Data.Markets))
	for i, market := range singleEvent.Data.Markets {
		// Convert detailed outcomes to regular outcomes
		outcomes := make([]models.IddaaOutcome, len(market.Outcomes))
		for j, outcome := range market.Outcomes {
			outcomes[j] = models.IddaaOutcome(outcome)
		}

		detailedMarkets[i] = models.IddaaMarket{
			ID:           market.ID,
			Type:         market.Type,
			SubType:      market.SubType,
			Version:      market.Version,
			Status:       market.Status,
			MBC:          market.MBC,
			SpecialValue: market.SpecialValue,
			Outcomes:     outcomes,
		}
	}

	// Process the detailed markets (this will override/update the basic odds with more comprehensive data)
	return s.processMarkets(ctx, eventID, detailedMarkets, time.Now())
}

// upsertTeam creates or updates a team record
func (s *EventsService) upsertTeam(ctx context.Context, teamName, externalID string) (int, error) {
	team, err := s.db.UpsertTeam(ctx, database.UpsertTeamParams{
		ExternalID: externalID,
		Name:       teamName,
		Country:    pgtype.Text{Valid: false},
		LogoUrl:    pgtype.Text{Valid: false},
	})
	if err != nil {
		return 0, err
	}
	return int(team.ID), nil
}

// getCompetitionID retrieves the competition ID, returns 0 if not found
func (s *EventsService) getCompetitionID(ctx context.Context, iddaaCompetitionID int) (int, error) {
	// Try to find league by external ID
	league, err := s.db.GetLeagueByExternalID(ctx, fmt.Sprintf("%d", iddaaCompetitionID))
	if err != nil {
		// Return 0 if league not found - this will make CompetitionID null in events table
		return 0, nil
	}
	return int(league.ID), nil
}

// processMarkets processes all markets and their odds for an event
func (s *EventsService) processMarkets(ctx context.Context, eventID int, markets []models.IddaaMarket, _ time.Time) error {
	for _, market := range markets {
		// Convert market subtype to market type code
		marketTypeCode := s.convertMarketSubType(market.SubType)

		// Upsert market type
		marketType, err := s.db.UpsertMarketType(ctx, database.UpsertMarketTypeParams{
			Code:        marketTypeCode,
			Name:        s.getMarketTypeName(market.SubType),
			Description: pgtype.Text{String: s.getMarketTypeDescription(market.SubType, market.SpecialValue), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to upsert market type %s: %w", marketTypeCode, err)
		}

		// Process outcomes (odds)
		for _, outcome := range market.Outcomes {
			// Use the outcome name with special value if applicable
			outcomeStr := outcome.Name
			if market.SpecialValue != "" {
				// For Over/Under markets, include the special value for clarity
				if market.SubType == 60 || market.SubType == 101 || market.SubType == 603 || market.SubType == 604 {
					outcomeStr = fmt.Sprintf("%s %s", outcome.Name, market.SpecialValue)
				} else {
					outcomeStr = fmt.Sprintf("%s (%s)", outcome.Name, market.SpecialValue)
				}
			}

			// Check if we need to store this odds value
			shouldStore, err := s.shouldStoreOdds(ctx, eventID, marketType.ID, outcomeStr, outcome.Odds)
			if err != nil {
				return fmt.Errorf("failed to check odds history: %w", err)
			}

			if !shouldStore {
				continue // Skip if odds haven't changed
			}

			// Convert odds to pgtype.Numeric
			var oddsNumeric pgtype.Numeric
			oddsStr := fmt.Sprintf("%.3f", outcome.Odds)
			if err := oddsNumeric.ScanScientific(oddsStr); err != nil {
				return fmt.Errorf("failed to convert odds value %.3f: %w", outcome.Odds, err)
			}

			// Convert winning odds to pgtype.Numeric
			var winningOddsNumeric pgtype.Numeric
			if outcome.WinningOdds > 0 {
				winningOddsStr := fmt.Sprintf("%.3f", outcome.WinningOdds)
				if err := winningOddsNumeric.ScanScientific(winningOddsStr); err != nil {
					return fmt.Errorf("failed to convert winning odds value %.3f: %w", outcome.WinningOdds, err)
				}
				winningOddsNumeric.Valid = true
			} else {
				winningOddsNumeric.Valid = false
			}

			// Get current odds to preserve opening value and detect changes
			currentOdds, err := s.db.GetCurrentOddsByMarket(ctx, database.GetCurrentOddsByMarketParams{
				EventID:      pgtype.Int4{Int32: int32(eventID), Valid: true},
				MarketTypeID: pgtype.Int4{Int32: marketType.ID, Valid: true},
			})
			if err != nil && err.Error() != "no rows in result set" {
				return fmt.Errorf("failed to get current odds: %w", err)
			}

			var openingValue pgtype.Numeric
			var previousValue pgtype.Numeric
			hasExistingOdds := false

			// Check if we have existing odds for this specific outcome
			for _, existing := range currentOdds {
				if existing.Outcome == outcomeStr {
					hasExistingOdds = true
					openingValue = existing.OpeningValue // Preserve original opening value
					previousValue = existing.OddsValue   // Store current as previous
					break
				}
			}

			if !hasExistingOdds {
				// First time seeing this odds, set opening value
				openingValue = oddsNumeric
				previousValue.Valid = false
			}

			_, err = s.db.UpsertCurrentOdds(ctx, database.UpsertCurrentOddsParams{
				EventID:      pgtype.Int4{Int32: int32(eventID), Valid: true},
				MarketTypeID: pgtype.Int4{Int32: marketType.ID, Valid: true},
				Outcome:      outcomeStr,
				OddsValue:    oddsNumeric,
				OpeningValue: openingValue, // Preserve original opening value
				HighestValue: oddsNumeric,
				LowestValue:  oddsNumeric,
				WinningOdds:  winningOddsNumeric,
			})
			if err != nil {
				return fmt.Errorf("failed to upsert odds for event %d, market %d, outcome %s: %w",
					eventID, marketType.ID, outcomeStr, err)
			}

			// If odds changed, create history record
			if hasExistingOdds && previousValue.Valid {
				prevFloat, _ := previousValue.Float64Value()
				if prevFloat.Valid && math.Abs(prevFloat.Float64-outcome.Odds) > 0.001 {
					_, err = s.db.CreateOddsHistory(ctx, database.CreateOddsHistoryParams{
						EventID:       pgtype.Int4{Int32: int32(eventID), Valid: true},
						MarketTypeID:  pgtype.Int4{Int32: marketType.ID, Valid: true},
						Outcome:       outcomeStr,
						OddsValue:     oddsNumeric,
						PreviousValue: previousValue,
						WinningOdds:   winningOddsNumeric,
					})
					if err != nil {
						return fmt.Errorf("failed to create odds history for event %d, market %d, outcome %s: %w",
							eventID, marketType.ID, outcomeStr, err)
					}
				}
			}
		}
	}
	return nil
}

// shouldStoreOdds checks if the odds have changed from the last recorded value
func (s *EventsService) shouldStoreOdds(ctx context.Context, eventID int, marketTypeID int32, outcome string, newOdds float64) (bool, error) {
	// Get the latest odds for this event/market/outcome
	latestOdds, err := s.db.GetCurrentOdds(ctx, pgtype.Int4{Int32: int32(eventID), Valid: true})
	if err != nil {
		return true, nil // If error or no previous odds, store the new value
	}

	// Check if this specific market/outcome exists in the latest odds
	for _, odds := range latestOdds {
		if odds.MarketTypeID.Int32 == marketTypeID && odds.Outcome == outcome {
			// Convert stored odds to float for comparison
			storedFloat, err := odds.OddsValue.Float64Value()
			if err != nil || !storedFloat.Valid {
				return true, nil // If can't parse, store new value
			}
			storedValue := storedFloat.Float64

			// Only store if the value has changed
			// Using a small epsilon for float comparison
			epsilon := 0.001
			if math.Abs(storedValue-newOdds) > epsilon {
				return true, nil // Odds have changed
			}
			return false, nil // Odds are the same
		}
	}

	return true, nil // No previous odds found, store the new value
}

// convertEventStatus converts integer status to string
func (s *EventsService) convertEventStatus(status int) string {
	switch status {
	case 0:
		return "scheduled"
	case 1:
		return "live"
	case 2:
		return "finished"
	case 3:
		return "postponed"
	case 4:
		return "cancelled"
	default:
		return "unknown"
	}
}

// convertMarketSubType converts iddaa market subtype to our market type code
func (s *EventsService) convertMarketSubType(subType int) string {
	switch subType {
	case 1:
		return "1X2"
	case 60:
		return "OU_0_5"
	case 101:
		return "OU_2_5"
	case 89:
		return "BTTS"
	case 88:
		return "HT"
	case 92:
		return "DC"
	case 77:
		return "DNB"
	case 91:
		return "OE"
	case 720:
		return "RED_CARD"
	case 36:
		return "EXACT_SCORE"
	case 603:
		return "HOME_OU"
	case 604:
		return "AWAY_OU"
	case 722:
		return "HOME_CORNER"
	case 723:
		return "AWAY_CORNER"
	default:
		return fmt.Sprintf("MARKET_%d", subType)
	}
}

// getMarketTypeName returns human-readable market type name
func (s *EventsService) getMarketTypeName(subType int) string {
	switch subType {
	case 1:
		return "Match Result"
	case 60:
		return "Over/Under 0.5 Goals"
	case 101:
		return "Over/Under 2.5 Goals"
	case 89:
		return "Both Teams to Score"
	case 88:
		return "Half Time Result"
	case 92:
		return "Double Chance"
	case 77:
		return "Draw No Bet"
	case 91:
		return "Total Goals Odd/Even"
	case 720:
		return "Red Card"
	case 36:
		return "Exact Score"
	case 603:
		return "Home Team Over/Under Goals"
	case 604:
		return "Away Team Over/Under Goals"
	case 722:
		return "Home Team Corner Kicks"
	case 723:
		return "Away Team Corner Kicks"
	default:
		return fmt.Sprintf("Market Type %d", subType)
	}
}

// getMarketTypeDescription returns detailed description with special values
func (s *EventsService) getMarketTypeDescription(subType int, specialValue string) string {
	baseName := s.getMarketTypeName(subType)
	if specialValue != "" {
		return fmt.Sprintf("%s (%s)", baseName, specialValue)
	}
	return baseName
}

// GetActiveSports returns all active sports from the database
func (s *EventsService) GetActiveSports(ctx context.Context) ([]database.Sport, error) {
	sports, err := s.db.ListSports(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sports: %w", err)
	}
	return sports, nil
}

// ProcessDetailedMarkets processes detailed markets from the single event endpoint
func (s *EventsService) ProcessDetailedMarkets(ctx context.Context, eventID int, markets []models.IddaaDetailedMarket, timestamp time.Time) error {
	for _, market := range markets {
		// Convert detailed market to standard market format for processing
		standardMarket := models.IddaaMarket{
			ID:           market.ID,
			Type:         market.Type,
			SubType:      market.SubType,
			SpecialValue: market.SpecialValue,
			Outcomes:     make([]models.IddaaOutcome, len(market.Outcomes)),
		}

		// Convert detailed outcomes to standard outcomes
		for i, outcome := range market.Outcomes {
			standardMarket.Outcomes[i] = models.IddaaOutcome{
				Number: outcome.Number,
				Odds:   outcome.Odds,
				Name:   outcome.Name,
			}
		}

		// Process using existing market processing logic
		err := s.processSingleMarket(ctx, eventID, standardMarket, timestamp)
		if err != nil {
			// Log error but continue processing other markets
			fmt.Printf("Failed to process detailed market %d for event %d: %v\n", market.ID, eventID, err)
		}
	}
	return nil
}

// processSingleMarket processes a single market (extracted from processMarkets for reuse)
func (s *EventsService) processSingleMarket(ctx context.Context, eventID int, market models.IddaaMarket, timestamp time.Time) error {
	// Create market type code similar to existing logic
	marketTypeCode := fmt.Sprintf("%d_%d", market.Type, market.SubType)

	// Upsert market type
	marketType, err := s.db.UpsertMarketType(ctx, database.UpsertMarketTypeParams{
		Code:        marketTypeCode,
		Name:        s.getMarketTypeName(market.SubType),
		Description: pgtype.Text{String: s.getMarketTypeDescription(market.SubType, market.SpecialValue), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to upsert market type %s: %w", marketTypeCode, err)
	}

	// Process each outcome
	for _, outcome := range market.Outcomes {
		outcomeName := outcome.Name
		if market.SpecialValue != "" {
			// Format outcome name with special value for better readability
			if market.SubType == 60 || market.SubType == 101 || market.SubType == 603 || market.SubType == 604 {
				// For Over/Under markets, integrate special value naturally
				outcomeName = fmt.Sprintf("%s %s", outcome.Name, market.SpecialValue)
			} else {
				// For other markets, append in parentheses
				outcomeName = fmt.Sprintf("%s (%s)", outcome.Name, market.SpecialValue)
			}
		}

		// Convert odds to pgtype.Numeric
		oddsNumeric := pgtype.Numeric{}
		if err := oddsNumeric.Scan(fmt.Sprintf("%.2f", outcome.Odds)); err != nil {
			return fmt.Errorf("failed to convert odds to numeric: %w", err)
		}

		// Upsert current odds
		_, err := s.db.UpsertCurrentOdds(ctx, database.UpsertCurrentOddsParams{
			EventID:      pgtype.Int4{Int32: int32(eventID), Valid: true},
			MarketTypeID: pgtype.Int4{Int32: marketType.ID, Valid: true},
			Outcome:      outcomeName,
			OddsValue:    oddsNumeric,
			OpeningValue: oddsNumeric, // Use current as opening if this is first time
			HighestValue: oddsNumeric,
			LowestValue:  oddsNumeric,
			WinningOdds:  oddsNumeric,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert current odds: %w", err)
		}

		// Don't create odds history here - it should only be created when odds actually change
		// This function is called for initial odds sync from ProcessDetailedMarkets
		// The processMarkets function handles proper odds history creation with change detection
	}

	return nil
}
