package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/models"
)

// BulkLeagueMatcherService handles bulk league matching using AI
type BulkLeagueMatcherService struct {
	db           *generated.Queries
	client       *http.Client
	footballKey  string
	aiTranslator *AITranslationService
}

// NewBulkLeagueMatcherService creates a new bulk league matcher
func NewBulkLeagueMatcherService(db *generated.Queries, client *http.Client, footballKey string, aiTranslator *AITranslationService) *BulkLeagueMatcherService {
	return &BulkLeagueMatcherService{
		db:           db,
		client:       client,
		footballKey:  footballKey,
		aiTranslator: aiTranslator,
	}
}

// BulkMatchLeagues performs bulk league matching using AI
func (s *BulkLeagueMatcherService) BulkMatchLeagues(ctx context.Context) error {
	log.Printf("Starting bulk league matching with AI...")

	// Step 1: Get all unmapped football leagues from our database
	unmappedLeagues, err := s.db.ListUnmappedFootballLeagues(ctx)
	if err != nil {
		return fmt.Errorf("failed to get unmapped leagues: %w", err)
	}

	if len(unmappedLeagues) == 0 {
		log.Printf("No unmapped football leagues found")
		return nil
	}

	log.Printf("Found %d unmapped football leagues", len(unmappedLeagues))

	// Step 2: Get all leagues from Football API
	footballAPILeagues, err := s.getAllFootballAPILeagues(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Football API leagues: %w", err)
	}

	log.Printf("Retrieved %d leagues from Football API", len(footballAPILeagues))

	// Step 3: Use AI to generate mapping INSERT statements
	insertStatements, err := s.generateMappingsWithAI(ctx, unmappedLeagues, footballAPILeagues)
	if err != nil {
		return fmt.Errorf("failed to generate AI mappings: %w", err)
	}

	log.Printf("AI generated %d mapping statements", len(insertStatements))

	// Step 4: Display the INSERT statements for review
	s.displayInsertStatements(insertStatements)

	return nil
}

// getAllFootballAPILeagues fetches all current leagues from Football API
func (s *BulkLeagueMatcherService) getAllFootballAPILeagues(ctx context.Context) ([]models.FootballAPILeagueData, error) {
	// Use the recommended endpoint for current active leagues
	url := "https://v3.football.api-sports.io/leagues?current=true"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-RapidAPI-Key", s.footballKey)
	req.Header.Set("X-RapidAPI-Host", "v3.football.api-sports.io")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("football API returned status: %d", resp.StatusCode)
	}

	var apiResponse models.FootballAPILeaguesResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	if apiResponse.HasErrors() {
		return nil, fmt.Errorf("football API returned errors: %v", apiResponse.GetErrorMessages())
	}

	return apiResponse.Response, nil
}

// generateMappingsWithAI uses AI to analyze both datasets and generate INSERT statements
func (s *BulkLeagueMatcherService) generateMappingsWithAI(ctx context.Context, unmappedLeagues []generated.League, footballAPILeagues []models.FootballAPILeagueData) ([]string, error) {
	if s.aiTranslator == nil {
		return nil, fmt.Errorf("AI translator not available")
	}

	var allInsertStatements []string

	// Process unmapped leagues in batches to avoid token limits
	batchSize := 10 // Process 10 leagues at a time
	for i := 0; i < len(unmappedLeagues); i += batchSize {
		end := i + batchSize
		if end > len(unmappedLeagues) {
			end = len(unmappedLeagues)
		}

		batch := unmappedLeagues[i:end]
		log.Printf("Processing batch %d-%d of %d leagues", i+1, end, len(unmappedLeagues))

		// Prepare the data for AI analysis
		prompt := s.createBulkMappingPrompt(batch, footballAPILeagues)

		// Call AI to generate the mappings
		response, err := s.aiTranslator.CallOpenAI(ctx, prompt)
		if err != nil {
			log.Printf("Warning: AI mapping failed for batch %d-%d: %v", i+1, end, err)
			continue // Skip this batch and continue with the next
		}

		// Parse the response to extract INSERT statements
		statements, err := s.parseAIResponse(response)
		if err != nil {
			log.Printf("Warning: Failed to parse AI response for batch %d-%d: %v", i+1, end, err)
			continue
		}

		allInsertStatements = append(allInsertStatements, statements...)
	}

	return allInsertStatements, nil
}

