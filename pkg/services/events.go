package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/gosimple/slug"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models"
)

type EventsService struct {
	db     *generated.Queries
	client *IddaaClient
	logger *logger.Logger
	// Market type cache - loaded once at startup
	marketTypes map[string]int32 // code -> id mapping
}

func NewEventsService(db *generated.Queries, client *IddaaClient) *EventsService {
	service := &EventsService{
		db:          db,
		client:      client,
		logger:      logger.New("events-service"),
		marketTypes: make(map[string]int32),
	}

	// Load all market types once at startup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	marketTypes, err := db.ListMarketTypes(ctx)
	if err != nil {
		service.logger.Error().Err(err).Msg("Failed to load market types")
	} else {
		for _, mt := range marketTypes {
			service.marketTypes[mt.Code] = mt.ID
		}
		service.logger.Info().
			Int("count", len(marketTypes)).
			Msg("Market types loaded")
	}

	return service
}

// ProcessEventsResponse processes the API response using bulk operations
func (s *EventsService) ProcessEventsResponse(ctx context.Context, response *models.IddaaEventsResponse) error {
	if !response.IsSuccess || response.Data == nil {
		return fmt.Errorf("invalid API response")
	}

	events := response.Data.Events
	if len(events) == 0 {
		return nil
	}

	log := logger.WithContext(ctx, "process-events")
	log.Info().Int("event_count", len(events)).Msg("Starting bulk event processing")

	startTime := time.Now()

	// Step 1: Bulk process teams
	teamMapping, err := s.bulkProcessTeams(ctx, events)
	if err != nil {
		return fmt.Errorf("failed to process teams: %w", err)
	}

	// Step 2: Bulk process events
	eventMapping, err := s.bulkProcessEvents(ctx, events, teamMapping)
	if err != nil {
		return fmt.Errorf("failed to process events: %w", err)
	}

	// Step 3: Bulk process odds with history
	oddsProcessed, historyCreated, err := s.bulkProcessOdds(ctx, events, eventMapping)
	if err != nil {
		return fmt.Errorf("failed to process odds: %w", err)
	}

	duration := time.Since(startTime)

	// Calculate comprehensive metrics
	historyRatio := float64(0)
	if oddsProcessed > 0 {
		historyRatio = float64(historyCreated) / float64(oddsProcessed) * 100
	}

	log.Info().
		Dur("duration", duration).
		Int("events_processed", len(events)).
		Int("odds_processed", oddsProcessed).
		Int("history_records", historyCreated).
		Float64("history_ratio_pct", historyRatio).
		Float64("events_per_second", float64(len(events))/duration.Seconds()).
		Float64("odds_per_second", float64(oddsProcessed)/duration.Seconds()).
		Msg("Event processing completed with metrics")

	return nil
}

// bulkProcessTeams extracts and bulk upserts all teams
func (s *EventsService) bulkProcessTeams(ctx context.Context, events []models.IddaaEvent) (map[string]int32, error) {
	// Extract unique teams
	teamSet := make(map[string]bool)
	for _, event := range events {
		teamSet[event.HomeTeam] = true
		teamSet[event.AwayTeam] = true
	}

	if len(teamSet) == 0 {
		return make(map[string]int32), nil
	}

	// Prepare arrays for bulk insert
	externalIDs := make([]string, 0, len(teamSet))
	names := make([]string, 0, len(teamSet))
	slugs := make([]string, 0, len(teamSet))

	for teamName := range teamSet {
		externalIDs = append(externalIDs, teamName)
		names = append(names, teamName)
		slugs = append(slugs, slug.Make(teamName))
	}

	// Bulk upsert teams
	teams, err := s.db.BulkUpsertTeams(ctx, generated.BulkUpsertTeamsParams{
		ExternalIds: externalIDs,
		Names:       names,
		Slugs:       slugs,
	})
	if err != nil {
		return nil, fmt.Errorf("bulk team upsert failed: %w", err)
	}

	// Create mapping
	teamMapping := make(map[string]int32)
	for _, team := range teams {
		teamMapping[team.ExternalID] = team.ID
	}

	return teamMapping, nil
}

