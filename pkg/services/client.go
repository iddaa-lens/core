package services

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/iddaa-lens/core/internal/config"
	"github.com/iddaa-lens/core/pkg/logger"
	"github.com/iddaa-lens/core/pkg/models"
)

type IddaaClient struct {
	baseURL string
	client  *http.Client
	logger  *logger.Logger
}

func NewIddaaClient(cfg *config.Config) *IddaaClient {
	return &IddaaClient{
		baseURL: "https://sportsbookv2.iddaa.com",
		client: &http.Client{
			Timeout: time.Duration(cfg.External.Timeout) * time.Second,
		},
		logger: logger.New("iddaa-client"),
	}
}

// generateClientTransactionID creates a unique transaction ID like the real site
func (c *IddaaClient) generateClientTransactionID() string {
	// Generate 16 random bytes and format as UUID-like string
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based ID if random generation fails
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}

	// Format as UUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}

// addBrowserHeaders adds realistic browser headers to avoid bot detection
func (c *IddaaClient) addBrowserHeaders(req *http.Request) {
	// Generate unique client transaction ID for each request
	clientTransactionID := c.generateClientTransactionID()

	// Current timestamp in milliseconds
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())

	// Headers matching exactly what real iddaa.com sends
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="136", "Google Chrome";v="136", "Not.A/Brand";v="99"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Priority", "u=1, i")

	// Iddaa-specific headers that the real site sends
	req.Header.Set("Client-Transaction-Id", clientTransactionID)
	req.Header.Set("Platform", "web")
	req.Header.Set("Timestamp", timestamp)

	// Add referer and origin based on the host
	switch req.URL.Host {
	case "sportsbookv2.iddaa.com":
		req.Header.Set("Referer", "https://www.iddaa.com/")
		req.Header.Set("Origin", "https://www.iddaa.com")
		req.Header.Set("Sec-Fetch-Site", "same-site")
	case "contentv2.iddaa.com", "statisticsv2.iddaa.com":
		req.Header.Set("Referer", "https://www.iddaa.com/")
		req.Header.Set("Sec-Fetch-Site", "same-site")
	}
}

// makeRequest creates a request with browser headers
func (c *IddaaClient) makeRequest(url string) (*http.Response, error) {
	start := time.Now()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.addBrowserHeaders(req)

	resp, err := c.client.Do(req)
	duration := time.Since(start)

	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}

	c.logger.LogAPICall("GET", url, statusCode, duration, err)

	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	return resp, nil
}

func (c *IddaaClient) GetCompetitions() (*models.IddaaAPIResponse[models.IddaaCompetition], error) {
	url := fmt.Sprintf("%s/sportsbook/competitions", c.baseURL)

	resp, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result models.IddaaAPIResponse[models.IddaaCompetition]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.IsSuccess {
		return nil, fmt.Errorf("API request failed: %s", result.Message)
	}

	return &result, nil
}

// GetEvents fetches all events for a specific sport (live + upcoming)
func (c *IddaaClient) GetEvents(sportID int) (*models.IddaaEventsResponse, error) {
	url := fmt.Sprintf("%s/sportsbook/events?st=%d&type=0&version=0", c.baseURL, sportID)

	resp, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result models.IddaaEventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.IsSuccess {
		return nil, fmt.Errorf("API request failed for GetEvents: %s", result.Message)
	}

	return &result, nil
}

// GetLiveEvents fetches only live events for a specific sport
func (c *IddaaClient) GetLiveEvents(sportID int) (*models.IddaaEventsResponse, error) {
	url := fmt.Sprintf("%s/sportsbook/events?st=%d&type=1&version=0", c.baseURL, sportID)

	resp, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result models.IddaaEventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.IsSuccess {
		return nil, fmt.Errorf("API request failed")
	}

	return &result, nil
}

// GetEventsByCompetition fetches events for a specific competition (legacy method)
func (c *IddaaClient) GetEventsByCompetition(competitionID int) (*models.IddaaAPIResponse[models.IddaaEvent], error) {
	url := fmt.Sprintf("%s/sportsbook/competitions/%d/events", c.baseURL, competitionID)

	resp, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result models.IddaaAPIResponse[models.IddaaEvent]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.IsSuccess {
		return nil, fmt.Errorf("API request failed: %s", result.Message)
	}

	return &result, nil
}

func (c *IddaaClient) GetOdds(eventID int) (*models.IddaaAPIResponse[models.IddaaOdds], error) {
	url := fmt.Sprintf("%s/sportsbook/events/%d/odds", c.baseURL, eventID)

	resp, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result models.IddaaAPIResponse[models.IddaaOdds]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.IsSuccess {
		return nil, fmt.Errorf("API request failed: %s", result.Message)
	}

	return &result, nil
}

