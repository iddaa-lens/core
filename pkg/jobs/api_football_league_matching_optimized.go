package jobs

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/iddaa-lens/core/pkg/apifootball"
	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models"
	"github.com/iddaa-lens/core/pkg/services"
)

// APIFootballLeagueMatchingJobV2 - Optimized version
type APIFootballLeagueMatchingJobV2 struct {
	db         *generated.Queries
	matcher    *services.TeamLeagueMatcher
	apiclient  *apifootball.Client
	translator *services.TeamLeagueMatcher
	apiKey     string
	logger     *logger.Logger

	// Pre-allocated for performance
	translationCache map[string]string
	cacheMutex       sync.RWMutex
}

// NewAPIFootballLeagueMatchingJobV2 creates optimized league matching job
func NewAPIFootballLeagueMatchingJobV2(db *generated.Queries) *APIFootballLeagueMatchingJobV2 {
	apiKey := os.Getenv("API_FOOTBALL_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")

	return &APIFootballLeagueMatchingJobV2{
		db:               db,
		matcher:          services.NewTeamLeagueMatcher(openaiKey),
		apiclient:        apifootball.NewClient(apifootball.DefaultConfig(apiKey)),
		translator:       services.NewTeamLeagueMatcher(openaiKey),
		apiKey:           apiKey,
		logger:           logger.New("api-football-league-matching-v2"),
		translationCache: make(map[string]string, 1000), // Pre-size for typical workload
	}
}

func (j *APIFootballLeagueMatchingJobV2) Name() string {
	return "api_football_league_matching"
}

func (j *APIFootballLeagueMatchingJobV2) Schedule() string {
	return "0 3 * * 2" // Weekly on Tuesdays at 3 AM
}

func (j *APIFootballLeagueMatchingJobV2) Execute(ctx context.Context) error {
	log := logger.WithContext(ctx, "league-matching-v2")
	log.Info().Msg("Starting optimized league matching job")

	// Early exit if no API key
	if j.apiKey == "" {
		log.Warn().Msg("API key missing, skipping")
		return nil
	}

	// 1. Fetch unmapped leagues only
	log.Info().Msg("Fetching unmapped leagues...")
	unmappedLeagues, err := j.db.ListUnmappedFootballLeagues(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch unmapped leagues")
		return err
	}

	if len(unmappedLeagues) == 0 {
		log.Info().Msg("No unmapped leagues")
		return nil
	}

	log.Info().
		Int("unmapped_count", len(unmappedLeagues)).
		Msg("Unmapped leagues fetched successfully")

	// 2. Batch translate all leagues at once
	log.Info().Msg("Starting batch translation...")
	translations := j.batchTranslateLeagues(ctx, unmappedLeagues)
	log.Info().
		Int("translation_count", len(translations)).
		Msg("Batch translation completed")

	// 3. Process matches using search-based matching
	log.Info().Msg("Processing matches using API-Football search...")
	results := j.processMatchesWithSearch(ctx, unmappedLeagues, translations)
	log.Info().
		Int("results_count", len(results)).
		Msg("Search-based match processing completed")

	// 4. Bulk insert all successful matches
	log.Info().Msg("Storing results...")
	err = j.bulkStoreResults(ctx, results)
	if err != nil {
		log.Error().Err(err).Msg("Failed to store results")
		return err
	}

	log.Info().Msg("Job completed successfully")
	return nil
}

// // fetchAllData fetches both unmapped leagues and API leagues in parallel
// func (j *APIFootballLeagueMatchingJobV2) fetchAllData(ctx context.Context) ([]generated.League, []models.SearchResult, error) {
// 	j.logger.Debug().Msg("Starting parallel data fetch")

// 	var unmappedLeagues []generated.League
// 	var apiLeagues []models.SearchResult
// 	var unmappedErr, apiErr error

// 	// Parallel fetch
// 	var wg sync.WaitGroup
// 	wg.Add(2)

// 	go func() {
// 		defer wg.Done()
// 		j.logger.Debug().Msg("Fetching unmapped leagues from generated...")
// 		unmappedLeagues, unmappedErr = j.db.ListUnmappedFootballLeagues(ctx)
// 		j.logger.Debug().
// 			Int("count", len(unmappedLeagues)).
// 			Err(unmappedErr).
// 			Msg("Database fetch completed")
// 	}()

