package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/models"
	"github.com/iddaa-lens/core/pkg/utils"
)

// MatchCandidate represents a potential match with confidence score
type MatchCandidate struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	Country    string  `json:"country"`
	Confidence float64 `json:"confidence"`
	Method     string  `json:"method"`
}

// TeamTranslations holds all translations for a team
type TeamTranslations struct {
	TeamName string
	Country  string
	League   string
	Original generated.Team
}

// LeagueTranslations holds all translations for a league
type LeagueTranslations struct {
	LeagueName string
	Country    string
	Original   generated.League
}

// TeamLeagueMatcher provides comprehensive matching for teams and leagues
type TeamLeagueMatcher struct {
	translator *EnhancedTranslator
	normalizer *utils.TeamNameNormalizer
}

// NewTeamLeagueMatcher creates a new team and league matcher
func NewTeamLeagueMatcher(openaiKey string) *TeamLeagueMatcher {
	return &TeamLeagueMatcher{
		translator: NewEnhancedTranslator(openaiKey),
		normalizer: utils.NewTeamNameNormalizer(),
	}
}

// MatchTeamWithAPI matches a Turkish team with API-Football teams
func (m *TeamLeagueMatcher) MatchTeamWithAPI(ctx context.Context, turkishTeam generated.Team, apiTeams []models.SearchResult) (*MatchCandidate, error) {
	// Step 1: Translate all Turkish data to English
	translations, err := m.translateTeamContext(ctx, turkishTeam)
	if err != nil {
		return nil, fmt.Errorf("translation failed: %w", err)
	}

	// Step 2: Find best matches using multiple strategies
	candidates := m.findTeamCandidates(translations, apiTeams)

	// Step 3: Return best candidate if confidence is high enough
	if len(candidates) > 0 && candidates[0].Confidence >= 0.70 {
		return &candidates[0], nil
	}

	return nil, nil // No good match found
}

// MatchLeagueWithAPI matches a Turkish league with API-Football leagues
func (m *TeamLeagueMatcher) MatchLeagueWithAPI(ctx context.Context, turkishLeague generated.League, apiLeagues []models.SearchResult) (*MatchCandidate, error) {
	// Step 1: Translate Turkish league data to English
	translations, err := m.translateLeagueContext(ctx, turkishLeague)
	if err != nil {
		return nil, fmt.Errorf("translation failed: %w", err)
	}

	// Step 2: Find best matches using multiple strategies
	candidates := m.findLeagueCandidates(translations, apiLeagues)

	// Step 3: Return best candidate if confidence is high enough
	if len(candidates) > 0 && candidates[0].Confidence >= 0.60 {
		return &candidates[0], nil
	}

	return nil, nil // No good match found
}

// translateTeamContext translates all relevant team context from Turkish to English
func (m *TeamLeagueMatcher) translateTeamContext(ctx context.Context, team generated.Team) (*TeamTranslations, error) {
	// Get country string
	countryStr := ""
	if team.Country != nil {
		countryStr = *team.Country
	}

	// Translate team name (most important)
	teamName, err := m.translator.TranslateTeamName(ctx, team.Name, countryStr)
	if err != nil {
		return nil, fmt.Errorf("failed to translate team name: %w", err)
	}

	// Translate country
	country := ""
	if team.Country != nil {
		country = m.translator.TranslateCountryName(*team.Country)
	}

	// League information would ideally come from team's event participation
	// For now, use empty league - this would need database access to implement properly
	league := ""

	return &TeamTranslations{
		TeamName: teamName,
		Country:  country,
		League:   league,
		Original: team,
	}, nil
}

// translateLeagueContext translates all relevant league context from Turkish to English
func (m *TeamLeagueMatcher) translateLeagueContext(ctx context.Context, league generated.League) (*LeagueTranslations, error) {
	// Get country string
	countryStr := ""
	if league.Country != nil {
		countryStr = *league.Country
	}

	// Translate league name
	leagueName, err := m.translator.TranslateLeagueName(ctx, league.Name, countryStr)
	if err != nil {
		return nil, fmt.Errorf("failed to translate league name: %w", err)
	}

	// Translate country
	country := ""
	if league.Country != nil {
		country = m.translator.TranslateCountryName(*league.Country)
	}

	return &LeagueTranslations{
		LeagueName: leagueName,
		Country:    country,
		Original:   league,
	}, nil
}