func (c *IddaaClient) GetAppConfig(platform string) (*models.IddaaConfigResponse, error) {
	url := fmt.Sprintf("https://contentv2.iddaa.com/appconfig?platform=%s", platform)

	resp, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result models.IddaaConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.IsSuccess {
		return nil, fmt.Errorf("API request failed: %s", result.Message)
	}

	return &result, nil
}

func (c *IddaaClient) GetSportInfo() (*models.IddaaAPIResponse[models.IddaaSportInfo], error) {
	url := fmt.Sprintf("%s/sportsbook/info", c.baseURL)

	resp, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result models.IddaaAPIResponse[models.IddaaSportInfo]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.IsSuccess {
		return nil, fmt.Errorf("API request failed: %s", result.Message)
	}

	return &result, nil
}

func (c *IddaaClient) GetMarketConfig() (*models.IddaaMarketConfigResponse, error) {
	url := fmt.Sprintf("%s/sportsbook/get_market_config", c.baseURL)

	resp, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result models.IddaaMarketConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.IsSuccess {
		return nil, fmt.Errorf("API request failed: %s", result.Message)
	}

	return &result, nil
}

func (c *IddaaClient) GetEventStatistics(sportID int, searchDate string) ([]models.IddaaEventStatistics, error) {
	url := fmt.Sprintf("https://statisticsv2.iddaa.com/broadage/getEventListCache?SportId=%d&SearchDate=%s", sportID, searchDate)

	resp, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Read the response body to handle multiple response formats
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to decode as standard wrapper format first
	var rawResult struct {
		Data      json.RawMessage `json:"data"`
		IsSuccess bool            `json:"isSuccess"`
		Message   string          `json:"message"`
	}

	if err := json.Unmarshal(body, &rawResult); err != nil {
		// If wrapper format fails, try direct array
		c.logger.Debug().
			Str("action", "try_direct_array").
			Str("url", url).
			Msg("Standard wrapper format failed, trying direct array")

		var directStats []models.IddaaEventStatistics
		if err := json.Unmarshal(body, &directStats); err != nil {
			c.logger.Error().
				Err(err).
				Str("action", "decode_failed").
				Str("response_preview", string(body[:min(200, len(body))])).
				Msg("Failed to decode response in any known format")
			return nil, fmt.Errorf("failed to decode response as wrapper or direct array: %w", err)
		}
		return directStats, nil
	}

	if !rawResult.IsSuccess {
		return nil, fmt.Errorf("API request failed: %s", rawResult.Message)
	}

	// Check if data is an array or object
	var stats []models.IddaaEventStatistics

	// Try to unmarshal as array first
	if err := json.Unmarshal(rawResult.Data, &stats); err != nil {
		// If that fails, try as object (map of eventID -> stats)
		var statsMap map[string]models.IddaaEventStatistics
		if err := json.Unmarshal(rawResult.Data, &statsMap); err != nil {
			// If both fail, try as a single object
			var singleStat models.IddaaEventStatistics
			if err := json.Unmarshal(rawResult.Data, &singleStat); err != nil {
				// Log the data for debugging
				previewLen := 200
				if len(rawResult.Data) < previewLen {
					previewLen = len(rawResult.Data)
				}
				c.logger.Error().
					Err(err).
					Str("action", "statistics_decode_failed").
					Str("data_preview", string(rawResult.Data[:previewLen])).
					Msg("Failed to decode statistics data in any format")
				return nil, fmt.Errorf("failed to decode data field as array, map, or single object: %w", err)
			}
			stats = []models.IddaaEventStatistics{singleStat}
		} else {
			// Convert map to slice
			stats = make([]models.IddaaEventStatistics, 0, len(statsMap))
			for _, stat := range statsMap {
				stats = append(stats, stat)
			}
		}
	}

	return stats, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *IddaaClient) GetSingleEvent(eventID int) (*models.IddaaSingleEventResponse, error) {
	url := fmt.Sprintf("%s/sportsbook/event/%d", c.baseURL, eventID)

	resp, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result models.IddaaSingleEventResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.IsSuccess {
		return nil, fmt.Errorf("API request failed: %s", result.Message)
	}

	return &result, nil
}

// FetchData fetches raw JSON data from the given URL
func (c *IddaaClient) FetchData(url string) ([]byte, error) {
	resp, err := c.makeRequest(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data from %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}
