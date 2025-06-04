package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/betslib/iddaa-core/pkg/database"
	"github.com/betslib/iddaa-core/pkg/models"
)

type LeaguesService struct {
	db          *database.Queries
	client      *http.Client
	apiKey      string
	iddaaClient *IddaaClient
}

func NewLeaguesService(db *database.Queries, client *http.Client, apiKey string, iddaaClient *IddaaClient) *LeaguesService {
	return &LeaguesService{
		db:          db,
		client:      client,
		apiKey:      apiKey,
		iddaaClient: iddaaClient,
	}
}

// SyncLeaguesFromIddaa fetches and syncs leagues from Iddaa competitions endpoint
func (s *LeaguesService) SyncLeaguesFromIddaa(ctx context.Context) error {
	log.Printf("Starting Iddaa leagues sync...")

	// Fetch competitions from Iddaa API
	url := "https://sportsbookv2.iddaa.com/sportsbook/competitions"
	data, err := s.iddaaClient.FetchData(url)
	if err != nil {
		return fmt.Errorf("failed to fetch competitions data: %w", err)
	}

	// Parse the response
	var response models.IddaaCompetitionsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return fmt.Errorf("failed to unmarshal competitions response: %w", err)
	}

	if !response.IsSuccess {
		return fmt.Errorf("API request failed")
	}

	log.Printf("Fetched %d competitions from Iddaa API", len(response.Data))

	// Process each competition
	synced := 0
	for _, competition := range response.Data {
		err := s.syncSingleLeague(ctx, competition)
		if err != nil {
			log.Printf("Failed to sync league %d (%s): %v", competition.ID, competition.Name, err)
			continue
		}
		synced++
	}

	log.Printf("Iddaa leagues sync completed. Synced %d out of %d leagues", synced, len(response.Data))
	return nil
}

// syncSingleLeague processes a single competition/league from Iddaa
func (s *LeaguesService) syncSingleLeague(ctx context.Context, competition models.IddaaCompetition) error {
	// Parse sport ID from string to int
	sportID := 1 // Default to football
	if competition.SportID != "" {
		if id, err := strconv.Atoi(competition.SportID); err == nil {
			sportID = id
		}
	}

	// Create league parameters
	params := database.UpsertLeagueParams{
		ExternalID: fmt.Sprintf("%d", competition.ID),
		Name:       competition.Name,
		Country:    pgtype.Text{String: competition.CountryID, Valid: competition.CountryID != ""},
		SportID:    pgtype.Int4{Int32: int32(sportID), Valid: true},
		IsActive:   pgtype.Bool{Bool: true, Valid: true},
	}

	// Upsert the league
	_, err := s.db.UpsertLeague(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to upsert league: %w", err)
	}

	log.Printf("Synced league: %s (ID: %d, Sport: %d, Country: %s)",
		competition.Name, competition.ID, sportID, competition.CountryID)

	return nil
}

// SyncLeaguesWithFootballAPI fetches leagues from Football API and maps them to our internal leagues
func (s *LeaguesService) SyncLeaguesWithFootballAPI(ctx context.Context) error {
	log.Printf("Starting Football API sync for leagues")

	// Step 1: Get all unmapped internal leagues (FOOTBALL ONLY - sport_id = 1)
	unmappedLeagues, err := s.getUnmappedFootballLeagues(ctx)
	if err != nil {
		return fmt.Errorf("failed to get unmapped football leagues: %w", err)
	}

	if len(unmappedLeagues) == 0 {
		log.Printf("No unmapped football leagues found")
		return nil
	}

	log.Printf("Found %d unmapped football leagues to process", len(unmappedLeagues))

	// Step 2: For each unmapped league, try targeted Football API searches
	mappedCount := 0
	for i, internalLeague := range unmappedLeagues {
		// Rate limiting: wait between requests to avoid hitting API limits
		if i > 0 {
			time.Sleep(200 * time.Millisecond) // 200ms delay between requests
		}

		bestMatch, err := s.findBestLeagueMatchTargeted(ctx, internalLeague)
		if err != nil {
			log.Printf("Error searching for league %s: %v", internalLeague.Name, err)
			continue
		}

		if bestMatch != nil && bestMatch.Similarity >= 0.7 { // 70% confidence threshold
			err := s.createLeagueMapping(ctx, int32(internalLeague.ID), int32(bestMatch.ID), bestMatch.Similarity, bestMatch.Method)
			if err != nil {
				log.Printf("Failed to create mapping for league %s -> %s: %v", internalLeague.Name, bestMatch.Name, err)
				continue
			}
			log.Printf("Mapped league: %s -> %s (confidence: %.2f, method: %s)",
				internalLeague.Name, bestMatch.Name, bestMatch.Similarity, bestMatch.Method)
			mappedCount++
		} else {
			log.Printf("No suitable match found for league: %s", internalLeague.Name)
		}
	}

	log.Printf("Football API sync completed. Mapped %d out of %d leagues", mappedCount, len(unmappedLeagues))
	return nil
}

