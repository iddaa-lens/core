package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/betslib/iddaa-core/internal/config"
	"github.com/betslib/iddaa-core/pkg/models"
)

type IddaaClient struct {
	baseURL string
	client  *http.Client
}

func NewIddaaClient(cfg *config.Config) *IddaaClient {
	return &IddaaClient{
		baseURL: "https://sportsbookv2.iddaa.com",
		client: &http.Client{
			Timeout: time.Duration(cfg.External.Timeout) * time.Second,
		},
	}
}

func (c *IddaaClient) GetCompetitions() (*models.IddaaAPIResponse[models.IddaaCompetition], error) {
	url := fmt.Sprintf("%s/sportsbook/competitions", c.baseURL)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
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

func (c *IddaaClient) GetEvents(competitionID int) (*models.IddaaAPIResponse[models.IddaaEvent], error) {
	url := fmt.Sprintf("%s/sportsbook/competitions/%d/events", c.baseURL, competitionID)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
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

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
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

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
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

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
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

func (c *IddaaClient) GetMarketConfig() (*models.IddaaAPIResponse[models.IddaaMarketConfig], error) {
	url := fmt.Sprintf("%s/sportsbook/get_market_config", c.baseURL)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result models.IddaaAPIResponse[models.IddaaMarketConfig]
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

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result []models.IddaaEventStatistics
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func (c *IddaaClient) GetSingleEvent(eventID int) (*models.IddaaSingleEventResponse, error) {
	url := fmt.Sprintf("%s/sportsbook/event/%d", c.baseURL, eventID)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
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
	resp, err := c.client.Get(url)
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
