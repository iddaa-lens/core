package services

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/betslib/iddaa-core/pkg/logger"
)

// Mock HTTP response
type mockRoundTripper struct {
	response *http.Response
	err      error
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

func TestGetEventStatistics_DirectArray(t *testing.T) {
	// Test response with direct array format (no wrapper)
	directArrayResponse := `[
		{
			"EventId": 123,
			"BulletinId": 456,
			"EventNo": "E123",
			"League": "Test League",
			"HomeTeam": "Team A",
			"AwayTeam": "Team B",
			"MatchDate": "2025-06-05",
			"Status": 1,
			"Half": 0,
			"MinuteOfMatch": 0,
			"HomeScore": 0,
			"AwayScore": 0,
			"HalfTimeScore": "0-0",
			"FullTimeScore": "0-0",
			"Statistics": {
				"HomeTeam": {"Shots": 5, "ShotsOnTarget": 2},
				"AwayTeam": {"Shots": 3, "ShotsOnTarget": 1}
			},
			"Events": [],
			"IsLive": false,
			"HasStatistics": true,
			"SportId": 1
		}
	]`

	client := &http.Client{
		Transport: &mockRoundTripper{
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(directArrayResponse)),
			},
		},
	}

	iddaaClient := &IddaaClient{
		baseURL: "https://test.com",
		client:  client,
		logger:  logger.New("test"),
	}

	stats, err := iddaaClient.GetEventStatistics(1, "2025-06-05")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(stats) != 1 {
		t.Errorf("Expected 1 stat, got %d", len(stats))
	}

	if stats[0].EventID != 123 {
		t.Errorf("Expected EventID 123, got %d", stats[0].EventID)
	}
}

func TestGetEventStatistics_WrappedResponse(t *testing.T) {
	// Test response with standard wrapper format
	wrappedResponse := `{
		"isSuccess": true,
		"message": "Success",
		"data": [
			{
				"EventId": 456,
				"BulletinId": 789,
				"EventNo": "E456",
				"League": "Test League 2",
				"HomeTeam": "Team C",
				"AwayTeam": "Team D",
				"MatchDate": "2025-06-05",
				"Status": 2,
				"Half": 1,
				"MinuteOfMatch": 45,
				"HomeScore": 1,
				"AwayScore": 0,
				"HalfTimeScore": "1-0",
				"FullTimeScore": "",
				"Statistics": {
					"HomeTeam": {"Shots": 8, "ShotsOnTarget": 4},
					"AwayTeam": {"Shots": 6, "ShotsOnTarget": 2}
				},
				"Events": [],
				"IsLive": true,
				"HasStatistics": true,
				"SportId": 1
			}
		]
	}`

	client := &http.Client{
		Transport: &mockRoundTripper{
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(wrappedResponse)),
			},
		},
	}

	iddaaClient := &IddaaClient{
		baseURL: "https://test.com",
		client:  client,
		logger:  logger.New("test"),
	}

	stats, err := iddaaClient.GetEventStatistics(1, "2025-06-05")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(stats) != 1 {
		t.Errorf("Expected 1 stat, got %d", len(stats))
	}

	if stats[0].EventID != 456 {
		t.Errorf("Expected EventID 456, got %d", stats[0].EventID)
	}

	if !stats[0].IsLive {
		t.Errorf("Expected IsLive to be true")
	}
}

func TestGetEventStatistics_ObjectMapResponse(t *testing.T) {
	// Test response with object map format (eventID -> stats)
	objectMapResponse := `{
		"isSuccess": true,
		"message": "Success",
		"data": {
			"789": {
				"EventId": 789,
				"BulletinId": 999,
				"EventNo": "E789",
				"League": "Test League 3",
				"HomeTeam": "Team E",
				"AwayTeam": "Team F",
				"MatchDate": "2025-06-05",
				"Status": 0,
				"Half": 0,
				"MinuteOfMatch": 0,
				"HomeScore": 0,
				"AwayScore": 0,
				"HalfTimeScore": "",
				"FullTimeScore": "",
				"Statistics": {
					"HomeTeam": {"Shots": 0, "ShotsOnTarget": 0},
					"AwayTeam": {"Shots": 0, "ShotsOnTarget": 0}
				},
				"Events": [],
				"IsLive": false,
				"HasStatistics": false,
				"SportId": 1
			}
		}
	}`

	client := &http.Client{
		Transport: &mockRoundTripper{
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(objectMapResponse)),
			},
		},
	}

	iddaaClient := &IddaaClient{
		baseURL: "https://test.com",
		client:  client,
		logger:  logger.New("test"),
	}

	stats, err := iddaaClient.GetEventStatistics(1, "2025-06-05")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(stats) != 1 {
		t.Errorf("Expected 1 stat, got %d", len(stats))
	}

	if stats[0].EventID != 789 {
		t.Errorf("Expected EventID 789, got %d", stats[0].EventID)
	}

	if stats[0].HasStatistics {
		t.Errorf("Expected HasStatistics to be false")
	}
}

func TestGetEventStatistics_APIFailure(t *testing.T) {
	// Test API failure response
	failureResponse := `{
		"isSuccess": false,
		"message": "API Error: No data found",
		"data": null
	}`

	client := &http.Client{
		Transport: &mockRoundTripper{
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(failureResponse)),
			},
		},
	}

	iddaaClient := &IddaaClient{
		baseURL: "https://test.com",
		client:  client,
		logger:  logger.New("test"),
	}

	_, err := iddaaClient.GetEventStatistics(1, "2025-06-05")
	if err == nil {
		t.Errorf("Expected error for API failure, got nil")
	}

	expectedError := "API request failed: API Error: No data found"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}