// getUnmappedFootballLeagues returns only football leagues (sport_id = 1) that don't have a Football API mapping
func (s *LeaguesService) getUnmappedFootballLeagues(ctx context.Context) ([]database.League, error) {
	// Get all football leagues that don't have a mapping in league_mappings table
	rows, err := s.db.ListUnmappedFootballLeagues(ctx)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// findBestLeagueMatchTargeted uses targeted Football API calls for more precise matching
func (s *LeaguesService) findBestLeagueMatchTargeted(ctx context.Context, internalLeague database.League) (*models.SearchResult, error) {
	log.Printf("Searching for Football API match for league: %s (Country: %s)", internalLeague.Name, internalLeague.Country.String)

	// Get English translations for the Turkish league name
	englishNames := s.translateLeagueNameToEnglish(internalLeague.Name)
	allSearchTerms := append([]string{internalLeague.Name}, englishNames...)

	log.Printf("Search terms: %v", allSearchTerms)

	// Strategy 1: Exact name match (try both Turkish and English)
	for _, searchTerm := range allSearchTerms {
		log.Printf("Trying exact name match for: %s", searchTerm)
		if match := s.tryExactNameMatch(ctx, searchTerm); match != nil {
			match.Similarity = 1.0
			match.Method = "exact_name"
			return match, nil
		}
	}

	// Strategy 2: Search by name (try both Turkish and English)
	for _, searchTerm := range allSearchTerms {
		log.Printf("Trying search match for: %s", searchTerm)
		if match := s.trySearchMatch(ctx, searchTerm); match != nil {
			match.Similarity = 0.95
			match.Method = "search_name"
			return match, nil
		}
	}

	// Strategy 3: Country-based search (if we have country info)
	if internalLeague.Country.Valid && internalLeague.Country.String != "" {
		// Try both Turkish and English country names
		englishCountry := s.translateCountryNameToEnglish(internalLeague.Country.String)
		countryNames := []string{internalLeague.Country.String, englishCountry}

		for _, countryName := range countryNames {
			if countryName == "" {
				continue
			}
			log.Printf("Trying country-based search for: %s", countryName)
			if match := s.tryCountryMatch(ctx, internalLeague, countryName); match != nil {
				match.Method = "country_match"
				return match, nil
			}
		}

		// Strategy 4: Search by country name
		for _, countryName := range countryNames {
			if countryName == "" {
				continue
			}
			log.Printf("Trying country search for: %s", countryName)
			if match := s.trySearchMatch(ctx, countryName); match != nil {
				// Check if any of our search terms are similar to the found league
				for _, searchTerm := range allSearchTerms {
					if s.isLeagueNameSimilar(searchTerm, match.Name) {
						match.Similarity = 0.85
						match.Method = "country_search"
						return match, nil
					}
				}
			}
		}
	}

	// Strategy 5: Fallback to similarity matching with a smaller dataset
	log.Printf("Trying fallback similarity matching for: %s", internalLeague.Name)
	return s.tryFallbackMatch(ctx, internalLeague)
}

// tryExactNameMatch attempts exact name matching via Football API
func (s *LeaguesService) tryExactNameMatch(ctx context.Context, leagueName string) *models.SearchResult {
	url := fmt.Sprintf("https://v3.football.api-sports.io/leagues?name=%s", leagueName)
	leagues := s.fetchFromFootballAPI(ctx, url)

	for _, league := range leagues {
		if s.normalizeString(league.Name) == s.normalizeString(leagueName) {
			return &league
		}
	}
	return nil
}

// trySearchMatch attempts search-based matching via Football API
func (s *LeaguesService) trySearchMatch(ctx context.Context, searchTerm string) *models.SearchResult {
	url := fmt.Sprintf("https://v3.football.api-sports.io/leagues?search=%s", searchTerm)
	leagues := s.fetchFromFootballAPI(ctx, url)

	var bestMatch *models.SearchResult
	maxSimilarity := 0.0

	for _, league := range leagues {
		similarity := s.calculateLeagueNameSimilarity(searchTerm, league.Name)
		if similarity > maxSimilarity && similarity >= 0.7 {
			maxSimilarity = similarity
			bestMatch = &models.SearchResult{
				ID:         league.ID,
				Name:       league.Name,
				Country:    league.Country,
				Similarity: similarity,
			}
		}
	}
	return bestMatch
}

// tryCountryMatch attempts country-based matching
func (s *LeaguesService) tryCountryMatch(ctx context.Context, internalLeague database.League, countryName string) *models.SearchResult {
	url := fmt.Sprintf("https://v3.football.api-sports.io/leagues?country=%s", countryName)
	leagues := s.fetchFromFootballAPI(ctx, url)

	var bestMatch *models.SearchResult
	maxSimilarity := 0.0

	for _, league := range leagues {
		similarity := s.calculateLeagueNameSimilarity(internalLeague.Name, league.Name)
		if similarity > maxSimilarity && similarity >= 0.6 { // Lower threshold for country-based
			maxSimilarity = similarity
			bestMatch = &models.SearchResult{
				ID:         league.ID,
				Name:       league.Name,
				Country:    league.Country,
				Similarity: similarity,
			}
		}
	}
	return bestMatch
}

// tryFallbackMatch uses the original similarity-based approach with recent leagues
func (s *LeaguesService) tryFallbackMatch(ctx context.Context, internalLeague database.League) (*models.SearchResult, error) {
	// Fetch recent leagues as fallback
	url := "https://v3.football.api-sports.io/leagues?last=99&current=true"
	leagues := s.fetchFromFootballAPI(ctx, url)

	return s.findBestLeagueMatch(internalLeague, leagues), nil
}

// fetchFromFootballAPI is a helper method for making Football API requests
func (s *LeaguesService) fetchFromFootballAPI(ctx context.Context, url string) []models.SearchResult {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("Failed to create request for %s: %v", url, err)
		return nil
	}

	req.Header.Set("X-RapidAPI-Key", s.apiKey)
	req.Header.Set("X-RapidAPI-Host", "v3.football.api-sports.io")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("Failed to make request to %s: %v", url, err)
		return nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Football API returned status %d for %s", resp.StatusCode, url)
		return nil
	}

	var apiResponse models.FootballAPILeaguesResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		log.Printf("Failed to decode response from %s: %v", url, err)
		return nil
	}

	// Convert to SearchResult format
	results := make([]models.SearchResult, 0, len(apiResponse.Response))
	for _, item := range apiResponse.Response {
		results = append(results, models.SearchResult{
			ID:      item.League.ID,
			Name:    item.League.Name,
			Country: item.Country.Name,
		})
	}

	return results
}

