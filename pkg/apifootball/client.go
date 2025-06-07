package apifootball

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/iddaa-lens/core/pkg/models"
)

// Client provides access to the API-Football service
type Client struct {
	httpClient  *http.Client
	apiKey      string
	baseURL     string
	rateLimiter *RateLimiter
}

// Config holds configuration for the API-Football client
type Config struct {
	APIKey         string
	Timeout        time.Duration
	RequestsPerMin int
	BaseURL        string
}

// DefaultConfig returns a default configuration
func DefaultConfig(apiKey string) *Config {
	return &Config{
		APIKey:         apiKey,
		Timeout:        30 * time.Second,
		RequestsPerMin: 60, // API-Football free tier limit
		BaseURL:        "https://v3.football.api-sports.io",
	}
}

// NewClient creates a new API-Football client
func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig("")
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		apiKey:      config.APIKey,
		baseURL:     config.BaseURL,
		rateLimiter: NewRateLimiter(config.RequestsPerMin),
	}
}

// RateLimiter implements simple rate limiting
type RateLimiter struct {
	tokens    chan struct{}
	interval  time.Duration
	lastReset time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		tokens:    make(chan struct{}, requestsPerMinute),
		interval:  time.Minute / time.Duration(requestsPerMinute),
		lastReset: time.Now(),
	}

	// Fill the token bucket initially
	for i := 0; i < requestsPerMinute; i++ {
		rl.tokens <- struct{}{}
	}

	return rl
}

// Wait blocks until a request can be made
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		// Add token back after interval
		go func() {
			time.Sleep(rl.interval)
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Token bucket is full
			}
		}()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// APIResponse represents the standard API-Football response structure
type APIResponse struct {
	Get        string                   `json:"get"`
	Parameters map[string]interface{}   `json:"parameters"`
	Errors     interface{}              `json:"errors"`
	Results    int                      `json:"results"`
	Paging     models.FootballAPIPaging `json:"paging"`
	Response   json.RawMessage          `json:"response"`
}

// HasErrors checks if the API response contains errors
func (r *APIResponse) HasErrors() bool {
	if r.Errors == nil {
		return false
	}

	switch errors := r.Errors.(type) {
	case []interface{}:
		return len(errors) > 0
	case map[string]interface{}:
		return len(errors) > 0
	case string:
		return errors != ""
	default:
		return false
	}
}

// GetErrorMessages extracts error messages from the response
func (r *APIResponse) GetErrorMessages() []string {
	if !r.HasErrors() {
		return nil
	}

	var messages []string
	switch errors := r.Errors.(type) {
	case []interface{}:
		for _, err := range errors {
			if errStr, ok := err.(string); ok {
				messages = append(messages, errStr)
			}
		}
	case map[string]interface{}:
		for key, value := range errors {
			if valueStr, ok := value.(string); ok {
				messages = append(messages, fmt.Sprintf("%s: %s", key, valueStr))
			}
		}
	case string:
		messages = append(messages, errors)
	}

	return messages
}

// makeRequest makes a request to the API-Football service
func (c *Client) makeRequest(ctx context.Context, endpoint string, params map[string]string) (*APIResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Apply rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Build URL with parameters
	u, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	query := u.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	u.RawQuery = query.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("X-RapidAPI-Key", c.apiKey)
	req.Header.Set("X-RapidAPI-Host", "v3.football.api-sports.io")
	req.Header.Set("Accept", "application/json")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response
	var apiResponse APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API errors
	if apiResponse.HasErrors() {
		errorMessages := apiResponse.GetErrorMessages()
		return nil, fmt.Errorf("API returned errors: %v", errorMessages)
	}

	return &apiResponse, nil
}

// IsAvailable checks if the API key is configured
func (c *Client) IsAvailable() bool {
	return c.apiKey != ""
}

// GetAPIKey returns the configured API key (for testing)
func (c *Client) GetAPIKey() string {
	return c.apiKey
}

// SetAPIKey sets the API key
func (c *Client) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// Health checks if the API is accessible
func (c *Client) Health(ctx context.Context) error {
	if !c.IsAvailable() {
		return fmt.Errorf("API key not configured")
	}

	// Make a simple request to test connectivity
	_, err := c.makeRequest(ctx, "/leagues", map[string]string{
		"current": "true",
		"last":    "1", // Limit to 1 result for health check
	})

	return err
}

// Common parameter builders for convenience
func ParamID(id int) map[string]string {
	return map[string]string{"id": strconv.Itoa(id)}
}

func ParamName(name string) map[string]string {
	return map[string]string{"name": name}
}

func ParamCountry(country string) map[string]string {
	return map[string]string{"country": country}
}

func ParamSearch(search string) map[string]string {
	return map[string]string{"search": search}
}

func ParamCurrent(current bool) map[string]string {
	return map[string]string{"current": strconv.FormatBool(current)}
}

func ParamSeason(season int) map[string]string {
	return map[string]string{"season": strconv.Itoa(season)}
}

func ParamTeam(teamID int) map[string]string {
	return map[string]string{"team": strconv.Itoa(teamID)}
}

func ParamType(leagueType string) map[string]string {
	return map[string]string{"type": leagueType}
}

func ParamLast(count int) map[string]string {
	return map[string]string{"last": strconv.Itoa(count)}
}

func ParamCode(code string) map[string]string {
	return map[string]string{"code": code}
}

// MergeParams merges multiple parameter maps
func MergeParams(paramMaps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, params := range paramMaps {
		for key, value := range params {
			result[key] = value
		}
	}
	return result
}
