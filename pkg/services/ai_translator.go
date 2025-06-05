package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// AITranslationService handles AI-powered translation of Turkish league names to English
type AITranslationService struct {
	client  *http.Client
	apiKey  string
	baseURL string
	cache   map[string][]string // Simple in-memory cache
}

// NewAITranslationService creates a new AI translation service
func NewAITranslationService(apiKey string) *AITranslationService {
	return &AITranslationService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1/chat/completions",
		cache:   make(map[string][]string),
	}
}

// OpenAIRequest represents the request structure for OpenAI API
type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents the response from OpenAI API
type OpenAIResponse struct {
	Choices []Choice  `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

// Choice represents a response choice
type Choice struct {
	Message Message `json:"message"`
}

// APIError represents an API error
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// TranslateLeagueName translates a Turkish league name to multiple English variations
func (s *AITranslationService) TranslateLeagueName(ctx context.Context, turkishName, country string) ([]string, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s|%s", turkishName, country)
	if cached, exists := s.cache[cacheKey]; exists {
		log.Printf("Using cached translation for: %s", turkishName)
		return cached, nil
	}

	// Create the translation prompt
	prompt := s.createTranslationPrompt(turkishName, country)

	// Call OpenAI API
	translations, err := s.callOpenAI(ctx, prompt)
	if err != nil {
		log.Printf("AI translation failed for %s: %v", turkishName, err)
		// Fallback to static translation
		return s.fallbackTranslation(turkishName), nil
	}

	// Cache the result
	s.cache[cacheKey] = translations
	log.Printf("AI translated '%s' to: %v", turkishName, translations)

	return translations, nil
}

// createTranslationPrompt creates a focused prompt for league name translation
func (s *AITranslationService) createTranslationPrompt(turkishName, country string) string {
	return fmt.Sprintf(`Translate this Turkish football league name to English for international football API matching.

Turkish League: "%s"
Country: %s

Provide 3-5 English variations commonly used in international football databases and APIs. Focus on:
1. Official English name used by FIFA/UEFA
2. Common name used in international media
3. Short/abbreviated version
4. Generic descriptive name
5. Alternative spelling if applicable

Return ONLY the English names, one per line, without numbers or explanations.

Examples:
Turkish: "Türkiye Süper Lig" → 
Super Lig
Turkish Super League
Turkey Super League

Turkish: "İspanya La Liga" →
La Liga
Spanish La Liga
Primera Division

Now translate: "%s"`, turkishName, country, turkishName)
}

// CallOpenAI makes the actual API call to OpenAI (public method)
func (s *AITranslationService) CallOpenAI(ctx context.Context, prompt string) ([]string, error) {
	return s.callOpenAI(ctx, prompt)
}

// callOpenAI makes the actual API call to OpenAI
func (s *AITranslationService) callOpenAI(ctx context.Context, prompt string) ([]string, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not provided")
	}

	request := OpenAIRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   150,
		Temperature: 0.1, // Low temperature for consistent translations
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned")
	}

	// Parse the response into individual translations
	content := strings.TrimSpace(response.Choices[0].Message.Content)
	lines := strings.Split(content, "\n")

	var translations []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Turkish:") && !strings.HasPrefix(line, "Examples:") {
			// Remove any numbering or bullet points
			line = strings.TrimPrefix(line, "- ")
			for i := 1; i <= 9; i++ {
				line = strings.TrimPrefix(line, fmt.Sprintf("%d. ", i))
			}
			line = strings.TrimSpace(line)
			if line != "" {
				translations = append(translations, line)
			}
		}
	}

	if len(translations) == 0 {
		return nil, fmt.Errorf("no valid translations found in response")
	}

	return translations, nil
}

// fallbackTranslation provides static fallback when AI translation fails
func (s *AITranslationService) fallbackTranslation(turkishName string) []string {
	// Simplified version of the existing static translation logic
	normalized := strings.ToLower(turkishName)
	normalized = strings.ReplaceAll(normalized, "ç", "c")
	normalized = strings.ReplaceAll(normalized, "ğ", "g")
	normalized = strings.ReplaceAll(normalized, "ı", "i")
	normalized = strings.ReplaceAll(normalized, "ö", "o")
	normalized = strings.ReplaceAll(normalized, "ş", "s")
	normalized = strings.ReplaceAll(normalized, "ü", "u")

	// Basic keyword replacements
	replacements := map[string]string{
		"türkiye":      "Turkey",
		"super lig":    "Super League",
		"lig":          "League",
		"ligi":         "League",
		"kupa":         "Cup",
		"kupası":       "Cup",
		"şampiyonluğu": "Championship",
		"premier":      "Premier",
		"birinci":      "First",
		"ikinci":       "Second",
		"üçüncü":       "Third",
	}

	result := normalized
	for turkish, english := range replacements {
		result = strings.ReplaceAll(result, turkish, english)
	}

	// Clean up and capitalize
	words := strings.Fields(result)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	fallback := strings.Join(words, " ")

	// Return the original and fallback
	return []string{turkishName, fallback}
}

// ClearCache clears the translation cache
func (s *AITranslationService) ClearCache() {
	s.cache = make(map[string][]string)
}

// GetCacheSize returns the number of cached translations
func (s *AITranslationService) GetCacheSize() int {
	return len(s.cache)
}