// calculateLeagueNameSimilarity focuses on name similarity for API results
func (s *LeaguesService) calculateLeagueNameSimilarity(name1, name2 string) float64 {
	norm1 := s.normalizeString(name1)
	norm2 := s.normalizeString(name2)

	// Exact match
	if norm1 == norm2 {
		return 1.0
	}

	// Contains match
	if strings.Contains(norm1, norm2) || strings.Contains(norm2, norm1) {
		return 0.9
	}

	// Levenshtein similarity
	return s.levenshteinSimilarity(norm1, norm2)
}

// isLeagueNameSimilar checks if two league names are reasonably similar
func (s *LeaguesService) isLeagueNameSimilar(name1, name2 string) bool {
	return s.calculateLeagueNameSimilarity(name1, name2) >= 0.6
}

// translateLeagueNameToEnglish converts Turkish league names to English equivalents
func (s *LeaguesService) translateLeagueNameToEnglish(turkishName string) []string {
	// Normalize the input for comparison
	normalized := s.normalizeString(turkishName)

	// Common Turkish to English league name mappings
	translations := map[string][]string{
		// Turkish leagues
		"turkiye super lig": {"Super Lig", "Turkish Super League"},
		"turkiye 1 lig":     {"1. Lig", "Turkish First League"},
		"turkiye 2 lig":     {"2. Lig", "Turkish Second League"},
		"turkiye 3 lig":     {"3. Lig", "Turkish Third League"},
		"turkiye kupa":      {"Turkish Cup", "Turkey Cup"},
		"super lig":         {"Super Lig", "Turkish Super League"},

		// Major European leagues
		"ingiltere premier lig":  {"Premier League", "English Premier League"},
		"ingiltere championship": {"Championship", "English Championship"},
		"ingiltere lig 1":        {"League One", "English League One"},
		"ingiltere lig 2":        {"League Two", "English League Two"},
		"ingiltere fa kupa":      {"FA Cup", "English FA Cup"},
		"ingiltere lig kupa":     {"EFL Cup", "English League Cup"},
		"premier lig":            {"Premier League"},

		"ispanya la liga": {"La Liga", "Spanish La Liga", "Primera Division"},
		"ispanya 2 lig":   {"Segunda Division", "Spanish Segunda"},
		"ispanya kupa":    {"Copa del Rey", "Spanish Cup"},
		"la liga":         {"La Liga", "Primera Division"},

		"italya serie a": {"Serie A", "Italian Serie A"},
		"italya serie b": {"Serie B", "Italian Serie B"},
		"italya kupa":    {"Coppa Italia", "Italian Cup"},
		"serie a":        {"Serie A"},
		"serie b":        {"Serie B"},

		"almanya bundesliga":   {"Bundesliga", "German Bundesliga"},
		"almanya 2 bundesliga": {"2. Bundesliga", "German 2. Bundesliga"},
		"almanya 3 lig":        {"3. Liga", "German 3. Liga"},
		"almanya kupa":         {"DFB Pokal", "German Cup"},
		"bundesliga":           {"Bundesliga"},

		"fransa ligue 1": {"Ligue 1", "French Ligue 1"},
		"fransa ligue 2": {"Ligue 2", "French Ligue 2"},
		"fransa kupa":    {"Coupe de France", "French Cup"},
		"ligue 1":        {"Ligue 1"},
		"ligue 2":        {"Ligue 2"},

		"hollanda eredivisie": {"Eredivisie", "Dutch Eredivisie"},
		"hollanda kupa":       {"KNVB Cup", "Dutch Cup"},
		"eredivisie":          {"Eredivisie"},

		"portekiz primeira liga": {"Primeira Liga", "Portuguese Liga"},
		"portekiz kupa":          {"Taca de Portugal", "Portuguese Cup"},
		"primeira liga":          {"Primeira Liga"},

		"belcika pro lig": {"Pro League", "Belgian Pro League"},
		"belcika kupa":    {"Belgian Cup"},

		// International competitions
		"uefa champions league":  {"Champions League", "UEFA Champions League"},
		"uefa europa league":     {"Europa League", "UEFA Europa League"},
		"uefa conference league": {"Conference League", "UEFA Conference League"},
		"champions league":       {"Champions League"},
		"europa league":          {"Europa League"},
		"conference league":      {"Conference League"},
		"nations league":         {"Nations League", "UEFA Nations League"},

		"fifa dunya kupa":     {"World Cup", "FIFA World Cup"},
		"uefa euro":           {"European Championship", "Euro", "UEFA Euro"},
		"dunya kupa":          {"World Cup"},
		"avrupa sampiyonligi": {"European Championship", "Euro"},

		// Other European leagues
		"rusya premier lig":    {"Premier League", "Russian Premier League"},
		"ukrayna premier lig":  {"Premier League", "Ukrainian Premier League"},
		"avusturya bundesliga": {"Bundesliga", "Austrian Bundesliga"},
		"isvicre super lig":    {"Super League", "Swiss Super League"},
		"norves eliteserien":   {"Eliteserien", "Norwegian Eliteserien"},
		"isvec allsvenskan":    {"Allsvenskan", "Swedish Allsvenskan"},
		"danimarka superliga":  {"Superliga", "Danish Superliga"},

		// South American leagues
		"brezilya serie a":  {"Serie A", "Brazilian Serie A", "Brasileirao"},
		"arjantin primera":  {"Primera Division", "Argentine Primera"},
		"kolombiya primera": {"Primera A", "Colombian Primera"},

		// Common keywords that appear in many league names
		"lig":          {"League"},
		"ligi":         {"League"},
		"kupa":         {"Cup"},
		"kupasi":       {"Cup"},
		"sampiyonligi": {"Championship"},
		"turnuvasi":    {"Tournament"},
		"super":        {"Super"},
		"premier":      {"Premier"},
		"birinci":      {"First", "1st"},
		"ikinci":       {"Second", "2nd"},
		"ucuncu":       {"Third", "3rd"},
	}

	// Direct lookup first
	if matches, exists := translations[normalized]; exists {
		return matches
	}

	// Try partial matching for compound names
	var results []string
	for turkishPattern, englishTerms := range translations {
		if strings.Contains(normalized, turkishPattern) {
			results = append(results, englishTerms...)
		}
	}

	// If we found partial matches, also try to construct full translation
	if len(results) > 0 {
		// Try to replace Turkish parts with English equivalents
		englishVersion := normalized
		for turkishTerm, englishTerms := range translations {
			if len(englishTerms) > 0 {
				englishVersion = strings.ReplaceAll(englishVersion, turkishTerm, englishTerms[0])
			}
		}
		if englishVersion != normalized {
			results = append(results, englishVersion)
		}
	}

	// Remove duplicates and return
	return s.removeDuplicates(results)
}