// findTeamCandidates finds potential team matches using multiple strategies
func (m *TeamLeagueMatcher) findTeamCandidates(translations *TeamTranslations, apiTeams []models.SearchResult) []MatchCandidate {
	var candidates []MatchCandidate
	seen := make(map[int]bool) // Prevent duplicates

	// Get normalized variations of the translated team name
	teamVariations := m.normalizer.GetNormalizedVariations(translations.TeamName)

	for _, apiTeam := range apiTeams {
		if seen[apiTeam.ID] {
			continue
		}

		confidence := m.calculateTeamMatchConfidence(translations, apiTeam, teamVariations)
		if confidence >= 0.60 { // Minimum threshold
			candidates = append(candidates, MatchCandidate{
				ID:         apiTeam.ID,
				Name:       apiTeam.Name,
				Country:    apiTeam.Country,
				Confidence: confidence,
				Method:     m.determineTeamMatchMethod(translations, apiTeam),
			})
			seen[apiTeam.ID] = true
		}
	}

	// Sort by confidence (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Confidence > candidates[j].Confidence
	})

	return candidates
}

// findLeagueCandidates finds potential league matches using multiple strategies
func (m *TeamLeagueMatcher) findLeagueCandidates(translations *LeagueTranslations, apiLeagues []models.SearchResult) []MatchCandidate {
	var candidates []MatchCandidate
	seen := make(map[int]bool) // Prevent duplicates

	// Get normalized variations of the translated league name
	leagueVariations := m.normalizer.GetNormalizedVariations(translations.LeagueName)

	for _, apiLeague := range apiLeagues {
		if seen[apiLeague.ID] {
			continue
		}

		confidence := m.calculateLeagueMatchConfidence(translations, apiLeague, leagueVariations)
		if confidence >= 0.50 { // Lowered minimum threshold for more matches
			candidates = append(candidates, MatchCandidate{
				ID:         apiLeague.ID,
				Name:       apiLeague.Name,
				Country:    apiLeague.Country,
				Confidence: confidence,
				Method:     m.determineLeagueMatchMethod(translations, apiLeague),
			})
			seen[apiLeague.ID] = true
		}
	}

	// Sort by confidence (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Confidence > candidates[j].Confidence
	})

	return candidates
}

// calculateTeamMatchConfidence calculates confidence score for team matching
func (m *TeamLeagueMatcher) calculateTeamMatchConfidence(translations *TeamTranslations, apiTeam models.SearchResult, teamVariations []string) float64 {
	var maxConfidence float64

	// Strategy 1: Direct name comparison with all variations
	for _, variation := range teamVariations {
		confidence := m.normalizer.CompareNormalized(variation, apiTeam.Name)
		if confidence > maxConfidence {
			maxConfidence = confidence
		}
	}

	// Strategy 2: Keyword-based matching
	teamKeywords := m.normalizer.ExtractKeywords(translations.TeamName)
	apiKeywords := m.normalizer.ExtractKeywords(apiTeam.Name)
	keywordSimilarity := m.calculateKeywordSimilarity(teamKeywords, apiKeywords)

	// Weight keyword similarity slightly lower
	weightedKeywordSimilarity := keywordSimilarity * 0.9
	if weightedKeywordSimilarity > maxConfidence {
		maxConfidence = weightedKeywordSimilarity
	}

	// Strategy 3: Country bonus
	if translations.Country != "" && apiTeam.Country != "" {
		countryMatch := m.normalizer.CompareNormalized(translations.Country, apiTeam.Country)
		if countryMatch > 0.8 {
			maxConfidence += 0.1 // Boost for country match
		}
	}

	// Strategy 4: Penalize if countries clearly don't match
	if translations.Country != "" && apiTeam.Country != "" {
		countryMatch := m.normalizer.CompareNormalized(translations.Country, apiTeam.Country)
		if countryMatch < 0.3 {
			maxConfidence *= 0.8 // Penalty for country mismatch
		}
	}

	// Ensure confidence doesn't exceed 1.0
	return math.Min(maxConfidence, 1.0)
}