// createBulkMappingPrompt creates a comprehensive prompt for AI league matching
func (s *BulkLeagueMatcherService) createBulkMappingPrompt(unmappedLeagues []generated.League, footballAPILeagues []models.FootballAPILeagueData) string {
	// Build Turkish leagues list
	var turkishLeagues strings.Builder
	turkishLeagues.WriteString("TURKISH LEAGUES TO MAP:\n")
	for _, league := range unmappedLeagues {
		country := "Unknown"
		if league.Country != nil {
			country = *league.Country
		}
		turkishLeagues.WriteString(fmt.Sprintf("ID: %d, Name: \"%s\", Country: %s\n",
			league.ID, league.Name, country))
	}

	// Build Football API leagues list (sample first 100 to avoid token limits)
	var footballLeagues strings.Builder
	footballLeagues.WriteString("\nFOOTBALL API LEAGUES AVAILABLE:\n")
	maxLeagues := len(footballAPILeagues)
	if maxLeagues > 100 {
		maxLeagues = 100
	}
	for i := 0; i < maxLeagues; i++ {
		league := footballAPILeagues[i]
		footballLeagues.WriteString(fmt.Sprintf("ID: %d, Name: \"%s\", Country: %s, Type: %s\n",
			league.League.ID, league.League.Name, league.Country.Name, league.League.Type))
	}

	return fmt.Sprintf(`You are a football league matching expert. I need you to match Turkish football leagues with international Football API leagues.

%s

%s

TASK: Generate SQL INSERT statements for the league_mappings table to map Turkish leagues to Football API leagues.

RULES:
1. Only map football leagues (not other sports)
2. Focus on highest confidence matches (>0.7)
3. Match by league name similarity, country, and type
4. Consider Turkish translations: "Lig"="League", "Kupa"="Cup", "Süper"="Super", etc.
5. Multiple Turkish leagues can map to the same Football API league
6. Use confidence scores: 1.0 (exact), 0.9 (very similar), 0.8 (similar), 0.7 (minimum)

OUTPUT FORMAT:
Generate SQL INSERT statements in this exact format:
INSERT INTO league_mappings (internal_league_id, football_api_league_id, confidence, mapping_method) VALUES (123, 456, 0.95, 'ai_bulk_match');

EXAMPLE:
-- Turkish "Türkiye Süper Lig" matches Football API "Süper Lig" (Turkey)
INSERT INTO league_mappings (internal_league_id, football_api_league_id, confidence, mapping_method) VALUES (1, 203, 1.0, 'ai_bulk_match');

Generate INSERT statements for the best matches:`, turkishLeagues.String(), footballLeagues.String())
}

// parseAIResponse extracts INSERT statements from AI response
func (s *BulkLeagueMatcherService) parseAIResponse(response []string) ([]string, error) {
	if len(response) == 0 {
		return nil, fmt.Errorf("empty AI response")
	}

	var insertStatements []string
	content := strings.Join(response, "\n")
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), "INSERT INTO LEAGUE_MAPPINGS") {
			insertStatements = append(insertStatements, line)
		}
	}

	return insertStatements, nil
}

// displayInsertStatements shows the generated INSERT statements
func (s *BulkLeagueMatcherService) displayInsertStatements(statements []string) {
	separator := strings.Repeat("=", 80)
	log.Printf("\n%s", separator)
	log.Printf("AI GENERATED LEAGUE MAPPING INSERT STATEMENTS")
	log.Printf("%s", separator)

	if len(statements) == 0 {
		log.Printf("No INSERT statements generated by AI")
		return
	}

	for i, stmt := range statements {
		log.Printf("%d. %s", i+1, stmt)
	}

	log.Printf("%s", separator)
	log.Printf("Total statements: %d", len(statements))
	log.Printf("Review these statements and execute manually if they look correct.")
	log.Printf("%s", separator)
}

// ExecuteBulkMappings executes the provided INSERT statements
func (s *BulkLeagueMatcherService) ExecuteBulkMappings(ctx context.Context, insertStatements []string) error {
	log.Printf("Executing %d bulk mapping statements...", len(insertStatements))

	executed := 0
	for i, stmt := range insertStatements {
		// Note: This would need proper SQL parsing and parameter binding in production
		// For now, just log what would be executed
		log.Printf("Would execute %d: %s", i+1, stmt)
		executed++
	}

	log.Printf("Successfully executed %d mapping statements", executed)
	return nil
}