// translateCountryNameToEnglish converts Turkish country names to English
func (s *LeaguesService) translateCountryNameToEnglish(turkishCountry string) string {
	normalized := s.normalizeString(turkishCountry)

	countryTranslations := map[string]string{
		"turkiye":         "Turkey",
		"ingiltere":       "England",
		"ispanya":         "Spain",
		"italya":          "Italy",
		"almanya":         "Germany",
		"fransa":          "France",
		"hollanda":        "Netherlands",
		"portekiz":        "Portugal",
		"belcika":         "Belgium",
		"rusya":           "Russia",
		"ukrayna":         "Ukraine",
		"avusturya":       "Austria",
		"isvicre":         "Switzerland",
		"norves":          "Norway",
		"isvec":           "Sweden",
		"danimarka":       "Denmark",
		"finlandiya":      "Finland",
		"polonya":         "Poland",
		"cek cumhuriyeti": "Czech Republic",
		"macaristan":      "Hungary",
		"romanya":         "Romania",
		"bulgaristan":     "Bulgaria",
		"hirvatistan":     "Croatia",
		"slovenya":        "Slovenia",
		"slovakya":        "Slovakia",
		"brezilya":        "Brazil",
		"arjantin":        "Argentina",
		"kolombiya":       "Colombia",
		"sili":            "Chile",
		"meksika":         "Mexico",
		"abd":             "United States",
		"kanada":          "Canada",
		"japonya":         "Japan",
		"guney kore":      "South Korea",
		"cin":             "China",
		"avustralya":      "Australia",
		"yeni zelanda":    "New Zealand",
		"misir":           "Egypt",
		"fas":             "Morocco",
		"cezayir":         "Algeria",
		"tunus":           "Tunisia",
		"nijerya":         "Nigeria",
		"gana":            "Ghana",
		"kamerun":         "Cameroon",
		"senegal":         "Senegal",
		"guney afrika":    "South Africa",
	}

	if english, exists := countryTranslations[normalized]; exists {
		return english
	}

	return turkishCountry // Return original if no translation found
}