// bulkProcessEvents bulk upserts all events
func (s *EventsService) bulkProcessEvents(ctx context.Context, events []models.IddaaEvent, teamMapping map[string]int32) (map[string]int32, error) {
	// Pre-fetch leagues
	leagueCache := make(map[int]int32)
	uniqueLeagueIDs := make(map[int]bool)
	for _, event := range events {
		uniqueLeagueIDs[event.CompetitionID] = true
	}

	for leagueID := range uniqueLeagueIDs {
		league, err := s.db.GetLeagueByExternalID(ctx, fmt.Sprintf("%d", leagueID))
		if err == nil {
			leagueCache[leagueID] = league.ID
		}
	}

	// Prepare arrays for bulk insert
	externalIDs := make([]string, 0, len(events))
	leagueIDs := make([]int32, 0, len(events))
	homeTeamIDs := make([]int32, 0, len(events))
	awayTeamIDs := make([]int32, 0, len(events))
	eventDates := make([]pgtype.Timestamp, 0, len(events))
	statuses := make([]string, 0, len(events))
	bulletinIDs := make([]int64, 0, len(events))
	versions := make([]int64, 0, len(events))
	sportIDs := make([]int32, 0, len(events))
	betPrograms := make([]int32, 0, len(events))
	mbcs := make([]int32, 0, len(events))
	hasKingOdds := make([]bool, 0, len(events))
	oddsCounts := make([]int32, 0, len(events))
	hasCombines := make([]bool, 0, len(events))
	isLives := make([]bool, 0, len(events))
	slugs := make([]string, 0, len(events))

	for _, event := range events {
		eventDate := time.Unix(event.Date, 0)
		eventSlug := slug.Make(fmt.Sprintf("%s-%s-%s",
			event.HomeTeam, event.AwayTeam, eventDate.Format("2006-01-02")))

		externalIDs = append(externalIDs, strconv.Itoa(event.ID))
		leagueIDs = append(leagueIDs, leagueCache[event.CompetitionID])
		homeTeamIDs = append(homeTeamIDs, teamMapping[event.HomeTeam])
		awayTeamIDs = append(awayTeamIDs, teamMapping[event.AwayTeam])
		eventDates = append(eventDates, pgtype.Timestamp{Time: eventDate, Valid: true})
		statuses = append(statuses, s.convertEventStatus(event.Status))
		bulletinIDs = append(bulletinIDs, int64(event.BulletinID))
		versions = append(versions, int64(event.Version))
		sportIDs = append(sportIDs, int32(event.SportID))
		betPrograms = append(betPrograms, int32(event.BetProgram))
		mbcs = append(mbcs, int32(event.MBC))
		hasKingOdds = append(hasKingOdds, event.HasKingOdd)
		oddsCounts = append(oddsCounts, int32(event.OddsCount))
		hasCombines = append(hasCombines, event.HasCombine)
		isLives = append(isLives, event.IsLive)
		slugs = append(slugs, eventSlug)
	}

	// Bulk upsert events
	upsertedEvents, err := s.db.BulkUpsertEvents(ctx, generated.BulkUpsertEventsParams{
		ExternalIds: externalIDs,
		LeagueIds:   leagueIDs,
		HomeTeamIds: homeTeamIDs,
		AwayTeamIds: awayTeamIDs,
		EventDates:  eventDates,
		Statuses:    statuses,
		BulletinIds: bulletinIDs,
		Versions:    versions,
		SportIds:    sportIDs,
		BetPrograms: betPrograms,
		Mbcs:        mbcs,
		HasKingOdds: hasKingOdds,
		OddsCounts:  oddsCounts,
		HasCombines: hasCombines,
		IsLives:     isLives,
		Slugs:       slugs,
	})
	if err != nil {
		return nil, fmt.Errorf("bulk event upsert failed: %w", err)
	}

	// Create mapping
	eventMapping := make(map[string]int32)
	for _, event := range upsertedEvents {
		eventMapping[event.ExternalID] = event.ID
	}

	return eventMapping, nil
}

