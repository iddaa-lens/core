package services

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/iddaa-lens/core/pkg/database"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models"
)

type EventsService struct {
	db          *database.Queries
	client      *IddaaClient
	logger      *logger.Logger
	rateLimiter *time.Ticker
	// Simple market type cache
	marketTypeCache map[string]database.MarketType
}

func NewEventsService(db *database.Queries, client *IddaaClient) *EventsService {
	service := &EventsService{
		db:              db,
		client:          client,
		logger:          logger.New("events-service"),
		rateLimiter:     time.NewTicker(100 * time.Millisecond), // Max 10 requests per second
		marketTypeCache: make(map[string]database.MarketType),
	}

	// Pre-populate market type cache
	go service.preloadMarketTypeCache()

	return service
}

// Stop gracefully shuts down the service
func (s *EventsService) Stop() {
	s.rateLimiter.Stop()
}

// ProcessEventsResponse processes the API response and saves events, teams, and odds
func (s *EventsService) ProcessEventsResponse(ctx context.Context, response *models.IddaaEventsResponse) error {
	if !response.IsSuccess || response.Data == nil {
		return fmt.Errorf("invalid API response: isSuccess=%t", response.IsSuccess)
	}

	log := logger.WithContext(ctx, "process-events")

	successCount := 0
	var lastError error

	// Process events sequentially to avoid complexity
	for _, event := range response.Data.Events {
		if err := s.processEvent(ctx, event); err != nil {
			lastError = err
			log.Error().
				Err(err).
				Int("event_id", event.ID).
				Str("home_team", event.HomeTeam).
				Str("away_team", event.AwayTeam).
				Msg("Failed to process event")
		} else {
			successCount++
		}
	}

	log.Info().
		Int("success_count", successCount).
		Int("total_events", len(response.Data.Events)).
		Msg("Completed processing events response")

	// Return error if all events failed
	if successCount == 0 && lastError != nil {
		return fmt.Errorf("all events failed to process: %w", lastError)
	}

	return nil
}

// processEvent processes a single event
func (s *EventsService) processEvent(ctx context.Context, event models.IddaaEvent) error {
	// Process teams
	homeTeamID, err := s.upsertTeam(ctx, event.HomeTeam, event.HomeTeam)
	if err != nil {
		return fmt.Errorf("failed to process home team: %w", err)
	}

	awayTeamID, err := s.upsertTeam(ctx, event.AwayTeam, event.AwayTeam)
	if err != nil {
		return fmt.Errorf("failed to process away team: %w", err)
	}

	// Get competition ID
	competitionID, err := s.getCompetitionID(ctx, event.CompetitionID)
	if err != nil {
		return fmt.Errorf("competition error %d: %w", event.CompetitionID, err)
	}

	eventDate := time.Unix(event.Date, 0)
	statusStr := s.convertEventStatus(event.Status)

	// Upsert event
	eventRecord, err := s.db.UpsertEvent(ctx, database.UpsertEventParams{
		ExternalID: strconv.Itoa(event.ID),
		LeagueID:   int32(competitionID),
		HomeTeamID: int32(homeTeamID),
		AwayTeamID: int32(awayTeamID),
		EventDate:  pgtype.Timestamp{Time: eventDate, Valid: true},
		Status:     statusStr,
		HomeScore:  0, // Will be null in DB due to query
		AwayScore:  0, // Will be null in DB due to query
		BulletinID: int64(event.BulletinID),
		Version:    int64(event.Version),
		SportID:    int32(event.SportID),
		BetProgram: int32(event.BetProgram),
		Mbc:        int32(event.MBC),
		HasKingOdd: event.HasKingOdd,
		OddsCount:  int32(event.OddsCount),
		HasCombine: event.HasCombine,
		IsLive:     event.IsLive,
	})
	if err != nil {
		return fmt.Errorf("event %d: %w", event.ID, err)
	}

	// Skip detailed odds for certain sports to reduce API load
	skipDetailedOddsSports := map[int]bool{
		// 117: true, // MMA
		// 5:   true, // Tennis
	}

	if skipDetailedOddsSports[event.SportID] || event.OddsCount > 100 {
		return s.processMarkets(ctx, int(eventRecord.ID), event.Markets)
	}

	// Try to fetch detailed odds with a timeout
	detailCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	err = s.fetchAndProcessDetailedOdds(detailCtx, int(eventRecord.ID), event.ID)
	cancel()

	if err != nil {
		// Fallback to bulk markets
		return s.processMarkets(ctx, int(eventRecord.ID), event.Markets)
	}

	return nil
}