// removeDuplicates removes duplicate strings from a slice
func (s *LeaguesService) removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, str := range slice {
		if str != "" && !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}

// findBestLeagueMatch finds the best matching league from Football API
func (s *LeaguesService) findBestLeagueMatch(internalLeague database.League, footballAPILeagues []models.SearchResult) *models.SearchResult {
	var bestMatch *models.SearchResult
	maxSimilarity := 0.0

	for _, apiLeague := range footballAPILeagues {
		// Try different matching methods
		similarity := s.calculateLeagueSimilarity(internalLeague, apiLeague)
		if similarity > maxSimilarity {
			maxSimilarity = similarity
			bestMatch = &models.SearchResult{
				ID:         apiLeague.ID,
				Name:       apiLeague.Name,
				Country:    apiLeague.Country,
				Similarity: similarity,
				Method:     s.determineBestMethod(internalLeague, apiLeague),
			}
		}
	}

	return bestMatch
}

// calculateLeagueSimilarity calculates similarity between internal and external leagues
func (s *LeaguesService) calculateLeagueSimilarity(internal database.League, external models.SearchResult) float64 {
	// Method 1: Exact name match
	if s.normalizeString(internal.Name) == s.normalizeString(external.Name) {
		return 1.0
	}

	// Method 2: Name contains or is contained
	internalNorm := s.normalizeString(internal.Name)
	externalNorm := s.normalizeString(external.Name)

	if strings.Contains(internalNorm, externalNorm) || strings.Contains(externalNorm, internalNorm) {
		return 0.9
	}

	// Method 3: Fuzzy matching with key words
	internalWords := s.extractKeyWords(internal.Name)
	externalWords := s.extractKeyWords(external.Name)

	commonWords := s.countCommonWords(internalWords, externalWords)
	totalWords := len(internalWords) + len(externalWords) - commonWords

	if totalWords > 0 {
		similarity := float64(commonWords*2) / float64(totalWords)

		// Bonus for country match
		if internal.Country.Valid && internal.Country.String != "" {
			internalCountry := s.normalizeString(internal.Country.String)
			externalCountry := s.normalizeString(external.Country)
			if internalCountry == externalCountry {
				similarity += 0.1
			}
		}

		return similarity
	}

	// Method 4: Levenshtein distance
	return s.levenshteinSimilarity(internalNorm, externalNorm)
}