// bulkProcessOdds processes all odds in bulk with history tracking
func (s *EventsService) bulkProcessOdds(ctx context.Context, events []models.IddaaEvent, eventMapping map[string]int32) (int, int, error) {
	// Prepare odds data
	var eventIDs []int32
	var marketTypeIDs []int32
	var outcomes []string
	var oddsValues []float64
	var marketParams [][]byte

	newOddsMap := make(map[string]float64)

	for _, event := range events {
		eventID := eventMapping[fmt.Sprintf("%d", event.ID)]

		for _, market := range event.Markets {
			marketTypeCode := fmt.Sprintf("%d_%d", market.Type, market.SubType)
			marketTypeID, exists := s.marketTypes[marketTypeCode]
			if !exists {
				s.logger.Warn().
					Str("code", marketTypeCode).
					Int("type", market.Type).
					Int("subtype", market.SubType).
					Msg("Unknown market type, skipping")
				continue
			}

			params := models.ExtractMarketParams(market.SpecialValue)
			paramsJSON, _ := json.Marshal(params)

			for _, outcome := range market.Outcomes {
				outcomeStr := s.formatOutcomeName(outcome.Name, market.SubType, market.SpecialValue)

				eventIDs = append(eventIDs, eventID)
				marketTypeIDs = append(marketTypeIDs, marketTypeID)
				outcomes = append(outcomes, outcomeStr)
				oddsValues = append(oddsValues, outcome.Odds)
				marketParams = append(marketParams, paramsJSON)

				// Store for history tracking
				key := fmt.Sprintf("%d-%d-%s", eventID, marketTypeID, outcomeStr)
				newOddsMap[key] = outcome.Odds
			}
		}
	}

	if len(eventIDs) == 0 {
		return 0, 0, nil
	}

	// Get existing odds for comparison
	existingOdds, err := s.db.BulkGetCurrentOddsForComparison(ctx, generated.BulkGetCurrentOddsForComparisonParams{
		EventIds:      eventIDs,
		MarketTypeIds: marketTypeIDs,
		Outcomes:      outcomes,
	})
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get existing odds for comparison")
	}

	// Build map of existing odds
	existingOddsMap := make(map[string]generated.BulkGetCurrentOddsForComparisonRow)
	for _, existing := range existingOdds {
		key := fmt.Sprintf("%v-%v-%s", existing.EventID, existing.MarketTypeID, existing.Outcome)
		existingOddsMap[key] = existing
	}

	// Bulk upsert current odds
	const chunkSize = 1000
	for i := 0; i < len(eventIDs); i += chunkSize {
		end := i + chunkSize
		if end > len(eventIDs) {
			end = len(eventIDs)
		}

		err := s.db.BulkUpsertCurrentOdds(ctx, generated.BulkUpsertCurrentOddsParams{
			EventIds:      eventIDs[i:end],
			MarketTypeIds: marketTypeIDs[i:end],
			Outcomes:      outcomes[i:end],
			OddsValues:    oddsValues[i:end],
			MarketParams:  marketParams[i:end],
		})
		if err != nil {
			s.logger.Error().
				Err(err).
				Int("chunk_start", i).
				Int("chunk_size", end-i).
				Msg("Failed to bulk upsert odds chunk")
		}
	}

	// Create history records for changed odds
	var historyEventIDs []int32
	var historyMarketTypeIDs []int32
	var historyOutcomes []string
	var historyOddsValues []float64
	var historyPreviousValues []float64
	var historyChangeAmounts []float64
	var historyChangePercentages []float64
	var historyMultipliers []float64
	var historyIsReverseMovements []bool
	var historySignificanceLevels []string
	var historyMinutesToKickoffs []int32
	var historyMarketParams [][]byte

	// Process changes
	for i := range eventIDs {
		key := fmt.Sprintf("%d-%d-%s", eventIDs[i], marketTypeIDs[i], outcomes[i])
		newValue := newOddsMap[key]

		if existing, exists := existingOddsMap[key]; exists {
			oldValue := existing.OddsValue
			if math.Abs(newValue-oldValue) > 0.001 {
				// Calculate change metrics
				changeAmount := newValue - oldValue
				changePercentage := (changeAmount / oldValue) * 100
				multiplier := newValue / oldValue

				// Determine significance
				significanceLevel := "normal"
				if math.Abs(changeAmount) >= 0.5 {
					significanceLevel = "extreme"
				} else if math.Abs(changeAmount) >= 0.2 {
					significanceLevel = "high"
				}

				// Calculate minutes to kickoff
				minutesToKickoff := int32(0)
				if existing.EventDate.Valid {
					duration := time.Until(existing.EventDate.Time)
					minutesToKickoff = int32(duration.Minutes())
				}

				// Add to history arrays
				historyEventIDs = append(historyEventIDs, eventIDs[i])
				historyMarketTypeIDs = append(historyMarketTypeIDs, marketTypeIDs[i])
				historyOutcomes = append(historyOutcomes, outcomes[i])
				historyOddsValues = append(historyOddsValues, oddsValues[i])
				historyPreviousValues = append(historyPreviousValues, oldValue)
				historyChangeAmounts = append(historyChangeAmounts, changeAmount)
				historyChangePercentages = append(historyChangePercentages, changePercentage)
				historyMultipliers = append(historyMultipliers, multiplier)
				historyIsReverseMovements = append(historyIsReverseMovements, false) // Simplified
				historySignificanceLevels = append(historySignificanceLevels, significanceLevel)
				historyMinutesToKickoffs = append(historyMinutesToKickoffs, minutesToKickoff)
				historyMarketParams = append(historyMarketParams, marketParams[i])
			}
		}
	}

	// Bulk insert history records if any
	if len(historyEventIDs) > 0 {
		const historyChunkSize = 500
		for i := 0; i < len(historyEventIDs); i += historyChunkSize {
			end := i + historyChunkSize
			if end > len(historyEventIDs) {
				end = len(historyEventIDs)
			}

			err := s.db.BulkInsertOddsHistory(ctx, generated.BulkInsertOddsHistoryParams{
				EventIds:           historyEventIDs[i:end],
				MarketTypeIds:      historyMarketTypeIDs[i:end],
				Outcomes:           historyOutcomes[i:end],
				OddsValues:         historyOddsValues[i:end],
				PreviousValues:     historyPreviousValues[i:end],
				ChangeAmounts:      historyChangeAmounts[i:end],
				ChangePercentages:  historyChangePercentages[i:end],
				Multipliers:        historyMultipliers[i:end],
				IsReverseMovements: historyIsReverseMovements[i:end],
				SignificanceLevels: historySignificanceLevels[i:end],
				MinutesToKickoffs:  historyMinutesToKickoffs[i:end],
				MarketParams:       historyMarketParams[i:end],
			})
			if err != nil {
				s.logger.Error().
					Err(err).
					Int("history_chunk_size", end-i).
					Msg("Failed to insert odds history")
			}
		}

		s.logger.Info().
			Int("history_records", len(historyEventIDs)).
			Msg("Created odds history records")
	}

	return len(eventIDs), len(historyEventIDs), nil
}

