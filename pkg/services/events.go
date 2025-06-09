package services

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sony/gobreaker"

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
	// Configuration for concurrent processing
	maxConcurrency int
	// Metrics for monitoring
	metrics *EventsMetrics
	// Circuit breaker for API calls
	circuitBreaker *gobreaker.CircuitBreaker
	// Shutdown management
	shutdownOnce sync.Once
	done         chan struct{}
	wg           sync.WaitGroup
}

type EventsMetrics struct {
	eventsProcessed        atomic.Int64
	eventsSucceeded        atomic.Int64
	eventsFailed           atomic.Int64
	detailedOddsFetched    atomic.Int64
	detailedOddsFailures   atomic.Int64
	marketTypeCacheHits    atomic.Int64
	marketTypeCacheMisses  atomic.Int64
	totalProcessingTime    atomic.Int64 // in nanoseconds
	lastProcessingDuration atomic.Int64 // in nanoseconds
}

func NewEventsService(db *database.Queries, client *IddaaClient) *EventsService {
	service := &EventsService{
		db:              db,
		client:          client,
		rateLimiter:     time.NewTicker(100 * time.Millisecond), // Max 10 requests per second
		logger:          logger.New("events-service"),
		marketTypeCache: make(map[string]database.MarketType),
		maxConcurrency:  5, // Process up to 5 events concurrently
		metrics:         &EventsMetrics{},
		done:            make(chan struct{}),
	}

	// Configure circuit breaker
	service.circuitBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "events-api",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
	})

	// Pre-populate market type cache on startup
	service.wg.Add(1)
	go func() {
		defer service.wg.Done()
		service.preloadMarketTypeCache()
	}()

	return service
}

// Stop gracefully shuts down the service
func (s *EventsService) Stop() {
	s.shutdownOnce.Do(func() {
		close(s.done)
		s.rateLimiter.Stop()
		s.wg.Wait()
	})
}