// 	go func() {
// 		defer wg.Done()
// 		j.logger.Debug().Msg("Fetching leagues from API-Football...")
// 		leagues, err := j.apiclient.GetCurrentLeagues(ctx)
// 		if err == nil && len(leagues) > 0 {
// 			// Pre-allocate exact size
// 			apiLeagues = make([]models.SearchResult, len(leagues))
// 			for i, l := range leagues {
// 				apiLeagues[i] = models.SearchResult{
// 					ID:      l.League.ID,
// 					Name:    l.League.Name,
// 					Country: l.Country.Name,
// 				}
// 			}
// 		}
// 		apiErr = err
// 		j.logger.Debug().
// 			Int("count", len(apiLeagues)).
// 			Err(apiErr).
// 			Msg("API-Football fetch completed")
// 	}()

// 	j.logger.Debug().Msg("Waiting for parallel fetches to complete...")
// 	wg.Wait()
// 	j.logger.Debug().Msg("All fetches completed")

// 	if unmappedErr != nil {
// 		return nil, nil, fmt.Errorf("fetch unmapped: %w", unmappedErr)
// 	}
// 	if apiErr != nil {
// 		var rateLimitErr *apifootball.RateLimitError
// 		if errors.As(apiErr, &rateLimitErr) {
// 			return nil, nil, nil // Graceful exit on rate limit
// 		}
// 		return nil, nil, fmt.Errorf("fetch API leagues: %w", apiErr)
// 	}

// 	return unmappedLeagues, apiLeagues, nil
// }

// batchTranslateLeagues translates all leagues in a single batch
func (j *APIFootballLeagueMatchingJobV2) batchTranslateLeagues(ctx context.Context, leagues []generated.League) map[int32]translatedData {
	j.logger.Debug().Int("league_count", len(leagues)).Msg("Starting batch translation for leagues")

	// Collect unique names and countries for batch translation
	uniqueNames := make(map[string]bool)
	uniqueCountries := make(map[string]bool)

	for _, league := range leagues {
		uniqueNames[league.Name] = true
		if league.Country != nil && *league.Country != "" {
			uniqueCountries[*league.Country] = true
		}
	}

	j.logger.Debug().
		Int("unique_names", len(uniqueNames)).
		Int("unique_countries", len(uniqueCountries)).
		Msg("Collected unique values for translation")

	// Batch translate all unique values
	j.logger.Debug().Msg("Starting name translations...")
	nameTranslations := j.batchTranslateNames(ctx, uniqueNames)
	j.logger.Debug().Int("name_translation_count", len(nameTranslations)).Msg("Name translations completed")

	j.logger.Debug().Msg("Starting country translations...")
	countryTranslations := j.batchTranslateCountries(uniqueCountries)
	j.logger.Debug().Int("country_translation_count", len(countryTranslations)).Msg("Country translations completed")

	// Build result map
	j.logger.Debug().Msg("Building translation result map...")
	results := make(map[int32]translatedData, len(leagues))
	for _, league := range leagues {
		td := translatedData{
			Name: nameTranslations[league.Name],
		}
		if league.Country != nil {
			td.Country = countryTranslations[*league.Country]
		}
		results[league.ID] = td
	}

	return results
}

// processMatchesWithSearch finds matches using API-Football search
func (j *APIFootballLeagueMatchingJobV2) processMatchesWithSearch(
	ctx context.Context,
	unmapped []generated.League,
	translations map[int32]translatedData,
) []matchResult {
	// Pre-allocate result slice
	results := make([]matchResult, 0, len(unmapped))
	resultMutex := sync.Mutex{}

	// Process in parallel batches of 5 (smaller to avoid rate limits)
	batchSize := 5
	for i := 0; i < len(unmapped); i += batchSize {
		end := i + batchSize
		if end > len(unmapped) {
			end = len(unmapped)
		}

		batch := unmapped[i:end]
		var wg sync.WaitGroup

		for _, league := range batch {
			wg.Add(1)
			go func(l generated.League) {
				defer wg.Done()

				trans := translations[l.ID]
				match := j.findBestMatchWithSearch(ctx, l, trans)

				if match != nil && match.Confidence >= 0.60 {
					resultMutex.Lock()
					results = append(results, matchResult{
						League:       l,
						Match:        match,
						Translations: trans,
					})
					resultMutex.Unlock()
				}
			}(league)
		}

		wg.Wait()

		// Longer delay between batches to respect rate limits
		if end < len(unmapped) {
			time.Sleep(200 * time.Millisecond)
		}
	}

	return results
}