// Helper methods

func (s *EventsService) formatOutcomeName(name string, subType int, specialValue string) string {
	if specialValue == "" {
		return name
	}
	// For Over/Under markets, include the special value
	if subType == 60 || subType == 101 || subType == 603 || subType == 604 {
		return fmt.Sprintf("%s %s", name, specialValue)
	}
	return fmt.Sprintf("%s (%s)", name, specialValue)
}

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

// GetActiveSports returns all active sports from the database
func (s *EventsService) GetActiveSports(ctx context.Context) ([]generated.Sport, error) {
	return s.db.ListSports(ctx)
}

// ProcessDetailedMarkets processes detailed markets from single event endpoint using bulk operations
func (s *EventsService) ProcessDetailedMarkets(ctx context.Context, eventID int, markets []models.IddaaDetailedMarket) error {
	// Convert to standard format
	standardMarkets := make([]models.IddaaMarket, len(markets))
	for i, market := range markets {
		standardMarkets[i] = models.IddaaMarket{
			ID:           market.ID,
			Type:         market.Type,
			SubType:      market.SubType,
			SpecialValue: market.SpecialValue,
			Outcomes:     make([]models.IddaaOutcome, len(market.Outcomes)),
		}
		for j, outcome := range market.Outcomes {
			standardMarkets[i].Outcomes[j] = models.IddaaOutcome{
				Number: outcome.Number,
				Odds:   outcome.Odds,
				Name:   outcome.Name,
			}
		}
	}

	// Use optimized bulk processing
	return s.processMarketsBulk(ctx, eventID, standardMarkets)
}