// calculateLeagueMatchConfidence calculates confidence score for league matching
func (m *TeamLeagueMatcher) calculateLeagueMatchConfidence(translations *LeagueTranslations, apiLeague models.SearchResult, leagueVariations []string) float64 {
	var maxConfidence float64

	// Strategy 1: Direct name comparison with all variations
	for _, variation := range leagueVariations {
		confidence := m.normalizer.CompareNormalized(variation, apiLeague.Name)
		if confidence > maxConfidence {
			maxConfidence = confidence
		}
	}

	// Strategy 2: Keyword-based matching
	leagueKeywords := m.normalizer.ExtractKeywords(translations.LeagueName)
	apiKeywords := m.normalizer.ExtractKeywords(apiLeague.Name)
	keywordSimilarity := m.calculateKeywordSimilarity(leagueKeywords, apiKeywords)

	// Weight keyword similarity slightly lower
	weightedKeywordSimilarity := keywordSimilarity * 0.9
	if weightedKeywordSimilarity > maxConfidence {
		maxConfidence = weightedKeywordSimilarity
	}

	// Strategy 3: Country bonus (very important for leagues)
	if translations.Country != "" && apiLeague.Country != "" {
		countryMatch := m.normalizer.CompareNormalized(translations.Country, apiLeague.Country)
		if countryMatch > 0.8 {
			maxConfidence += 0.15 // Higher boost for leagues
		}
	}

	// Strategy 4: Strong penalty if countries clearly don't match
	if translations.Country != "" && apiLeague.Country != "" {
		countryMatch := m.normalizer.CompareNormalized(translations.Country, apiLeague.Country)
		if countryMatch < 0.3 {
			maxConfidence *= 0.7 // Reduced penalty to allow more matches
		}
	}

	// Strategy 5: Partial name matching for common league patterns
	if maxConfidence < 0.6 {
		partialMatch := m.calculatePartialLeagueMatch(translations.LeagueName, apiLeague.Name)
		if partialMatch > maxConfidence {
			maxConfidence = partialMatch
		}
	}

	// Ensure confidence doesn't exceed 1.0
	return math.Min(maxConfidence, 1.0)
}

// calculatePartialLeagueMatch looks for partial matches in league names
func (m *TeamLeagueMatcher) calculatePartialLeagueMatch(translatedName, apiName string) float64 {
	// Common league terms that should match
	leagueTerms := map[string][]string{
		"super league":   {"super lig", "süper lig", "superlig"},
		"first league":   {"1. lig", "birinci lig", "first division"},
		"second league":  {"2. lig", "ikinci lig", "second division"},
		"third league":   {"3. lig", "üçüncü lig", "third division"},
		"premier league": {"premier lig", "premier league"},
		"championship":   {"şampiyonluk", "championship"},
		"cup":            {"kupa", "kupası", "cup"},
		"playoffs":       {"play-off", "playoff"},
	}

	translatedLower := strings.ToLower(translatedName)
	apiLower := strings.ToLower(apiName)

	// Check for common terms
	for englishTerm, variations := range leagueTerms {
		// Check if API name contains the English term
		if strings.Contains(apiLower, englishTerm) {
			// Check if translated name contains any variation
			for _, variation := range variations {
				if strings.Contains(translatedLower, variation) {
					return 0.65 // Good partial match
				}
			}
		}
	}

	// Check for number-based leagues (1st, 2nd, etc.)
	if strings.Contains(translatedLower, "1.") && (strings.Contains(apiLower, "first") || strings.Contains(apiLower, "1st")) {
		return 0.60
	}
	if strings.Contains(translatedLower, "2.") && (strings.Contains(apiLower, "second") || strings.Contains(apiLower, "2nd")) {
		return 0.60
	}
	if strings.Contains(translatedLower, "3.") && (strings.Contains(apiLower, "third") || strings.Contains(apiLower, "3rd")) {
		return 0.60
	}

	return 0.0
}

// calculateKeywordSimilarity calculates similarity based on common keywords
func (m *TeamLeagueMatcher) calculateKeywordSimilarity(keywords1, keywords2 []string) float64 {
	if len(keywords1) == 0 || len(keywords2) == 0 {
		return 0.0
	}

	// Create a map for faster lookup
	keywordMap := make(map[string]bool)
	for _, kw := range keywords1 {
		keywordMap[strings.ToLower(kw)] = true
	}

	// Count common keywords
	common := 0
	for _, kw := range keywords2 {
		if keywordMap[strings.ToLower(kw)] {
			common++
		}
	}

	// Calculate Jaccard similarity
	union := len(keywords1) + len(keywords2) - common
	if union == 0 {
		return 0.0
	}

	return float64(common) / float64(union)
}