// ProcessEventsResponse processes the API response and saves events, teams, and odds
func (s *EventsService) ProcessEventsResponse(ctx context.Context, response *models.IddaaEventsResponse) error {
	if !response.IsSuccess || response.Data == nil {
		return fmt.Errorf("invalid API response: isSuccess=%t", response.IsSuccess)
	}

	startTime := time.Now()
	log := logger.WithContext(ctx, "process-events")

	// Create a child context that we can cancel
	processCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Monitor for service shutdown
	go func() {
		select {
		case <-s.done:
			cancel()
		case <-processCtx.Done():
			// Context already cancelled
		}
	}()

	var processErrors []error
	successCount := int64(0)

	// Add concurrency control to process events in parallel
	sem := make(chan struct{}, s.maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Group events by sport for better processing
	eventsBySport := s.groupEventsBySport(response.Data.Events)

	for sportID, events := range eventsBySport {
		log.Info().
			Int("sport_id", sportID).
			Int("event_count", len(events)).
			Msg("Processing events for sport")

		for _, event := range events {
			// Check if we should stop processing
			select {
			case <-processCtx.Done():
				log.Warn().
					Err(processCtx.Err()).
					Int64("processed", successCount).
					Int("remaining", len(response.Data.Events)-int(successCount)).
					Msg("Processing cancelled, stopping event processing")
				goto ProcessingComplete
			default:
				// Continue processing
			}

			wg.Add(1)
			sem <- struct{}{} // Acquire semaphore

			go func(event models.IddaaEvent) {
				defer wg.Done()
				defer func() { <-sem }() // Release semaphore

				// Create a timeout context for each event
				eventCtx, cancel := context.WithTimeout(processCtx, 60*time.Second)
				defer cancel()

				// Process event with metrics
				err := s.processEvent(eventCtx, event)

				s.metrics.eventsProcessed.Add(1)
				if err != nil {
					mu.Lock()
					processErrors = append(processErrors, err)
					mu.Unlock()
					s.metrics.eventsFailed.Add(1)
					log.Error().
						Err(err).
						Int("event_id", event.ID).
						Str("home_team", event.HomeTeam).
						Str("away_team", event.AwayTeam).
						Msg("Failed to process event")
				} else {
					atomic.AddInt64(&successCount, 1)
					s.metrics.eventsSucceeded.Add(1)
				}
			}(event)
		}
	}

ProcessingComplete:
	wg.Wait()

	// Update metrics
	duration := time.Since(startTime)
	s.metrics.lastProcessingDuration.Store(duration.Nanoseconds())
	s.metrics.totalProcessingTime.Add(duration.Nanoseconds())

	// Log summary with detailed metrics
	log.Info().
		Int64("success_count", successCount).
		Int("error_count", len(processErrors)).
		Int("total_events", len(response.Data.Events)).
		Dur("processing_time", duration).
		Float64("events_per_second", float64(successCount)/duration.Seconds()).
		Msg("Completed processing events response")

	// Return error if all events failed
	if successCount == 0 && len(processErrors) > 0 {
		return fmt.Errorf("all events failed to process: %w", processErrors[0])
	}

	return nil
}

// processEvent processes a single event
func (s *EventsService) processEvent(ctx context.Context, event models.IddaaEvent) error {
	// Create a span for tracing
	log := s.logger.With().
		Int("event_id", event.ID).
		Str("home_team", event.HomeTeam).
		Str("away_team", event.AwayTeam).
		Logger()

	// Process home and away teams with batch optimization
	homeTeamID, awayTeamID, err := s.processTeams(ctx, event.HomeTeam, event.AwayTeam)
	if err != nil {
		return fmt.Errorf("failed to process teams: %w", err)
	}

	// Get competition ID
	competitionID, err := s.getCompetitionID(ctx, event.CompetitionID)
	if err != nil {
		return fmt.Errorf("competition %d: %w", event.CompetitionID, err)
	}

	eventDate := time.Unix(event.Date, 0)
	statusStr := s.convertEventStatus(event.Status)

	// Upsert event
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
		return fmt.Errorf("event %d: %w", event.ID, err)
	}

	// Skip detailed odds for certain sports or if we're running low on time
	skipDetailedOdds := s.shouldSkipDetailedOdds(ctx, event)

	if skipDetailedOdds {
		log.Debug().
			Bool("skip_detailed", true).
			Msg("Skipping detailed odds fetch")
		return s.processMarkets(ctx, int(eventRecord.ID), event.Markets)
	}

	// Try to fetch detailed odds with a shorter timeout
	detailCtx, detailCancel := context.WithTimeout(ctx, 20*time.Second)
	err = s.fetchAndProcessDetailedOdds(detailCtx, int(eventRecord.ID), event.ID)
	detailCancel()

	if err != nil {
		s.metrics.detailedOddsFailures.Add(1)

		log.Warn().
			Err(err).
			Msg("Failed to fetch detailed odds, using bulk markets")

		// Fallback to bulk markets
		return s.processMarkets(ctx, int(eventRecord.ID), event.Markets)
	}

	s.metrics.detailedOddsFetched.Add(1)

	return nil
}

// processTeams processes both teams in a single transaction for better performance
func (s *EventsService) processTeams(ctx context.Context, homeTeam, awayTeam string) (homeID, awayID int, err error) {
	// Process both teams in parallel
	var wg sync.WaitGroup
	var homeErr, awayErr error

	wg.Add(2)

	go func() {
		defer wg.Done()
		homeID, homeErr = s.upsertTeam(ctx, homeTeam, homeTeam)
	}()

	go func() {
		defer wg.Done()
		awayID, awayErr = s.upsertTeam(ctx, awayTeam, awayTeam)
	}()

	wg.Wait()

	if homeErr != nil {
		return 0, 0, fmt.Errorf("home team %s: %w", homeTeam, homeErr)
	}
	if awayErr != nil {
		return 0, 0, fmt.Errorf("away team %s: %w", awayTeam, awayErr)
	}

	return homeID, awayID, nil
}