// // processMarkets handles market processing for single events (used by detailed odds job)
// func (s *EventsService) processMarkets(ctx context.Context, eventID int, markets []models.IddaaMarket) error {
// 	// First, get ALL current odds for this event in one query
// 	currentOdds, err := s.db.GetCurrentOdds(ctx, int32(eventID))
// 	if err != nil {
// 		s.logger.Warn().Err(err).Int("event_id", eventID).Msg("Failed to get current odds")
// 	}

// 	// Build a map for O(1) lookups
// 	currentOddsMap := make(map[string]generated.GetCurrentOddsRow)
// 	for _, odd := range currentOdds {
// 		key := fmt.Sprintf("%d-%s", *odd.MarketTypeID, odd.Outcome)
// 		currentOddsMap[key] = odd
// 	}

// 	// Now process markets without N+1 queries
// 	for _, market := range markets {
// 		marketTypeCode := fmt.Sprintf("%d_%d", market.Type, market.SubType)
// 		marketTypeID, exists := s.marketTypes[marketTypeCode]
// 		if !exists {
// 			s.logger.Warn().
// 				Str("code", marketTypeCode).
// 				Int("event_id", eventID).
// 				Msg("Unknown market type, skipping")
// 			continue
// 		}

// 		params := models.ExtractMarketParams(market.SpecialValue)
// 		paramsJSON, _ := json.Marshal(params)

// 		for _, outcome := range market.Outcomes {
// 			outcomeStr := s.formatOutcomeName(outcome.Name, market.SubType, market.SpecialValue)

// 			// Look up current odds from our map (O(1) instead of a database query)
// 			key := fmt.Sprintf("%d-%s", marketTypeID, outcomeStr)
// 			currentOdd, hasExisting := currentOddsMap[key]

// 			var openingValue, previousValue, highestValue, lowestValue float64

// 			if hasExisting {
// 				// Update existing - use native float64 pointers
// 				if currentOdd.OpeningValue != nil {
// 					openingValue = *currentOdd.OpeningValue
// 				}

// 				if currentOdd.HighestValue != nil {
// 					highestValue = *currentOdd.HighestValue
// 				}

// 				if currentOdd.LowestValue != nil {
// 					lowestValue = *currentOdd.LowestValue
// 				}

// 				previousValue = currentOdd.OddsValue

// 				// Update high/low
// 				if outcome.Odds > highestValue {
// 					highestValue = outcome.Odds
// 				}
// 				if outcome.Odds < lowestValue {
// 					lowestValue = outcome.Odds
// 				}
// 			} else {
// 				// New odds
// 				openingValue = outcome.Odds
// 				highestValue = outcome.Odds
// 				lowestValue = outcome.Odds
// 			}

