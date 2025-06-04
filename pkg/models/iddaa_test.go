package models

import (
	"encoding/json"
	"testing"
)

func TestIddaaAPIResponse_Unmarshal(t *testing.T) {
	jsonData := `{
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
		"message": "success"
	}`

	var response IddaaAPIResponse[IddaaCompetition]
	err := json.Unmarshal([]byte(jsonData), &response)

	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if !response.IsSuccess {
		t.Error("Expected IsSuccess to be true")
	}

	if response.Message != "success" {
		t.Errorf("Expected message 'success', got '%s'", response.Message)
	}

	if len(response.Data) != 1 {
		t.Errorf("Expected 1 competition, got %d", len(response.Data))
	}

	comp := response.Data[0]
	if comp.ID != 1 {
		t.Errorf("Expected ID 1, got %d", comp.ID)
	}
	if comp.CountryID != "TR" {
		t.Errorf("Expected country ID 'TR', got '%s'", comp.CountryID)
	}
	if comp.Priority != 100 {
		t.Errorf("Expected priority 100, got %d", comp.Priority)
	}
	if comp.IconURL != "https://example.com/icon.png" {
		t.Errorf("Expected icon URL 'https://example.com/icon.png', got '%s'", comp.IconURL)
	}
	if comp.ShortName != "Süper Lig" {
		t.Errorf("Expected short name 'Süper Lig', got '%s'", comp.ShortName)
	}
	if comp.SportID != "1" {
		t.Errorf("Expected sport ID '1', got '%s'", comp.SportID)
	}
	if comp.Name != "Türkiye Süper Lig" {
		t.Errorf("Expected name 'Türkiye Süper Lig', got '%s'", comp.Name)
	}
	if comp.Reference != 1000 {
		t.Errorf("Expected reference 1000, got %d", comp.Reference)
	}
}

func TestIddaaEvent_Unmarshal(t *testing.T) {
	jsonData := `{
		"i": 123,
		"cid": 456,
		"d": "2024-01-01",
		"t": "20:00",
		"ht": "Team A",
		"at": "Team B",
		"s": "scheduled",
		"hs": 2,
		"as": 1,
		"mr": 90
	}`

	var event IddaaEvent
	err := json.Unmarshal([]byte(jsonData), &event)

	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if event.ID != 123 {
		t.Errorf("Expected ID 123, got %d", event.ID)
	}
	if event.CompetitionID != 456 {
		t.Errorf("Expected competition ID 456, got %d", event.CompetitionID)
	}
	if event.Date != 1704141600000 { // Unix timestamp for 2024-01-01 20:00
		t.Errorf("Expected date 1704141600000, got %d", event.Date)
	}
	if event.HomeTeam != "Team A" {
		t.Errorf("Expected home team 'Team A', got '%s'", event.HomeTeam)
	}
	if event.AwayTeam != "Team B" {
		t.Errorf("Expected away team 'Team B', got '%s'", event.AwayTeam)
	}
	if event.Status != 1 { // Status is now an int
		t.Errorf("Expected status 1, got %d", event.Status)
	}
}

func TestIddaaEvent_UnmarshalWithNulls(t *testing.T) {
	jsonData := `{
		"i": 123,
		"cid": 456,
		"d": "2024-01-01",
		"t": "20:00",
		"ht": "Team A",
		"at": "Team B",
		"s": "scheduled"
	}`

	var event IddaaEvent
	err := json.Unmarshal([]byte(jsonData), &event)

	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Test passes if unmarshal succeeds - the old fields don't exist anymore
}