// determineBestMethod determines which method produced the best match
func (s *LeaguesService) determineBestMethod(internal database.League, external models.SearchResult) string {
	if s.normalizeString(internal.Name) == s.normalizeString(external.Name) {
		return "exact"
	}

	internalNorm := s.normalizeString(internal.Name)
	externalNorm := s.normalizeString(external.Name)

	if strings.Contains(internalNorm, externalNorm) || strings.Contains(externalNorm, internalNorm) {
		return "fuzzy"
	}

	internalWords := s.extractKeyWords(internal.Name)
	externalWords := s.extractKeyWords(external.Name)
	commonWords := s.countCommonWords(internalWords, externalWords)

	if commonWords > 0 {
		return "keyword"
	}

	return "levenshtein"
}

// normalizeString normalizes a string for comparison
func (s *LeaguesService) normalizeString(str string) string {
	// Convert to lowercase and remove special characters
	var result strings.Builder
	for _, r := range strings.ToLower(str) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}
	// Replace multiple spaces with single space and trim
	normalized := strings.Fields(result.String())
	return strings.Join(normalized, " ")
}

// extractKeyWords extracts meaningful words from a string
func (s *LeaguesService) extractKeyWords(str string) []string {
	// Common stop words to ignore
	stopWords := map[string]bool{
		"league": true, "cup": true, "championship": true, "division": true,
		"premier": true, "first": true, "second": true, "third": true,
		"super": true, "national": true, "football": true, "soccer": true,
		"the": true, "of": true, "and": true, "in": true, "at": true,
	}

	words := strings.Fields(s.normalizeString(str))
	var keyWords []string

	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			keyWords = append(keyWords, word)
		}
	}

	return keyWords
}

// countCommonWords counts common words between two slices
func (s *LeaguesService) countCommonWords(words1, words2 []string) int {
	wordMap := make(map[string]bool)
	for _, word := range words1 {
		wordMap[word] = true
	}

	common := 0
	for _, word := range words2 {
		if wordMap[word] {
			common++
		}
	}

	return common
}

