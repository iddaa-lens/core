package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iddaa-lens/core/internal/config"
	"github.com/iddaa-lens/core/pkg/models"
)

func TestIddaaClient_GetCompetitions(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		wantError      bool
		wantSuccess    bool
		wantDataCount  int
	}{
		{
			name: "successful response",
			serverResponse: `{
				"isSuccess": true,
				"data": [
					{
						"i": 1,
						"cid": "TR",
						"p": 100,
						"ic": "https://example.com/icon.png",
						"sn": "Süper Lig",
						"si": "1",
						"n": "Türkiye Süper Lig",
						"cref": 1000
					}
				],
				"message": ""
			}`,
			serverStatus:  http.StatusOK,
			wantError:     false,
			wantSuccess:   true,
			wantDataCount: 1,
		},
		{
			name: "API failure response",
			serverResponse: `{
				"isSuccess": false,
				"data": [],
				"message": "Internal server error"
			}`,
			serverStatus: http.StatusOK,
			wantError:    true,
			wantSuccess:  false,
		},
		{
			name:           "HTTP error",
			serverResponse: "",
			serverStatus:   http.StatusInternalServerError,
			wantError:      true,
		},
		{
			name:           "invalid JSON",
			serverResponse: "invalid json",
			serverStatus:   http.StatusOK,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/sportsbook/competitions" {
					t.Errorf("Expected path /sportsbook/competitions, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			cfg := &config.Config{
				External: config.ExternalAPIConfig{
					Timeout: 30,
				},
			}

			client := NewIddaaClient(cfg)
			client.baseURL = server.URL

			result, err := client.GetCompetitions()

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.IsSuccess != tt.wantSuccess {
				t.Errorf("Expected isSuccess=%v, got %v", tt.wantSuccess, result.IsSuccess)
			}

			if len(result.Data) != tt.wantDataCount {
				t.Errorf("Expected %d competitions, got %d", tt.wantDataCount, len(result.Data))
			}

			if tt.wantDataCount > 0 {
				comp := result.Data[0]
				if comp.ID != 1 {
					t.Errorf("Expected competition ID 1, got %d", comp.ID)
				}
				if comp.CountryID != "TR" {
					t.Errorf("Expected country ID TR, got %s", comp.CountryID)
				}
			}
		})
	}
}

func TestIddaaClient_GetEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Updated to expect the new events endpoint with sport ID and type parameters
		if r.URL.RawQuery != "st=1&type=0&version=0" {
			t.Errorf("Expected query st=1&type=0&version=0, got %s", r.URL.RawQuery)
		}

		response := models.IddaaEventsResponse{
			IsSuccess: true,
			Data: &models.IddaaEventsData{
				IsDiff:  false,
				Version: 1,
				Events: []models.IddaaEvent{
					{
						ID:            456,
						CompetitionID: 123,
						Date:          1704141600, // Unix timestamp in seconds
						HomeTeam:      "Team A",
						AwayTeam:      "Team B",
						Status:        1,
						SportID:       1,
						BulletinID:    789,
						Version:       1,
						BetProgram:    1,
						IsLive:        false,
						MBC:           1,
						HasKingOdd:    false,
						OddsCount:     5,
						HasCombine:    true,
						Markets:       []models.IddaaMarket{},
					},
				},
			},
			Message: "",
		}

		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{
		External: config.ExternalAPIConfig{
			Timeout: 30,
		},
	}

	client := NewIddaaClient(cfg)
	client.baseURL = server.URL

	result, err := client.GetEvents(1) // Pass sport ID instead of competition ID

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if !result.IsSuccess {
		t.Errorf("Expected successful response")
	}

	if result.Data == nil {
		t.Errorf("Expected data to be non-nil")
		return
	}

	if len(result.Data.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(result.Data.Events))
	}

	event := result.Data.Events[0]
	if event.ID != 456 {
		t.Errorf("Expected event ID 456, got %d", event.ID)
	}
	if event.CompetitionID != 123 {
		t.Errorf("Expected competition ID 123, got %d", event.CompetitionID)
	}
}