// // Legacy method kept for compatibility
// func (j *APIFootballLeagueMatchingJobV2) processMatches(
// 	ctx context.Context,
// 	unmapped []generated.League,
// 	apiLeagues []models.SearchResult,
// 	translations map[int32]translatedData,
// ) []matchResult {
// 	// Pre-allocate result slice
// 	results := make([]matchResult, 0, len(unmapped))
// 	resultMutex := sync.Mutex{}

// 	// Process in parallel batches of 10
// 	batchSize := 10
// 	for i := 0; i < len(unmapped); i += batchSize {
// 		end := i + batchSize
// 		if end > len(unmapped) {
// 			end = len(unmapped)
// 		}

// 		batch := unmapped[i:end]
// 		var wg sync.WaitGroup

// 		for _, league := range batch {
// 			wg.Add(1)
// 			go func(l generated.League) {
// 				defer wg.Done()

// 				trans := translations[l.ID]
// 				match := j.findBestMatch(l, trans, apiLeagues)

// 				if match != nil && match.Confidence >= 0.60 {
// 					resultMutex.Lock()
// 					results = append(results, matchResult{
// 						League:       l,
// 						Match:        match,
// 						Translations: trans,
// 					})
// 					resultMutex.Unlock()
// 				}
// 			}(league)
// 		}

// 		wg.Wait()

// 		// Small delay between batches for rate limiting
// 		if end < len(unmapped) {
// 			time.Sleep(100 * time.Millisecond)
// 		}
// 	}

// 	return results
// }

// bulkStoreResults stores all results in a single transaction
func (j *APIFootballLeagueMatchingJobV2) bulkStoreResults(ctx context.Context, results []matchResult) error {
	if len(results) == 0 {
		return nil
	}

	// Pre-allocate all arrays
	count := len(results)
	internalIDs := make([]int32, count)
	externalIDs := make([]int32, count)
	confidences := make([]float32, count)
	methods := make([]string, count)
	translatedNames := make([]*string, count)
	translatedCountries := make([]*string, count)
	originalNames := make([]*string, count)
	originalCountries := make([]*string, count)
	matchFactorsArray := make([][]byte, count)
	needsReviewArray := make([]*bool, count)
	aiUsedArray := make([]*bool, count)
	normAppliedArray := make([]*bool, count)
	matchScores := make([]*float32, count)

	// Fill arrays
	for i, r := range results {
		internalIDs[i] = r.League.ID
		externalIDs[i] = int32(r.Match.ID)
		confidences[i] = float32(r.Match.Confidence)
		methods[i] = r.Match.Method

		// Translations
		translatedNames[i] = &r.Translations.Name
		if r.Translations.Country != "" {
			translatedCountries[i] = &r.Translations.Country
		}

		// Original data
		originalNames[i] = &r.League.Name
		originalCountries[i] = r.League.Country

		// Match factors (simplified)
		factors := map[string]any{
			"method":     r.Match.Method,
			"confidence": r.Match.Confidence,
			"timestamp":  time.Now().UTC(),
		}
		matchFactorsArray[i], _ = json.Marshal(factors)

		// Flags
		needsReview := r.Match.Confidence < 0.85
		needsReviewArray[i] = &needsReview

		aiUsed := true
		aiUsedArray[i] = &aiUsed

		normApplied := true
		normAppliedArray[i] = &normApplied

		score := float32(r.Match.Confidence)
		matchScores[i] = &score
	}

	// Convert float32 to float64 for database params
	confidences64 := make([]float64, len(confidences))
	for i, c := range confidences {
		confidences64[i] = float64(c)
	}

	matchScores64 := make([]float64, len(matchScores))
	for i, s := range matchScores {
		if s != nil {
			matchScores64[i] = float64(*s)
		}
	}

	// Convert nullable arrays to non-nullable
	translatedNamesStr := make([]string, len(translatedNames))
	for i, n := range translatedNames {
		if n != nil {
			translatedNamesStr[i] = *n
		}
	}

	translatedCountriesStr := make([]string, len(translatedCountries))
	for i, c := range translatedCountries {
		if c != nil {
			translatedCountriesStr[i] = *c
		}
	}

	originalNamesStr := make([]string, len(originalNames))
	for i, n := range originalNames {
		if n != nil {
			originalNamesStr[i] = *n
		}
	}

	originalCountriesStr := make([]string, len(originalCountries))
	for i, c := range originalCountries {
		if c != nil {
			originalCountriesStr[i] = *c
		}
	}

	needsReviewBool := make([]bool, len(needsReviewArray))
	for i, r := range needsReviewArray {
		if r != nil {
			needsReviewBool[i] = *r
		}
	}

	aiUsedBool := make([]bool, len(aiUsedArray))
	for i, a := range aiUsedArray {
		if a != nil {
			aiUsedBool[i] = *a
		}
	}

	normAppliedBool := make([]bool, len(normAppliedArray))
	for i, n := range normAppliedArray {
		if n != nil {
			normAppliedBool[i] = *n
		}
	}

	// Single bulk insert
	return j.db.BulkCreateLeagueMappings(ctx, generated.BulkCreateLeagueMappingsParams{
		InternalLeagueIds:     internalIDs,
		FootballApiLeagueIds:  externalIDs,
		Confidences:           confidences64,
		MappingMethods:        methods,
		TranslatedLeagueNames: translatedNamesStr,
		TranslatedCountries:   translatedCountriesStr,
		OriginalLeagueNames:   originalNamesStr,
		OriginalCountries:     originalCountriesStr,
		MatchFactors:          matchFactorsArray,
		NeedsReview:           needsReviewBool,
		AiTranslationUsed:     aiUsedBool,
		NormalizationApplied:  normAppliedBool,
		MatchScores:           matchScores64,
	})
}

