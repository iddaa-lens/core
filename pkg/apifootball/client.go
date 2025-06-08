package apifootball

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/iddaa-lens/core/pkg/models"
)

// RateLimitError represents a rate limit error from the API
type RateLimitError struct {
	StatusCode int
	RetryAfter string
	Message    string
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter != "" {
		return fmt.Sprintf("rate limit exceeded (status %d), retry after: %s", e.StatusCode, e.RetryAfter)
	}
	return fmt.Sprintf("rate limit exceeded (status %d): %s", e.StatusCode, e.Message)
}

func (e *RateLimitError) IsRateLimit() bool {
	return true
}

// APIError represents a general API error
type APIError struct {
	StatusCode int
	Message    string
	Errors     []string
}

func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("API error (status %d): %s - %s", e.StatusCode, e.Message, strings.Join(e.Errors, ", "))
	}
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
}

// CacheEntry represents a cached API response
type CacheEntry struct {
	Data      *APIResponse
	ExpiresAt time.Time
}

// SimpleCache provides basic in-memory caching with TTL
type SimpleCache struct {
	entries map[string]*CacheEntry
	mutex   sync.RWMutex
	ttl     time.Duration
}

// NewSimpleCache creates a new cache with the specified TTL
func NewSimpleCache(ttl time.Duration) *SimpleCache {
	cache := &SimpleCache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Get retrieves a cached entry if it exists and hasn't expired
func (c *SimpleCache) Get(key string) (*APIResponse, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		delete(c.entries, key)
		return nil, false
	}

	return entry.Data, true
}

// Set stores an entry in the cache
func (c *SimpleCache) Set(key string, data *APIResponse) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries[key] = &CacheEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// cleanupExpired removes expired entries from the cache
func (c *SimpleCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		for key, entry := range c.entries {
			if now.After(entry.ExpiresAt) {
				delete(c.entries, key)
			}
		}
		c.mutex.Unlock()
	}
}

// Client provides access to the API-Football service
type Client struct {
	httpClient  *http.Client
	apiKey      string
	baseURL     string
	rateLimiter *RateLimiter
	cache       *SimpleCache
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
		cache:       NewSimpleCache(15 * time.Minute), // Cache responses for 15 minutes
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

// generateCacheKey creates a cache key from endpoint and parameters
func (c *Client) generateCacheKey(endpoint string, params map[string]string) string {
	// Sort parameters for consistent key generation
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	parts = append(parts, endpoint)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
	}

	keyStr := strings.Join(parts, "&")
	return fmt.Sprintf("%x", md5.Sum([]byte(keyStr)))
}

// makeRequest makes a request to the API-Football service
func (c *Client) makeRequest(ctx context.Context, endpoint string, params map[string]string) (*APIResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Check cache first
	cacheKey := c.generateCacheKey(endpoint, params)
	if cachedResponse, found := c.cache.Get(cacheKey); found {
		return cachedResponse, nil
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
		// Handle rate limit errors specifically
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			return nil, &RateLimitError{
				StatusCode: resp.StatusCode,
				RetryAfter: retryAfter,
				Message:    "Daily request limit exceeded",
			}
		}

		// Handle other API errors
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("API returned status %d", resp.StatusCode),
		}
	}

	// Parse response
	var apiResponse APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API errors
	if apiResponse.HasErrors() {
		errorMessages := apiResponse.GetErrorMessages()

		// Check if any error message indicates rate limiting
		for _, msg := range errorMessages {
			if strings.Contains(strings.ToLower(msg), "request limit") ||
				strings.Contains(strings.ToLower(msg), "rate limit") ||
				strings.Contains(strings.ToLower(msg), "quota exceeded") {
				return nil, &RateLimitError{
					StatusCode: 200, // API returned 200 but with rate limit error in body
					Message:    msg,
				}
			}
		}

		return nil, &APIError{
			StatusCode: 200,
			Message:    "API returned errors in response body",
			Errors:     errorMessages,
		}
	}

	// Cache successful responses
	c.cache.Set(cacheKey, &apiResponse)

	return &apiResponse, nil
}

// makeRequestWithRetry makes a request with retry logic for rate limit errors
func (c *Client) makeRequestWithRetry(ctx context.Context, endpoint string, params map[string]string, maxRetries int) (*APIResponse, error) {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := c.makeRequest(ctx, endpoint, params)

		if err == nil {
			return resp, nil
		}

		// Check if it's a rate limit error
		if rateLimitErr, ok := err.(*RateLimitError); ok {
			if attempt == maxRetries {
				return nil, err // Final attempt failed
			}

			// Calculate backoff delay (exponential backoff with jitter)
			baseDelay := time.Duration(1<<uint(attempt)) * time.Second // 1s, 2s, 4s, 8s...
			maxDelay := 60 * time.Second
			if baseDelay > maxDelay {
				baseDelay = maxDelay
			}

			// Add some jitter (±25%)
			jitterFactor := float64(time.Now().UnixNano()%1000) / 1000.0 // 0.0 to 1.0
			jitter := time.Duration(float64(baseDelay) * 0.25 * (2*jitterFactor - 1))
			delay := baseDelay + jitter

			// If API provided Retry-After header, respect it
			if rateLimitErr.RetryAfter != "" {
				if retryAfterSeconds, parseErr := strconv.Atoi(rateLimitErr.RetryAfter); parseErr == nil {
					providedDelay := time.Duration(retryAfterSeconds) * time.Second
					if providedDelay > delay {
						delay = providedDelay
					}
				}
			}

			select {
			case <-time.After(delay):
				continue // Retry
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// For non-rate-limit errors, fail immediately
		return nil, err
	}

	return nil, fmt.Errorf("max retries (%d) exceeded", maxRetries)
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