// levenshteinSimilarity calculates similarity using Levenshtein distance
func (s *LeaguesService) levenshteinSimilarity(s1, s2 string) float64 {
	if len(s1) == 0 && len(s2) == 0 {
		return 1.0
	}
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	distance := s.levenshteinDistance(s1, s2)
	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func (s *LeaguesService) levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

// createLeagueMapping creates a mapping between internal and external leagues
func (s *LeaguesService) createLeagueMapping(ctx context.Context, internalID, externalID int32, confidence float64, method string) error {
	var confidenceNumeric pgtype.Numeric
	confidenceStr := fmt.Sprintf("%.3f", confidence)
	if err := confidenceNumeric.ScanScientific(confidenceStr); err != nil {
		return fmt.Errorf("failed to convert confidence value %.3f: %w", confidence, err)
	}

	params := database.CreateLeagueMappingParams{
		InternalLeagueID:    internalID,
		FootballApiLeagueID: externalID,
		Confidence:          confidenceNumeric,
		MappingMethod:       method,
	}

	_, err := s.db.CreateLeagueMapping(ctx, params)
	return err
}

// SyncTeamsWithFootballAPI syncs teams for mapped leagues
func (s *LeaguesService) SyncTeamsWithFootballAPI(ctx context.Context) error {
	log.Printf("Starting Football API sync for teams")

	// Step 1: Get all mapped leagues
	mappedLeagues, err := s.getMappedLeagues(ctx)
	if err != nil {
		return fmt.Errorf("failed to get mapped leagues: %w", err)
	}

	if len(mappedLeagues) == 0 {
		log.Printf("No mapped leagues found")
		return nil
	}

	log.Printf("Found %d mapped leagues to process teams for", len(mappedLeagues))

	totalTeamsMapped := 0

	// Step 2: For each mapped league, get teams and sync them
	for _, mapping := range mappedLeagues {
		// Get internal teams for this league
		internalTeams, err := s.getTeamsForLeague(ctx, mapping.InternalLeagueID)
		if err != nil {
			log.Printf("Failed to get teams for league %d: %v", mapping.InternalLeagueID, err)
			continue
		}

		if len(internalTeams) == 0 {
			log.Printf("No teams found for league %d", mapping.InternalLeagueID)
			continue
		}

		// Get Football API teams for this league
		footballAPITeams, err := s.fetchFootballAPITeams(ctx, mapping.FootballApiLeagueID)
		if err != nil {
			log.Printf("Failed to fetch teams from Football API for league %d: %v", mapping.FootballApiLeagueID, err)
			continue
		}

		// Map teams
		teamsMapped := s.mapTeamsForLeague(ctx, internalTeams, footballAPITeams)
		totalTeamsMapped += teamsMapped
		log.Printf("Mapped %d teams for league %d", teamsMapped, mapping.InternalLeagueID)
	}

	log.Printf("Football API team sync completed. Mapped %d teams total", totalTeamsMapped)
	return nil
}

// getMappedLeagues returns all league mappings
func (s *LeaguesService) getMappedLeagues(ctx context.Context) ([]database.LeagueMapping, error) {
	return s.db.ListLeagueMappings(ctx)
}

// getTeamsForLeague returns all teams for a specific league
func (s *LeaguesService) getTeamsForLeague(ctx context.Context, leagueID int32) ([]database.Team, error) {
	return s.db.ListTeamsByLeague(ctx, pgtype.Int4{Int32: leagueID, Valid: true})
}

// fetchFootballAPITeams fetches teams for a specific league from Football API
func (s *LeaguesService) fetchFootballAPITeams(ctx context.Context, leagueID int32) ([]models.SearchResult, error) {
	url := fmt.Sprintf("https://v3.football.api-sports.io/teams?league=%d&season=2024", leagueID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-RapidAPI-Key", s.apiKey)
	req.Header.Set("X-RapidAPI-Host", "v3.football.api-sports.io")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("football API returned status: %d", resp.StatusCode)
	}

	var apiResponse models.FootballAPITeamsResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	// Convert to SearchResult format
	results := make([]models.SearchResult, 0, len(apiResponse.Response))
	for _, item := range apiResponse.Response {
		results = append(results, models.SearchResult{
			ID:      item.Team.ID,
			Name:    item.Team.Name,
			Country: item.Team.Country,
		})
	}

	return results, nil
}

// mapTeamsForLeague maps internal teams to Football API teams for a specific league
func (s *LeaguesService) mapTeamsForLeague(ctx context.Context, internalTeams []database.Team, footballAPITeams []models.SearchResult) int {
	mappedCount := 0

	for _, internalTeam := range internalTeams {
		// Check if team is already mapped
		existing, err := s.db.GetTeamMapping(ctx, internalTeam.ID)
		if err == nil && existing.ID > 0 {
			continue // Already mapped
		}

		// Find best match
		bestMatch := s.findBestTeamMatch(internalTeam, footballAPITeams)
		if bestMatch != nil && bestMatch.Similarity >= 0.7 { // 70% confidence threshold
			err := s.createTeamMapping(ctx, internalTeam.ID, int32(bestMatch.ID), bestMatch.Similarity, bestMatch.Method)
			if err != nil {
				log.Printf("Failed to create team mapping for %s -> %s: %v", internalTeam.Name, bestMatch.Name, err)
				continue
			}
			log.Printf("Mapped team: %s -> %s (confidence: %.2f, method: %s)",
				internalTeam.Name, bestMatch.Name, bestMatch.Similarity, bestMatch.Method)
			mappedCount++
		}
	}

	return mappedCount
}

// findBestTeamMatch finds the best matching team from Football API
func (s *LeaguesService) findBestTeamMatch(internalTeam database.Team, footballAPITeams []models.SearchResult) *models.SearchResult {
	var bestMatch *models.SearchResult
	maxSimilarity := 0.0

	for _, apiTeam := range footballAPITeams {
		similarity := s.calculateTeamSimilarity(internalTeam, apiTeam)
		if similarity > maxSimilarity {
			maxSimilarity = similarity
			bestMatch = &models.SearchResult{
				ID:         apiTeam.ID,
				Name:       apiTeam.Name,
				Country:    apiTeam.Country,
				Similarity: similarity,
				Method:     s.determineTeamBestMethod(internalTeam, apiTeam),
			}
		}
	}

	return bestMatch
}

// calculateTeamSimilarity calculates similarity between internal and external teams
func (s *LeaguesService) calculateTeamSimilarity(internal database.Team, external models.SearchResult) float64 {
	// Method 1: Exact name match
	if s.normalizeString(internal.Name) == s.normalizeString(external.Name) {
		return 1.0
	}

	// Method 2: Name contains or is contained
	internalNorm := s.normalizeString(internal.Name)
	externalNorm := s.normalizeString(external.Name)

	if strings.Contains(internalNorm, externalNorm) || strings.Contains(externalNorm, internalNorm) {
		return 0.9
	}

	// Method 3: Fuzzy matching with key words
	internalWords := s.extractTeamKeyWords(internal.Name)
	externalWords := s.extractTeamKeyWords(external.Name)

	commonWords := s.countCommonWords(internalWords, externalWords)
	totalWords := len(internalWords) + len(externalWords) - commonWords

	if totalWords > 0 {
		similarity := float64(commonWords*2) / float64(totalWords)
		return similarity
	}

	// Method 4: Levenshtein distance
	return s.levenshteinSimilarity(internalNorm, externalNorm)
}

// extractTeamKeyWords extracts meaningful words from a team name
func (s *LeaguesService) extractTeamKeyWords(str string) []string {
	// Common team stop words to ignore
	stopWords := map[string]bool{
		"fc": true, "club": true, "united": true, "city": true, "town": true,
		"football": true, "soccer": true, "sports": true, "athletic": true,
		"the": true, "of": true, "and": true, "in": true, "at": true,
	}

	words := strings.Fields(s.normalizeString(str))
	var keyWords []string

	for _, word := range words {
		if len(word) > 1 && !stopWords[word] {
			keyWords = append(keyWords, word)
		}
	}

	return keyWords
}

// determineTeamBestMethod determines which method produced the best team match
func (s *LeaguesService) determineTeamBestMethod(internal database.Team, external models.SearchResult) string {
	if s.normalizeString(internal.Name) == s.normalizeString(external.Name) {
		return "exact"
	}

	internalNorm := s.normalizeString(internal.Name)
	externalNorm := s.normalizeString(external.Name)

	if strings.Contains(internalNorm, externalNorm) || strings.Contains(externalNorm, internalNorm) {
		return "fuzzy"
	}

	internalWords := s.extractTeamKeyWords(internal.Name)
	externalWords := s.extractTeamKeyWords(external.Name)
	commonWords := s.countCommonWords(internalWords, externalWords)

	if commonWords > 0 {
		return "keyword"
	}

	return "levenshtein"
}

// createTeamMapping creates a mapping between internal and external teams
func (s *LeaguesService) createTeamMapping(ctx context.Context, internalID, externalID int32, confidence float64, method string) error {
	var confidenceNumeric pgtype.Numeric
	confidenceStr := fmt.Sprintf("%.3f", confidence)
	if err := confidenceNumeric.ScanScientific(confidenceStr); err != nil {
		return fmt.Errorf("failed to convert confidence value %.3f: %w", confidence, err)
	}

	params := database.CreateTeamMappingParams{
		InternalTeamID:    internalID,
		FootballApiTeamID: externalID,
		Confidence:        confidenceNumeric,
		MappingMethod:     method,
	}

	_, err := s.db.CreateTeamMapping(ctx, params)
	return err
}