// 			// Upsert current odds
// 			_, err = s.db.UpsertCurrentOdds(ctx, generated.UpsertCurrentOddsParams{
// 				EventID:      int32(eventID),
// 				MarketTypeID: marketTypeID,
// 				Outcome:      outcomeStr,
// 				OddsValue:    outcome.Odds,
// 				OpeningValue: openingValue,
// 				HighestValue: highestValue,
// 				LowestValue:  lowestValue,
// 				WinningOdds:  0,
// 				MarketParams: paramsJSON,
// 			})
// 			if err != nil {
// 				s.logger.Error().
// 					Err(err).
// 					Int("event_id", eventID).
// 					Str("outcome", outcomeStr).
// 					Msg("Failed to upsert odds")
// 				continue
// 			}

// 			// Create history if changed
// 			if hasExisting && math.Abs(previousValue-outcome.Odds) > 0.001 {
// 				_, err = s.db.CreateOddsHistory(ctx, generated.CreateOddsHistoryParams{
// 					EventID:       int32(eventID),
// 					MarketTypeID:  marketTypeID,
// 					Outcome:       outcomeStr,
// 					OddsValue:     outcome.Odds,
// 					PreviousValue: previousValue,
// 					WinningOdds:   0,
// 					MarketParams:  paramsJSON,
// 				})
// 				if err != nil {
// 					s.logger.Error().
// 						Err(err).
// 						Int("event_id", eventID).
// 						Str("outcome", outcomeStr).
// 						Msg("Failed to create odds history")
// 				}
// 			}
// 		}
// 	}

// 	return nil
// }