// Helper types
type translatedData struct {
	Name    string
	Country string
}

type matchResult struct {
	League       generated.League
	Match        *services.MatchCandidate
	Translations translatedData
}

// findBestMatchWithSearch uses API-Football search for better matching
func (j *APIFootballLeagueMatchingJobV2) findBestMatchWithSearch(
	ctx context.Context,
	league generated.League,
	trans translatedData,
) *services.MatchCandidate {
	j.logger.Debug().
		Int32("league_id", league.ID).
		Str("original_name", league.Name).
		Str("translated_name", trans.Name).
		Str("country", trans.Country).
		Msg("Starting search-based matching")

	// Search with multiple strategies
	searchTerms := []string{
		trans.Name,                       // Primary translated name
		trans.Name + " " + trans.Country, // Name with country
		trans.Country + " " + trans.Name, // Country with name
	}

	// If original has recognizable terms, try those too
	if league.Name != trans.Name {
		searchTerms = append(searchTerms, league.Name)
	}

	var bestMatch *services.MatchCandidate
	maxConfidence := 0.0

	for i, searchTerm := range searchTerms {
		j.logger.Debug().
			Str("search_term", searchTerm).
			Int("attempt", i+1).
			Msg("Searching API-Football")

		// Search API-Football
		searchResults, err := j.apiclient.SearchLeagues(ctx, searchTerm)
		if err != nil {
			j.logger.Error().
				Err(err).
				Str("search_term", searchTerm).
				Msg("Search failed, trying next term")
			continue
		}

		j.logger.Debug().
			Int("results_count", len(searchResults)).
			Str("search_term", searchTerm).
			Msg("Search completed")

		if len(searchResults) == 0 {
			continue
		}

		// Convert to SearchResult format and find best match
		apiLeagues := make([]models.SearchResult, len(searchResults))
		for j, result := range searchResults {
			apiLeagues[j] = models.SearchResult{
				ID:      result.League.ID,
				Name:    result.League.Name,
				Country: result.Country.Name,
			}
		}

		// Use existing matching logic
		translatedLeague := generated.League{
			ID:      league.ID,
			Name:    trans.Name,
			Country: &trans.Country,
		}

		match, err := j.matcher.MatchLeagueWithAPI(ctx, translatedLeague, apiLeagues)
		if err != nil {
			j.logger.Error().
				Err(err).
				Str("search_term", searchTerm).
				Msg("Matching failed")
			continue
		}

		if match != nil && match.Confidence > maxConfidence {
			maxConfidence = match.Confidence
			bestMatch = match
			bestMatch.Method = "search_" + bestMatch.Method
			j.logger.Debug().
				Float64("confidence", match.Confidence).
				Str("matched_name", match.Name).
				Str("search_term", searchTerm).
				Msg("Found better match")
		}

		// Small delay between search attempts
		time.Sleep(50 * time.Millisecond)
	}

	if bestMatch != nil {
		j.logger.Info().
			Int32("league_id", league.ID).
			Str("original_name", league.Name).
			Str("matched_name", bestMatch.Name).
			Float64("confidence", bestMatch.Confidence).
			Str("method", bestMatch.Method).
			Msg("Search-based match found")
	}

	return bestMatch
}

