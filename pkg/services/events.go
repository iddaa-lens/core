package services

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models"
)

type EventsService struct {
	db          *database.Queries
	client      *IddaaClient
	rateLimiter *time.Ticker
	mutex       sync.Mutex
	logger      *logger.Logger
	// Market type cache to reduce database calls
	marketTypeCache map[string]database.MarketType
	marketTypeMutex sync.RWMutex
}

func NewEventsService(db *database.Queries, client *IddaaClient) *EventsService {
	service := &EventsService{
		db:              db,
		client:          client,
		rateLimiter:     time.NewTicker(100 * time.Millisecond), // Max 10 requests per second
		logger:          logger.New("events-service"),
		marketTypeCache: make(map[string]database.MarketType),
	}

	// Pre-populate market type cache on startup to reduce database calls
	go service.preloadMarketTypeCache()

	return service
}

// ProcessEventsResponse processes the API response and saves events, teams, and odds
func (s *EventsService) ProcessEventsResponse(ctx context.Context, response *models.IddaaEventsResponse) error {
	if !response.IsSuccess || response.Data == nil {
		return fmt.Errorf("invalid API response: isSuccess=%t", response.IsSuccess)
	}

	log := logger.WithContext(ctx, "process-events")
	var processErrors []error
	successCount := 0

	for _, event := range response.Data.Events {
		// Create a timeout context for each event to prevent one event from blocking others
		eventCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

		// Process home and away teams
		homeTeamID, err := s.upsertTeam(eventCtx, event.HomeTeam, event.HomeTeam)
		if err != nil {
			cancel()
			log.Error().
				Err(err).
				Str("team_name", event.HomeTeam).
				Int("event_id", event.ID).
				Msg("Failed to upsert home team")
			processErrors = append(processErrors, fmt.Errorf("home team %s: %w", event.HomeTeam, err))
			continue
		}

		awayTeamID, err := s.upsertTeam(eventCtx, event.AwayTeam, event.AwayTeam)
		if err != nil {
			cancel()
			log.Error().
				Err(err).
				Str("team_name", event.AwayTeam).
				Int("event_id", event.ID).
				Msg("Failed to upsert away team")
			processErrors = append(processErrors, fmt.Errorf("away team %s: %w", event.AwayTeam, err))
			continue
		}

		// Get competition ID
		competitionID, err := s.getCompetitionID(eventCtx, event.CompetitionID)
		if err != nil {
			cancel()
			log.Error().
				Err(err).
				Int("competition_id", event.CompetitionID).
				Int("event_id", event.ID).
				Msg("Failed to get competition")
			processErrors = append(processErrors, fmt.Errorf("competition %d: %w", event.CompetitionID, err))
			continue
		}

		// Convert Unix timestamp to time (iddaa returns seconds, not milliseconds)
		// IMPORTANT: Iddaa API provides timestamps that are already in UTC
		// No timezone conversion needed - the timestamps are correct as-is

		eventDate := time.Unix(event.Date, 0)

		s.logger.Debug().
			Int64("original_unix", event.Date).
			Str("utc_time", eventDate.UTC().Format("2006-01-02 15:04:05 UTC")).
			Str("will_display_in_la", eventDate.In(time.FixedZone("PDT", -7*3600)).Format("2006-01-02 15:04:05 PDT")).
			Int("event_id", event.ID).
			Msg("Using Iddaa UTC timestamp directly")

		// Convert status to string
		statusStr := s.convertEventStatus(event.Status)

		// Upsert event with all Iddaa fields
		eventRecord, err := s.db.UpsertEvent(eventCtx, database.UpsertEventParams{
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
			cancel()
			log.Error().
				Err(err).
				Int("event_id", event.ID).
				Msg("Failed to upsert event")
			processErrors = append(processErrors, fmt.Errorf("event %d: %w", event.ID, err))
			continue
		}

		// Skip bulk markets processing - use detailed endpoint only for better accuracy
		// Process detailed odds for this specific event (more comprehensive than bulk)
		err = s.fetchAndProcessDetailedOdds(eventCtx, int(eventRecord.ID), event.ID)
		if err != nil {
			// Fallback to bulk markets if detailed fetch fails
			s.logger.Warn().
				Err(err).
				Int("event_id", event.ID).
				Str("action", "fallback_to_bulk").
				Msg("Failed to fetch detailed odds, using bulk markets")
			err = s.processMarkets(eventCtx, int(eventRecord.ID), event.Markets, time.Now())
			if err != nil {
				log.Error().
					Err(err).
					Int("event_id", event.ID).
					Msg("Failed to process fallback markets")
				processErrors = append(processErrors, fmt.Errorf("fallback markets for event %d: %w", event.ID, err))
			}
		}

		cancel()
		successCount++
	}

	// Log summary
	log.Info().
		Int("success_count", successCount).
		Int("error_count", len(processErrors)).
		Int("total_events", len(response.Data.Events)).
		Msg("Completed processing events response")

	// Return error if all events failed
	if successCount == 0 && len(processErrors) > 0 {
		return fmt.Errorf("all events failed to process: %v", processErrors[0])
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
		// Use consistent code format: TYPE_SUBTYPE (same as processSingleMarket)
		marketTypeCode := fmt.Sprintf("%d_%d", market.Type, market.SubType)

		// Get or create market type with caching to reduce database load
		marketType, err := s.getOrCreateMarketType(ctx, marketTypeCode, market)
		if err != nil {
			// Log error but continue processing other markets instead of failing completely
			s.logger.Error().
				Err(err).
				Str("market_type_code", marketTypeCode).
				Int("event_id", eventID).
				Int("market_id", market.ID).
				Msg("Failed to get or create market type, skipping this market")
			continue
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

// getMarketTypeName returns human-readable market type name
func (s *EventsService) getMarketTypeName(subType int) string {
	switch subType {
	case 1:
		return "Maç Sonucu"
	case 60:
		return "Alt/Üst 0.5 Gol"
	case 101:
		return "Alt/Üst 2.5 Gol"
	case 89:
		return "İki Takım da Gol Atar"
	case 88:
		return "İlk Yarı Sonucu"
	case 92:
		return "Çifte Şans"
	case 77:
		return "Beraberlik Yoksa İade"
	case 91:
		return "Toplam Gol Tek/Çift"
	case 720:
		return "Kırmızı Kart"
	case 36:
		return "Tam Skor"
	case 603:
		return "Ev Sahibi Alt/Üst Gol"
	case 604:
		return "Deplasman Alt/Üst Gol"
	case 722:
		return "Ev Sahibi Korner"
	case 723:
		return "Deplasman Korner"
	case 7:
		return "Alt/Üst Gol"
	case 4:
		return "Toplam Gol Sayısı"
	case 85:
		return "Gol Atacak Takım"
	case 86:
		return "Tam Skor"
	case 87:
		return "İlk Gol"
	case 90:
		return "Her İki Yarı"
	case 698:
		return "Penaltı Var/Yok"
	case 699:
		return "Kırmızı Kart Var/Yok"
	case 717:
		return "İlk Yarı Tam Skor"
	case 718:
		return "Yarı/Maç Sonucu"
	case 724:
		return "Korner Sayısı"
	case 100:
		return "Handikap"
	default:
		return fmt.Sprintf("Pazar Tipi %d", subType)
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
			logger.WithContext(ctx, "process-detailed-markets").Error().
				Err(err).
				Int("market_id", market.ID).
				Int("event_id", eventID).
				Int("market_type", market.Type).
				Int("market_subtype", market.SubType).
				Msg("Failed to process detailed market")
		}
	}
	return nil
}

// processSingleMarket processes a single market (extracted from processMarkets for reuse)
func (s *EventsService) processSingleMarket(ctx context.Context, eventID int, market models.IddaaMarket, timestamp time.Time) error {
	// Create market type code similar to existing logic
	marketTypeCode := fmt.Sprintf("%d_%d", market.Type, market.SubType)

	// Generate slug from code (same logic as SQL: LOWER(REPLACE(REPLACE(code, '_', '-'), ' ', '-')))
	slug := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(marketTypeCode, "_", "-"), " ", "-"))

	// Upsert market type
	marketType, err := s.db.UpsertMarketType(ctx, database.UpsertMarketTypeParams{
		Code:                  marketTypeCode,
		Name:                  s.getMarketTypeName(market.SubType),
		Slug:                  slug,
		Description:           pgtype.Text{String: s.getMarketTypeDescription(market.SubType, market.SpecialValue), Valid: true},
		IddaaMarketID:         pgtype.Int4{Int32: int32(market.ID), Valid: true},
		IsLive:                pgtype.Bool{Valid: false}, // Not set
		MarketType:            pgtype.Int4{Int32: int32(market.Type), Valid: true},
		MinMarketDefaultValue: pgtype.Int4{Valid: false}, // Not set
		MaxMarketLimitValue:   pgtype.Int4{Valid: false}, // Not set
		Priority:              pgtype.Int4{Valid: false}, // Not set
		SportType:             pgtype.Int4{Valid: false}, // Not set
		MarketSubType:         pgtype.Int4{Int32: int32(market.SubType), Valid: true},
		MinDefaultValue:       pgtype.Int4{Valid: false}, // Not set
		MaxLimitValue:         pgtype.Int4{Valid: false}, // Not set
		IsActive:              pgtype.Bool{Bool: true, Valid: true},
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

// getOrCreateMarketType gets a market type from cache or creates it in database
func (s *EventsService) getOrCreateMarketType(ctx context.Context, marketTypeCode string, market models.IddaaMarket) (database.MarketType, error) {
	// Check cache first
	s.marketTypeMutex.RLock()
	if cachedMarketType, exists := s.marketTypeCache[marketTypeCode]; exists {
		s.marketTypeMutex.RUnlock()
		return cachedMarketType, nil
	}
	s.marketTypeMutex.RUnlock()

	// Not in cache, try to get from database first
	marketType, err := s.db.GetMarketType(ctx, marketTypeCode)
	if err == nil {
		// Found in database, add to cache
		s.marketTypeMutex.Lock()
		s.marketTypeCache[marketTypeCode] = marketType
		s.marketTypeMutex.Unlock()
		return marketType, nil
	}

	// Not found in database, create with shorter timeout and retry logic
	marketCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Generate slug from code (same logic as SQL: LOWER(REPLACE(REPLACE(code, '_', '-'), ' ', '-')))
	slug := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(marketTypeCode, "_", "-"), " ", "-"))

	// Use exponential backoff for retries
	maxRetries := 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		marketType, err = s.db.UpsertMarketType(marketCtx, database.UpsertMarketTypeParams{
			Code:                  marketTypeCode,
			Name:                  s.getMarketTypeName(market.SubType),
			Slug:                  slug,
			Description:           pgtype.Text{String: s.getMarketTypeDescription(market.SubType, market.SpecialValue), Valid: true},
			IddaaMarketID:         pgtype.Int4{Int32: int32(market.ID), Valid: true},
			IsLive:                pgtype.Bool{Valid: false}, // Not set
			MarketType:            pgtype.Int4{Int32: int32(market.Type), Valid: true},
			MinMarketDefaultValue: pgtype.Int4{Valid: false}, // Not set
			MaxMarketLimitValue:   pgtype.Int4{Valid: false}, // Not set
			Priority:              pgtype.Int4{Valid: false}, // Not set
			SportType:             pgtype.Int4{Valid: false}, // Not set
			MarketSubType:         pgtype.Int4{Int32: int32(market.SubType), Valid: true},
			MinDefaultValue:       pgtype.Int4{Valid: false}, // Not set
			MaxLimitValue:         pgtype.Int4{Valid: false}, // Not set
			IsActive:              pgtype.Bool{Bool: true, Valid: true},
		})

		if err == nil {
			// Success! Add to cache and return
			s.marketTypeMutex.Lock()
			s.marketTypeCache[marketTypeCode] = marketType
			s.marketTypeMutex.Unlock()
			return marketType, nil
		}

		// If it's a timeout or temporary error, retry with exponential backoff
		if attempt < maxRetries {
			backoffDuration := time.Duration(50*(attempt+1)) * time.Millisecond
			s.logger.Warn().
				Err(err).
				Str("market_type_code", marketTypeCode).
				Int("attempt", attempt+1).
				Dur("backoff", backoffDuration).
				Msg("Market type upsert failed, retrying")
			time.Sleep(backoffDuration)
		}
	}

	return database.MarketType{}, fmt.Errorf("failed to upsert market type %s after %d attempts: %w", marketTypeCode, maxRetries+1, err)
}

// preloadMarketTypeCache loads existing market types into cache on startup
func (s *EventsService) preloadMarketTypeCache() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	marketTypes, err := s.db.ListMarketTypes(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to preload market type cache")
		return
	}

	s.marketTypeMutex.Lock()
	defer s.marketTypeMutex.Unlock()

	for _, marketType := range marketTypes {
		s.marketTypeCache[marketType.Code] = marketType
	}

	s.logger.Info().
		Int("count", len(marketTypes)).
		Msg("Market type cache preloaded successfully")
}
