package utils

import (
	"testing"
)

func TestNormalizeSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic text with spaces",
			input:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "Turkish characters",
			input:    "İstanbul Başakşehir",
			expected: "istanbul-basaksehir",
		},
		{
			name:     "Turkish special characters",
			input:    "Galatasaray İçin Güzel Şehir Ölçüsü",
			expected: "galatasaray-icin-guzel-sehir-olcusu",
		},
		{
			name:     "German special characters",
			input:    "Bayern München",
			expected: "bayern-munchen",
		},
		{
			name:     "French special characters",
			input:    "Olympique Marseille",
			expected: "olympique-marseille",
		},
		{
			name:     "Spanish special characters",
			input:    "Real Madrid España",
			expected: "real-madrid-espana",
		},
		{
			name:     "Mixed special characters",
			input:    "Fenerbahçe-Galatasaray",
			expected: "fenerbahce-galatasaray",
		},
		{
			name:     "Numbers and special chars",
			input:    "Team 123! @#$% Test",
			expected: "team-123-at-test",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only special characters",
			input:    "!@#$%^&*()",
			expected: "at-and",
		},
		{
			name:     "Multiple spaces and hyphens",
			input:    "Test    ---    Multiple   Spaces",
			expected: "test-multiple-spaces",
		},
		{
			name:     "Leading and trailing spaces",
			input:    "   Test Text   ",
			expected: "test-text",
		},
		{
			name:     "Accented characters",
			input:    "Café Résumé Naïve",
			expected: "cafe-resume-naive",
		},
		{
			name:     "Polish characters",
			input:    "Kraków Łódź Gdańsk",
			expected: "krakow-lodz-gdansk",
		},
		{
			name:     "Czech characters",
			input:    "Praha Brno Ostrava",
			expected: "praha-brno-ostrava",
		},
		{
			name:     "Real team names",
			input:    "FC Barcelona vs Real Madrid",
			expected: "fc-barcelona-vs-real-madrid",
		},
		{
			name:     "Turkish team example",
			input:    "Fenerbahçe vs Galatasaray",
			expected: "fenerbahce-vs-galatasaray",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeSlug(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeSlug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateEventSlug(t *testing.T) {
	tests := []struct {
		name       string
		homeTeam   string
		awayTeam   string
		externalID string
		expected   string
	}{
		{
			name:       "Basic event",
			homeTeam:   "Arsenal",
			awayTeam:   "Chelsea",
			externalID: "12345",
			expected:   "arsenal-vs-chelsea-12345",
		},
		{
			name:       "Turkish teams",
			homeTeam:   "Fenerbahçe",
			awayTeam:   "Galatasaray",
			externalID: "67890",
			expected:   "fenerbahce-vs-galatasaray-67890",
		},
		{
			name:       "German teams",
			homeTeam:   "Bayern München",
			awayTeam:   "Borussia Dortmund",
			externalID: "54321",
			expected:   "bayern-munchen-vs-borussia-dortmund-54321",
		},
		{
			name:       "Empty team names",
			homeTeam:   "",
			awayTeam:   "",
			externalID: "99999",
			expected:   "team-vs-team-99999",
		},
		{
			name:       "Special characters in team names",
			homeTeam:   "FC Barcelona!!!",
			awayTeam:   "Real Madrid C.F.",
			externalID: "11111",
			expected:   "fc-barcelona-vs-real-madrid-c-f-11111",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateEventSlug(tt.homeTeam, tt.awayTeam, tt.externalID)
			if result != tt.expected {
				t.Errorf("GenerateEventSlug(%q, %q, %q) = %q, want %q",
					tt.homeTeam, tt.awayTeam, tt.externalID, result, tt.expected)
			}
		})
	}
}

func TestGenerateTeamSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "English team",
			input:    "Manchester United",
			expected: "manchester-united",
		},
		{
			name:     "Turkish team",
			input:    "Fenerbahçe",
			expected: "fenerbahce",
		},
		{
			name:     "German team",
			input:    "Bayern München",
			expected: "bayern-munchen",
		},
		{
			name:     "Empty name",
			input:    "",
			expected: "team",
		},
		{
			name:     "Special characters",
			input:    "FC Barcelona!!!",
			expected: "fc-barcelona",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateTeamSlug(tt.input)
			if result != tt.expected {
				t.Errorf("GenerateTeamSlug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateLeagueSlug(t *testing.T) {
	tests := []struct {
		name       string
		leagueName string
		country    string
		expected   string
	}{
		{
			name:       "Premier League",
			leagueName: "Premier League",
			country:    "England",
			expected:   "premier-league-england",
		},
		{
			name:       "Turkish Super League",
			leagueName: "Süper Lig",
			country:    "Türkiye",
			expected:   "super-lig-turkiye",
		},
		{
			name:       "No country",
			leagueName: "Champions League",
			country:    "",
			expected:   "champions-league",
		},
		{
			name:       "Empty league name",
			leagueName: "",
			country:    "Spain",
			expected:   "league-spain",
		},
		{
			name:       "Both empty",
			leagueName: "",
			country:    "",
			expected:   "league",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateLeagueSlug(tt.leagueName, tt.country)
			if result != tt.expected {
				t.Errorf("GenerateLeagueSlug(%q, %q) = %q, want %q",
					tt.leagueName, tt.country, result, tt.expected)
			}
		})
	}
}

func TestGenerateMarketTypeSlug(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		typeName string
		expected string
	}{
		{
			name:     "Basic market type",
			code:     "1X2",
			typeName: "Match Result",
			expected: "1x2-match-result",
		},
		{
			name:     "Turkish market type",
			code:     "MARKET_7",
			typeName: "Gol Sayısı",
			expected: "market-7-gol-sayisi",
		},
		{
			name:     "Only code",
			code:     "OU_2_5",
			typeName: "",
			expected: "ou-2-5",
		},
		{
			name:     "Only name",
			code:     "",
			typeName: "Over/Under Goals",
			expected: "over-under-goals",
		},
		{
			name:     "Both empty",
			code:     "",
			typeName: "",
			expected: "market",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateMarketTypeSlug(tt.code, tt.typeName)
			if result != tt.expected {
				t.Errorf("GenerateMarketTypeSlug(%q, %q) = %q, want %q",
					tt.code, tt.typeName, result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkNormalizeSlug(b *testing.B) {
	input := "Fenerbahçe vs Galatasaray İçin Güzel Şehir Ölçüsü"
	for i := 0; i < b.N; i++ {
		NormalizeSlug(input)
	}
}

func BenchmarkGenerateEventSlug(b *testing.B) {
	homeTeam := "Bayern München"
	awayTeam := "Borussia Dortmund"
	externalID := "12345"
	for i := 0; i < b.N; i++ {
		GenerateEventSlug(homeTeam, awayTeam, externalID)
	}
}