// determineTeamMatchMethod determines which method produced the best team match
func (m *TeamLeagueMatcher) determineTeamMatchMethod(translations *TeamTranslations, apiTeam models.SearchResult) string {
	// Exact match
	if m.normalizer.CompareNormalized(translations.TeamName, apiTeam.Name) >= 0.95 {
		return "exact_name"
	}

	// Normalized match
	normalizedTranslated := m.normalizer.Normalize(translations.TeamName)
	normalizedAPI := m.normalizer.Normalize(apiTeam.Name)
	if m.normalizer.CompareNormalized(normalizedTranslated, normalizedAPI) >= 0.90 {
		return "normalized_name"
	}

	// Keyword match
	teamKeywords := m.normalizer.ExtractKeywords(translations.TeamName)
	apiKeywords := m.normalizer.ExtractKeywords(apiTeam.Name)
	if m.calculateKeywordSimilarity(teamKeywords, apiKeywords) >= 0.70 {
		return "keyword_match"
	}

	// Country-assisted match
	if translations.Country != "" && apiTeam.Country != "" {
		countryMatch := m.normalizer.CompareNormalized(translations.Country, apiTeam.Country)
		if countryMatch > 0.8 {
			return "country_assisted"
		}
	}

	return "fuzzy_match"
}

// determineLeagueMatchMethod determines which method produced the best league match
func (m *TeamLeagueMatcher) determineLeagueMatchMethod(translations *LeagueTranslations, apiLeague models.SearchResult) string {
	// Exact match
	if m.normalizer.CompareNormalized(translations.LeagueName, apiLeague.Name) >= 0.95 {
		return "exact_name"
	}

	// Normalized match
	normalizedTranslated := m.normalizer.Normalize(translations.LeagueName)
	normalizedAPI := m.normalizer.Normalize(apiLeague.Name)
	if m.normalizer.CompareNormalized(normalizedTranslated, normalizedAPI) >= 0.90 {
		return "normalized_name"
	}

	// Keyword match
	leagueKeywords := m.normalizer.ExtractKeywords(translations.LeagueName)
	apiKeywords := m.normalizer.ExtractKeywords(apiLeague.Name)
	if m.calculateKeywordSimilarity(leagueKeywords, apiKeywords) >= 0.70 {
		return "keyword_match"
	}

	// Country-assisted match
	if translations.Country != "" && apiLeague.Country != "" {
		countryMatch := m.normalizer.CompareNormalized(translations.Country, apiLeague.Country)
		if countryMatch > 0.8 {
			return "country_assisted"
		}
	}

	return "fuzzy_match"
}

// GetTeamNameWithAI gets the most common English name for a team using AI
func (m *TeamLeagueMatcher) GetTeamNameWithAI(ctx context.Context, teamName, country string) (string, error) {
	if m.translator.aiTranslator == nil {
		return "", fmt.Errorf("AI translator not available")
	}

	// Use the proper translation method that handles different countries
	translations, err := m.translator.aiTranslator.TranslateTeamName(ctx, teamName, country)
	if err != nil {
		return "", err
	}

	if len(translations) > 0 {
		return strings.TrimSpace(translations[0]), nil
	}

	return "", fmt.Errorf("no translation returned")
}

// GetLeagueNameWithAI gets the most common English name for a league using AI
func (m *TeamLeagueMatcher) GetLeagueNameWithAI(ctx context.Context, turkishName, country string) (string, error) {
	if m.translator.aiTranslator == nil {
		return "", fmt.Errorf("AI translator not available")
	}

	// Use the proper translation method that handles different countries
	translations, err := m.translator.aiTranslator.TranslateLeagueName(ctx, turkishName, country)
	if err != nil {
		return "", err
	}

	if len(translations) > 0 {
		return strings.TrimSpace(translations[0]), nil
	}

	return "", fmt.Errorf("no translation returned")
}

// ValidateMatch validates if a match makes sense based on additional context
func (m *TeamLeagueMatcher) ValidateMatch(translations interface{}, apiResult models.SearchResult, confidence float64) bool {
	// Basic confidence threshold
	if confidence < 0.60 {
		return false
	}

	// Additional validation logic can be added here
	// For example, checking if team names are suspiciously different despite high confidence

	return true
}

// UsesAI returns whether this matcher uses AI translation
func (m *TeamLeagueMatcher) UsesAI() bool {
	return m.translator.aiTranslator != nil
}

// GetAITranslator returns the AI translator instance if available
func (m *TeamLeagueMatcher) GetAITranslator() *AITranslationService {
	if m.translator != nil {
		return m.translator.aiTranslator
	}
	return nil
}