// fetchAndProcessDetailedOdds fetches detailed odds for a specific event
func (s *EventsService) fetchAndProcessDetailedOdds(ctx context.Context, eventID int, externalEventID int) error {
	// Rate limit API calls
	select {
	case <-s.rateLimiter.C:
		// Proceed
	case <-ctx.Done():
		return ctx.Err()
	}

	singleEvent, err := s.client.GetSingleEvent(externalEventID)
	if err != nil {
		return fmt.Errorf("failed to fetch single event: %w", err)
	}

	// Convert detailed markets to regular markets for processing
	detailedMarkets := make([]models.IddaaMarket, len(singleEvent.Data.Markets))
	for i, market := range singleEvent.Data.Markets {
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

	return s.processMarkets(ctx, eventID, detailedMarkets)
}

// processMarkets processes all markets and their odds for an event
func (s *EventsService) processMarkets(ctx context.Context, eventID int, markets []models.IddaaMarket) error {
	for _, market := range markets {
		if err := s.processSingleMarket(ctx, eventID, market); err != nil {
			// Log error but continue processing other markets
			s.logger.Error().
				Err(err).
				Int("event_id", eventID).
				Int("market_id", market.ID).
				Msg("Failed to process market")
		}
	}
	return nil
}

// processSingleMarket processes a single market and its outcomes
func (s *EventsService) processSingleMarket(ctx context.Context, eventID int, market models.IddaaMarket) error {
	marketTypeCode := fmt.Sprintf("%d_%d", market.Type, market.SubType)

	// Get or create market type
	marketType, err := s.getOrCreateMarketType(ctx, marketTypeCode, market)
	if err != nil {
		return fmt.Errorf("market type %s: %w", marketTypeCode, err)
	}

	// Process each outcome
	for _, outcome := range market.Outcomes {
		outcomeStr := s.formatOutcomeName(outcome.Name, market.SubType, market.SpecialValue)

		// Get current odds to check if we need to create history
		currentOdds, err := s.db.GetCurrentOddsByOutcome(ctx, database.GetCurrentOddsByOutcomeParams{
			EventID:      int32(eventID),
			MarketTypeID: marketType.ID,
			Outcome:      outcomeStr,
		})

		// Create numeric values
		oddsValue := pgtype.Numeric{}
		if err := oddsValue.Scan(fmt.Sprintf("%.3f", outcome.Odds)); err != nil {
			s.logger.Error().Err(err).Float64("odds", outcome.Odds).Msg("Failed to scan odds value")
			continue
		}

		var openingValue, highestValue, lowestValue pgtype.Numeric
		var previousValue float64
		hasExistingOdds := err == nil

		if hasExistingOdds {
			openingValue = currentOdds.OpeningValue
			prevFloat, _ := currentOdds.OddsValue.Float64Value()
			if prevFloat.Valid {
				previousValue = prevFloat.Float64
			}
			highestValue = currentOdds.HighestValue
			lowestValue = currentOdds.LowestValue

			// Update highest/lowest if needed
			highestFloat, _ := highestValue.Float64Value()
			lowestFloat, _ := lowestValue.Float64Value()

			if highestFloat.Valid && outcome.Odds > highestFloat.Float64 {
				highestValue = oddsValue
			}
			if lowestFloat.Valid && outcome.Odds < lowestFloat.Float64 {
				lowestValue = oddsValue
			}
		} else {
			openingValue = oddsValue
			highestValue = oddsValue
			lowestValue = oddsValue
		}

		// Upsert current odds
		_, err = s.db.UpsertCurrentOdds(ctx, database.UpsertCurrentOddsParams{
			EventID:      int32(eventID),
			MarketTypeID: marketType.ID,
			Outcome:      outcomeStr,
			OddsValue:    oddsValue,
			OpeningValue: openingValue,
			HighestValue: highestValue,
			LowestValue:  lowestValue,
			WinningOdds:  pgtype.Numeric{}, // Null value
		})
		if err != nil {
			return fmt.Errorf("failed to upsert odds: %w", err)
		}

		// Create history record if odds changed
		if hasExistingOdds && math.Abs(previousValue-outcome.Odds) > 0.001 {
			prevValue := pgtype.Numeric{}
			if err := prevValue.Scan(fmt.Sprintf("%.3f", previousValue)); err != nil {
				s.logger.Error().Err(err).Float64("previous_value", previousValue).Msg("Failed to scan previous value")
				continue
			}

			_, err = s.db.CreateOddsHistory(ctx, database.CreateOddsHistoryParams{
				EventID:       int32(eventID),
				MarketTypeID:  marketType.ID,
				Outcome:       outcomeStr,
				OddsValue:     oddsValue,
				PreviousValue: prevValue,
				WinningOdds:   pgtype.Numeric{}, // Null value
			})
			if err != nil {
				s.logger.Warn().
					Err(err).
					Str("outcome", outcomeStr).
					Msg("Failed to create odds history")
			}
		}
	}

	return nil
}

// formatOutcomeName formats the outcome name with special value if applicable
func (s *EventsService) formatOutcomeName(name string, subType int, specialValue string) string {
	if specialValue == "" {
		return name
	}

	// For Over/Under markets, include the special value for clarity
	if subType == 60 || subType == 101 || subType == 603 || subType == 604 {
		return fmt.Sprintf("%s %s", name, specialValue)
	}

	return fmt.Sprintf("%s (%s)", name, specialValue)
}

// upsertTeam creates or updates a team record
func (s *EventsService) upsertTeam(ctx context.Context, teamName, externalID string) (int, error) {
	team, err := s.db.UpsertTeam(ctx, database.UpsertTeamParams{
		ExternalID: externalID,
		Name:       teamName,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to upsert team: %w", err)
	}
	return int(team.ID), nil
}

// getCompetitionID retrieves the competition ID, returns 0 if not found
func (s *EventsService) getCompetitionID(ctx context.Context, iddaaCompetitionID int) (int, error) {
	league, err := s.db.GetLeagueByExternalID(ctx, fmt.Sprintf("%d", iddaaCompetitionID))
	if err != nil {
		// Return 0 if league not found - this will make LeagueID null in events table
		return 0, nil
	}
	return int(league.ID), nil
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
	if specialValue != "" && strings.Contains(baseName, "{0}") {
		// Replace {0} placeholder with the special value
		return strings.ReplaceAll(baseName, "{0}", specialValue)
	} else if specialValue != "" {
		// Fallback for markets without placeholders
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
func (s *EventsService) ProcessDetailedMarkets(ctx context.Context, eventID int, markets []models.IddaaDetailedMarket) error {
	for _, market := range markets {
		// Convert detailed market to standard market format
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
		err := s.processSingleMarket(ctx, eventID, standardMarket)
		if err != nil {
			s.logger.Error().
				Err(err).
				Int("market_id", market.ID).
				Int("event_id", eventID).
				Msg("Failed to process detailed market")
		}
	}
	return nil
}

// getOrCreateMarketType gets a market type from cache or creates it in database
func (s *EventsService) getOrCreateMarketType(ctx context.Context, marketTypeCode string, market models.IddaaMarket) (database.MarketType, error) {
	// Check cache first
	if cachedMarketType, exists := s.marketTypeCache[marketTypeCode]; exists {
		return cachedMarketType, nil
	}

	// Try to get from database
	marketType, err := s.db.GetMarketType(ctx, marketTypeCode)
	if err == nil {
		// Found in database, add to cache
		s.marketTypeCache[marketTypeCode] = marketType
		return marketType, nil
	}

	// Not found, create new market type
	slug := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(marketTypeCode, "_", "-"), " ", "-"))

	marketType, err = s.db.UpsertMarketType(ctx, database.UpsertMarketTypeParams{
		Code:                  marketTypeCode,
		Name:                  s.getMarketTypeName(market.SubType),
		Slug:                  slug,
		Description:           pgtype.Text{String: s.getMarketTypeDescription(market.SubType, market.SpecialValue), Valid: true},
		IddaaMarketID:         pgtype.Int4{Int32: int32(market.ID), Valid: true},
		IsLive:                pgtype.Bool{Valid: false},
		MarketType:            pgtype.Int4{Int32: int32(market.Type), Valid: true},
		MinMarketDefaultValue: pgtype.Int4{Valid: false},
		MaxMarketLimitValue:   pgtype.Int4{Valid: false},
		Priority:              pgtype.Int4{Valid: false},
		SportType:             pgtype.Int4{Valid: false},
		MarketSubType:         pgtype.Int4{Int32: int32(market.SubType), Valid: true},
		MinDefaultValue:       pgtype.Int4{Valid: false},
		MaxLimitValue:         pgtype.Int4{Valid: false},
		IsActive:              pgtype.Bool{Bool: true, Valid: true},
	})

	if err != nil {
		return database.MarketType{}, fmt.Errorf("failed to upsert market type: %w", err)
	}

	// Add to cache
	s.marketTypeCache[marketTypeCode] = marketType
	return marketType, nil
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

	for _, marketType := range marketTypes {
		s.marketTypeCache[marketType.Code] = marketType
	}

	s.logger.Info().
		Int("count", len(marketTypes)).
		Msg("Market type cache preloaded successfully")
}