// shouldSkipDetailedOdds determines if we should skip fetching detailed odds
func (s *EventsService) shouldSkipDetailedOdds(ctx context.Context, event models.IddaaEvent) bool {
	// Skip detailed odds for sports with many markets (like MMA, Tennis)
	skipDetailedOddsSports := map[int]bool{
		117: true, // MMA
		5:   true, // Tennis
		// Add other sports as needed
	}

	if skipDetailedOddsSports[event.SportID] {
		return true
	}

	// Skip if we're running low on time
	if deadline, ok := ctx.Deadline(); ok {
		timeRemaining := time.Until(deadline)
		if timeRemaining < 30*time.Second {
			return true
		}
	}

	// Skip for events with too many markets
	if event.OddsCount > 100 {
		return true
	}

	return false
}

// groupEventsBySport groups events by sport ID for better processing
func (s *EventsService) groupEventsBySport(events []models.IddaaEvent) map[int][]models.IddaaEvent {
	grouped := make(map[int][]models.IddaaEvent)
	for _, event := range events {
		grouped[event.SportID] = append(grouped[event.SportID], event)
	}
	return grouped
}

// fetchAndProcessDetailedOdds fetches detailed odds for a specific event and processes them
func (s *EventsService) fetchAndProcessDetailedOdds(ctx context.Context, eventID int, externalEventID int) error {
	// Rate limit API calls to avoid overwhelming the server
	s.mutex.Lock()
	select {
	case <-s.rateLimiter.C:
		s.mutex.Unlock()
	case <-ctx.Done():
		s.mutex.Unlock()
		return ctx.Err()
	}

	// Use circuit breaker for API call
	result, err := s.circuitBreaker.Execute(func() (interface{}, error) {
		return s.client.GetSingleEvent(externalEventID)
	})

	if err != nil {
		return fmt.Errorf("circuit breaker error: %w", err)
	}

	singleEvent, ok := result.(*models.IddaaSingleEventResponse)
	if !ok {
		return fmt.Errorf("unexpected response type")
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

	// Process the detailed markets
	return s.processMarkets(ctx, eventID, detailedMarkets)
}

// processMarkets processes all markets and their odds for an event with batching
func (s *EventsService) processMarkets(ctx context.Context, eventID int, markets []models.IddaaMarket) error {
	// Batch process markets for better performance
	const batchSize = 10

	for i := 0; i < len(markets); i += batchSize {
		end := i + batchSize
		if end > len(markets) {
			end = len(markets)
		}

		batch := markets[i:end]

		// Process batch in parallel
		var wg sync.WaitGroup
		var mu sync.Mutex
		var batchErrors []error

		for _, market := range batch {
			wg.Add(1)

			go func(market models.IddaaMarket) {
				defer wg.Done()

				err := s.processSingleMarketOptimized(ctx, eventID, market)
				if err != nil {
					mu.Lock()
					batchErrors = append(batchErrors, err)
					mu.Unlock()

					s.logger.Error().
						Err(err).
						Int("event_id", eventID).
						Int("market_id", market.ID).
						Msg("Failed to process market")
				}
			}(market)
		}

		wg.Wait()

		// If too many errors in batch, return early
		if len(batchErrors) > len(batch)/2 {
			return fmt.Errorf("too many failures in batch: %d/%d markets failed", len(batchErrors), len(batch))
		}
	}

	return nil
}

// processSingleMarketOptimized is an optimized version of processSingleMarket
func (s *EventsService) processSingleMarketOptimized(ctx context.Context, eventID int, market models.IddaaMarket) error {
	marketTypeCode := fmt.Sprintf("%d_%d", market.Type, market.SubType)

	// Get or create market type with caching
	marketType, err := s.getOrCreateMarketType(ctx, marketTypeCode, market)
	if err != nil {
		return fmt.Errorf("market type %s: %w", marketTypeCode, err)
	}

	// Batch process outcomes
	outcomeParams := make([]database.UpsertCurrentOddsParams, 0, len(market.Outcomes))
	historyParams := make([]database.CreateOddsHistoryParams, 0)

	// Get current odds for all outcomes at once
	currentOdds, err := s.db.GetCurrentOddsByMarket(ctx, database.GetCurrentOddsByMarketParams{
		EventID:      pgtype.Int4{Int32: int32(eventID), Valid: true},
		MarketTypeID: pgtype.Int4{Int32: marketType.ID, Valid: true},
	})
	if err != nil && err.Error() != "no rows in result set" {
		// Log but continue without history tracking
		s.logger.Warn().
			Err(err).
			Int("event_id", eventID).
			Msg("Failed to get current odds, continuing without history")
		currentOdds = []database.GetCurrentOddsByMarketRow{}
	}

	// Create a map for quick lookup
	currentOddsMap := make(map[string]database.GetCurrentOddsByMarketRow)
	for _, odds := range currentOdds {
		currentOddsMap[odds.Outcome] = odds
	}

	// Process each outcome
	for _, outcome := range market.Outcomes {
		outcomeStr := s.formatOutcomeName(outcome.Name, market.SubType, market.SpecialValue)

		// Convert odds to pgtype.Numeric
		oddsNumeric := pgtype.Numeric{}
		if err := oddsNumeric.Scan(fmt.Sprintf("%.3f", outcome.Odds)); err != nil {
			s.logger.Error().
				Err(err).
				Float64("odds_value", outcome.Odds).
				Msg("Failed to convert odds to numeric")
			continue
		}

		var openingValue pgtype.Numeric
		var previousValue pgtype.Numeric
		var highestValue pgtype.Numeric
		var lowestValue pgtype.Numeric
		hasExistingOdds := false

		// Check if we have existing odds
		if existing, found := currentOddsMap[outcomeStr]; found {
			hasExistingOdds = true
			openingValue = existing.OpeningValue
			previousValue = existing.OddsValue

			// Check if odds actually changed
			prevFloat, _ := previousValue.Float64Value()
			if prevFloat.Valid && math.Abs(prevFloat.Float64-outcome.Odds) < 0.001 {
				continue // Skip if odds haven't changed
			}

			// Update highest and lowest values
			highestValue = existing.HighestValue
			lowestValue = existing.LowestValue

			// Check if current odds are higher than highest
			highestFloat, _ := highestValue.Float64Value()
			if !highestFloat.Valid || outcome.Odds > highestFloat.Float64 {
				highestValue = oddsNumeric
			}

			// Check if current odds are lower than lowest
			lowestFloat, _ := lowestValue.Float64Value()
			if !lowestFloat.Valid || outcome.Odds < lowestFloat.Float64 {
				lowestValue = oddsNumeric
			}
		} else {
			openingValue = oddsNumeric
			previousValue.Valid = false
			highestValue = oddsNumeric
			lowestValue = oddsNumeric
		}

		// Add to batch
		outcomeParams = append(outcomeParams, database.UpsertCurrentOddsParams{
			EventID:      pgtype.Int4{Int32: int32(eventID), Valid: true},
			MarketTypeID: pgtype.Int4{Int32: marketType.ID, Valid: true},
			Outcome:      outcomeStr,
			OddsValue:    oddsNumeric,
			OpeningValue: openingValue,
			HighestValue: highestValue,
			LowestValue:  lowestValue,
			WinningOdds:  pgtype.Numeric{Valid: false},
		})

		// Add history record if odds changed
		if hasExistingOdds && previousValue.Valid {
			historyParams = append(historyParams, database.CreateOddsHistoryParams{
				EventID:       pgtype.Int4{Int32: int32(eventID), Valid: true},
				MarketTypeID:  pgtype.Int4{Int32: marketType.ID, Valid: true},
				Outcome:       outcomeStr,
				OddsValue:     oddsNumeric,
				PreviousValue: previousValue,
				WinningOdds:   pgtype.Numeric{Valid: false},
			})
		}
	}

	// Batch upsert current odds
	for _, params := range outcomeParams {
		if _, err := s.db.UpsertCurrentOdds(ctx, params); err != nil {
			return fmt.Errorf("failed to upsert odds batch: %w", err)
		}
	}

	// Batch create history records
	for _, params := range historyParams {
		if _, err := s.db.CreateOddsHistory(ctx, params); err != nil {
			s.logger.Warn().
				Err(err).
				Str("outcome", params.Outcome).
				Msg("Failed to create odds history")
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

// GetMetrics returns the current metrics
func (s *EventsService) GetMetrics() map[string]interface{} {
	cacheHits := s.metrics.marketTypeCacheHits.Load()
	cacheMisses := s.metrics.marketTypeCacheMisses.Load()
	total := cacheHits + cacheMisses

	cacheHitRate := float64(0)
	if total > 0 {
		cacheHitRate = float64(cacheHits) / float64(total) * 100
	}

	return map[string]interface{}{
		"events_processed":         s.metrics.eventsProcessed.Load(),
		"events_succeeded":         s.metrics.eventsSucceeded.Load(),
		"events_failed":            s.metrics.eventsFailed.Load(),
		"detailed_odds_fetched":    s.metrics.detailedOddsFetched.Load(),
		"detailed_odds_failures":   s.metrics.detailedOddsFailures.Load(),
		"cache_hit_rate":           cacheHitRate,
		"last_processing_duration": time.Duration(s.metrics.lastProcessingDuration.Load()).Seconds(),
		"total_processing_time":    time.Duration(s.metrics.totalProcessingTime.Load()).Seconds(),
		"circuit_breaker_state":    s.circuitBreaker.State().String(),
	}
}

// upsertTeam creates or updates a team record with retry logic
func (s *EventsService) upsertTeam(ctx context.Context, teamName, externalID string) (int, error) {
	var team database.Team
	var err error

	// Retry logic for transient failures
	for attempt := 0; attempt < 3; attempt++ {
		team, err = s.db.UpsertTeam(ctx, database.UpsertTeamParams{
			ExternalID: externalID,
			Name:       teamName,
			Country:    pgtype.Text{Valid: false},
			LogoUrl:    pgtype.Text{Valid: false},
		})

		if err == nil {
			return int(team.ID), nil
		}

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}

		// Exponential backoff
		if attempt < 2 {
			time.Sleep(time.Duration(50<<attempt) * time.Millisecond)
		}
	}

	return 0, fmt.Errorf("failed after %d attempts: %w", 3, err)
}

// getCompetitionID retrieves the competition ID, returns 0 if not found
func (s *EventsService) getCompetitionID(ctx context.Context, iddaaCompetitionID int) (int, error) {
	// Try to find league by external ID
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
func (s *EventsService) ProcessDetailedMarkets(ctx context.Context, eventID int, markets []models.IddaaDetailedMarket) error {
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
		err := s.processSingleMarketOptimized(ctx, eventID, standardMarket)
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

// getOrCreateMarketType gets a market type from cache or creates it in database
func (s *EventsService) getOrCreateMarketType(ctx context.Context, marketTypeCode string, market models.IddaaMarket) (database.MarketType, error) {
	// Check cache first
	s.marketTypeMutex.RLock()
	if cachedMarketType, exists := s.marketTypeCache[marketTypeCode]; exists {
		s.marketTypeMutex.RUnlock()
		s.metrics.marketTypeCacheHits.Add(1)
		return cachedMarketType, nil
	}
	s.marketTypeMutex.RUnlock()

	s.metrics.marketTypeCacheMisses.Add(1)

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

	// Generate slug from code
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

	// Monitor for shutdown
	go func() {
		select {
		case <-s.done:
			cancel()
		case <-ctx.Done():
			// Context already cancelled
		}
	}()

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

// HealthCheck verifies the service is healthy
func (s *EventsService) HealthCheck(ctx context.Context) error {
	// Check circuit breaker state
	if s.circuitBreaker.State() == gobreaker.StateOpen {
		return fmt.Errorf("circuit breaker is open")
	}

	// Check if we have cached market types
	s.marketTypeMutex.RLock()
	cacheSize := len(s.marketTypeCache)
	s.marketTypeMutex.RUnlock()

	if cacheSize == 0 {
		return fmt.Errorf("market type cache is empty")
	}

	return nil
}