// // Legacy matching logic (kept for compatibility)
// func (j *APIFootballLeagueMatchingJobV2) findBestMatch(
// 	league generated.League,
// 	trans translatedData,
// 	apiLeagues []models.SearchResult,
// ) *services.MatchCandidate {
// 	// Create a temporary translated league for matching
// 	translatedLeague := generated.League{
// 		ID:      league.ID,
// 		Name:    trans.Name,
// 		Country: &trans.Country,
// 	}

// 	// Use the standard MatchLeagueWithAPI method
// 	match, err := j.matcher.MatchLeagueWithAPI(context.Background(), translatedLeague, apiLeagues)
// 	if err != nil {
// 		return nil
// 	}
// 	return match
// }

// Cache-aware translation helpers
func (j *APIFootballLeagueMatchingJobV2) batchTranslateNames(ctx context.Context, names map[string]bool) map[string]string {
	j.logger.Debug().Int("total_names", len(names)).Msg("Starting batch name translation")

	results := make(map[string]string)
	toTranslate := make([]string, 0)

	// Check cache first
	j.logger.Debug().Msg("Checking translation cache...")
	j.cacheMutex.RLock()
	for name := range names {
		if cached, ok := j.translationCache[name]; ok {
			results[name] = cached
		} else {
			toTranslate = append(toTranslate, name)
		}
	}
	j.cacheMutex.RUnlock()

	j.logger.Debug().
		Int("cached", len(results)).
		Int("to_translate", len(toTranslate)).
		Msg("Cache check completed")

	// Batch translate missing ones
	if len(toTranslate) > 0 {
		j.logger.Debug().Msg("Creating AI translation service...")
		aiService := services.NewAITranslationService(os.Getenv("OPENAI_API_KEY"))

		j.logger.Debug().Msg("Calling batch translation API...")
		// Use batch translation for efficiency
		batchResults, err := aiService.BatchTranslateLeagueNames(ctx, toTranslate)
		if err != nil {
			j.logger.Error().
				Err(err).
				Int("count", len(toTranslate)).
				Msg("Batch translation failed")
		} else {
			j.logger.Debug().
				Int("batch_results", len(batchResults)).
				Msg("Batch translation API call completed")
		}

		j.logger.Debug().Msg("Updating cache with results...")
		j.cacheMutex.Lock()
		for _, name := range toTranslate {
			if translations, ok := batchResults[name]; ok && len(translations) > 0 {
				// Use first translation
				results[name] = translations[0]
				j.translationCache[name] = translations[0]
			} else {
				// Fallback to simple translation
				results[name] = name
				j.translationCache[name] = name
			}
		}
		j.cacheMutex.Unlock()
	}

	return results
}

func (j *APIFootballLeagueMatchingJobV2) batchTranslateCountries(countries map[string]bool) map[string]string {
	// Use static mapping for countries (no AI needed)
	results := make(map[string]string)
	for country := range countries {
		// Use static mapping for countries
		countryMappings := map[string]string{
			"Türkiye":   "Turkey",
			"İngiltere": "England",
			"İspanya":   "Spain",
			"Almanya":   "Germany",
			"Fransa":    "France",
			"İtalya":    "Italy",
			"Hollanda":  "Netherlands",
			"Portekiz":  "Portugal",
			"Belçika":   "Belgium",
			"Brezilya":  "Brazil",
			"Arjantin":  "Argentina",
		}

		if translated, ok := countryMappings[country]; ok {
			results[country] = translated
		} else {
			results[country] = country // Keep original if not found
		}
	}
	return results
}

func (j *APIFootballLeagueMatchingJobV2) Timeout() time.Duration {
	return 30 * time.Minute
}