// processMarketsBulk handles market processing using bulk UNNEST operations for maximum performance
func (s *EventsService) processMarketsBulk(ctx context.Context, eventID int, markets []models.IddaaMarket) error {
	// Prepare slices for bulk operations
	var (
		eventIDs      []int32
		marketTypeIDs []int32
		outcomes      []string
		oddsValues    []float64
		marketParams  [][]byte

		// For history tracking
		histEventIDs      []int32
		histMarketTypeIDs []int32
		histOutcomes      []string
		histOddsValues    []float64
		histPrevValues    []float64
		histChangeAmounts []float64
		histChangePcts    []float64
		histMultipliers   []float64
		histReverseMoves  []bool
		histSigLevels     []string
		histMinutesToKO   []int32
		histMarketParams  [][]byte
	)

	// Build arrays for all outcomes
	for _, market := range markets {
		marketTypeCode := fmt.Sprintf("%d_%d", market.Type, market.SubType)
		marketTypeID, exists := s.marketTypes[marketTypeCode]
		if !exists {
			s.logger.Warn().
				Str("code", marketTypeCode).
				Int("event_id", eventID).
				Msg("Unknown market type, skipping")
			continue
		}

		params := models.ExtractMarketParams(market.SpecialValue)
		paramsJSON, _ := json.Marshal(params)

		for _, outcome := range market.Outcomes {
			outcomeStr := s.formatOutcomeName(outcome.Name, market.SubType, market.SpecialValue)

			// Add to bulk arrays
			eventIDs = append(eventIDs, int32(eventID))
			marketTypeIDs = append(marketTypeIDs, marketTypeID)
			outcomes = append(outcomes, outcomeStr)
			oddsValues = append(oddsValues, outcome.Odds)
			marketParams = append(marketParams, paramsJSON)
		}
	}

	// If no valid markets, return early
	if len(eventIDs) == 0 {
		return nil
	}

	// Get current odds for comparison (bulk query)
	currentOddsRows, err := s.db.BulkGetCurrentOddsForComparison(ctx, generated.BulkGetCurrentOddsForComparisonParams{
		EventIds:      eventIDs,
		MarketTypeIds: marketTypeIDs,
		Outcomes:      outcomes,
	})
	if err != nil {
		s.logger.Warn().Err(err).Int("event_id", eventID).Msg("Failed to get current odds for comparison")
	}

	// Build map of current odds
	type oddsKey struct {
		eventID      int32
		marketTypeID int32
		outcome      string
	}
	currentOddsMap := make(map[oddsKey]generated.BulkGetCurrentOddsForComparisonRow)
	for _, odd := range currentOddsRows {
		// Skip if required fields are nil
		if odd.EventID == nil || odd.MarketTypeID == nil {
			continue
		}
		key := oddsKey{
			eventID:      *odd.EventID,
			marketTypeID: *odd.MarketTypeID,
			outcome:      odd.Outcome,
		}
		currentOddsMap[key] = odd
	}

	// Prepare history records for odds that changed
	for i := range eventIDs {
		key := oddsKey{
			eventID:      eventIDs[i],
			marketTypeID: marketTypeIDs[i],
			outcome:      outcomes[i],
		}

		if currentOdd, exists := currentOddsMap[key]; exists {
			// Check if odds changed
			if math.Abs(currentOdd.OddsValue-oddsValues[i]) > 0.001 {
				// Calculate change metrics
				changeAmount := oddsValues[i] - currentOdd.OddsValue
				var changePct, multiplier float64

				if currentOdd.OddsValue > 0 {
					changePct = (changeAmount / currentOdd.OddsValue) * 100
					multiplier = oddsValues[i] / currentOdd.OddsValue
				} else {
					multiplier = 1
				}

				// Determine significance level
				var sigLevel string
				absPct := math.Abs(changePct)
				switch {
				case absPct >= 20:
					sigLevel = "extreme"
				case absPct >= 10:
					sigLevel = "high"
				default:
					sigLevel = "normal"
				}

				// Calculate minutes to kickoff
				minutesToKO := int32(time.Until(currentOdd.EventDate.Time).Minutes())

				// Add to history arrays
				histEventIDs = append(histEventIDs, eventIDs[i])
				histMarketTypeIDs = append(histMarketTypeIDs, marketTypeIDs[i])
				histOutcomes = append(histOutcomes, outcomes[i])
				histOddsValues = append(histOddsValues, oddsValues[i])
				histPrevValues = append(histPrevValues, currentOdd.OddsValue)
				histChangeAmounts = append(histChangeAmounts, changeAmount)
				histChangePcts = append(histChangePcts, changePct)
				histMultipliers = append(histMultipliers, multiplier)
				histReverseMoves = append(histReverseMoves, false) // TODO: implement reverse movement detection
				histSigLevels = append(histSigLevels, sigLevel)
				histMinutesToKO = append(histMinutesToKO, minutesToKO)
				histMarketParams = append(histMarketParams, marketParams[i])
			}
		}
	}

	// Bulk upsert current odds
	err = s.db.BulkUpsertCurrentOdds(ctx, generated.BulkUpsertCurrentOddsParams{
		EventIds:      eventIDs,
		MarketTypeIds: marketTypeIDs,
		Outcomes:      outcomes,
		OddsValues:    oddsValues,
		MarketParams:  marketParams,
	})
	if err != nil {
		return fmt.Errorf("failed to bulk upsert current odds: %w", err)
	}

	// Bulk insert history records if any odds changed
	if len(histEventIDs) > 0 {
		err = s.db.BulkInsertOddsHistory(ctx, generated.BulkInsertOddsHistoryParams{
			EventIds:           histEventIDs,
			MarketTypeIds:      histMarketTypeIDs,
			Outcomes:           histOutcomes,
			OddsValues:         histOddsValues,
			PreviousValues:     histPrevValues,
			ChangeAmounts:      histChangeAmounts,
			ChangePercentages:  histChangePcts,
			Multipliers:        histMultipliers,
			IsReverseMovements: histReverseMoves,
			SignificanceLevels: histSigLevels,
			MinutesToKickoffs:  histMinutesToKO,
			MarketParams:       histMarketParams,
		})
		if err != nil {
			s.logger.Error().
				Err(err).
				Int("event_id", eventID).
				Int("history_count", len(histEventIDs)).
				Msg("Failed to bulk insert odds history")
		}
	}

	s.logger.Debug().
		Int("event_id", eventID).
		Int("total_odds", len(eventIDs)).
		Int("changed_odds", len(histEventIDs)).
		Msg("Bulk processed event markets")

	return nil
}
