package models

import (
	"fmt"
	"time"
)

// Football API Models for external API integration

// FootballAPILeague represents a league from Football API
type FootballAPILeague struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Country string `json:"country"`
	Logo    string `json:"logo"`
	Flag    string `json:"flag"`
	Season  int    `json:"season"`
	Type    string `json:"type"`
}

// FootballAPITeam represents a team from Football API
type FootballAPITeam struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Country  string `json:"country"`
	Founded  int    `json:"founded"`
	National bool   `json:"national"`
	Logo     string `json:"logo"`
}

// FootballAPIVenue represents a venue from Football API
type FootballAPIVenue struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	City     string `json:"city"`
	Capacity int    `json:"capacity"`
	Surface  string `json:"surface"`
	Image    string `json:"image"`
}

// FootballAPICountry represents a country from Football API
type FootballAPICountry struct {
	Name string `json:"name"`
	Code string `json:"code"`
	Flag string `json:"flag"`
}

// FootballAPIResponse represents the standard response structure
type FootballAPIResponse struct {
	Get        string                 `json:"get"`
	Parameters map[string]interface{} `json:"parameters"`
	Errors     interface{}            `json:"errors"` // Can be []string or map[string]interface{}
	Results    int                    `json:"results"`
	Paging     FootballAPIPaging      `json:"paging"`
	Response   interface{}            `json:"response"`
}

// FootballAPIPaging represents pagination info
type FootballAPIPaging struct {
	Current int `json:"current"`
	Total   int `json:"total"`
}

// FootballAPILeaguesResponse represents leagues API response
type FootballAPILeaguesResponse struct {
	Get        string                  `json:"get"`
	Parameters map[string]interface{}  `json:"parameters"`
	Errors     interface{}             `json:"errors"` // Can be []string or map[string]interface{}
	Results    int                     `json:"results"`
	Paging     FootballAPIPaging       `json:"paging"`
	Response   []FootballAPILeagueData `json:"response"`
}

// FootballAPILeagueData represents individual league data from API
type FootballAPILeagueData struct {
	League  FootballAPILeague   `json:"league"`
	Country FootballAPICountry  `json:"country"`
	Seasons []FootballAPISeason `json:"seasons"`
}

// FootballAPISeason represents season information
type FootballAPISeason struct {
	Year     int      `json:"year"`
	Start    string   `json:"start"`
	End      string   `json:"end"`
	Current  bool     `json:"current"`
	Coverage Coverage `json:"coverage"`
}

// Coverage represents what data is available for a season
type Coverage struct {
	Fixtures    CoverageDetails `json:"fixtures"`
	Standings   bool            `json:"standings"`
	Players     bool            `json:"players"`
	TopScorers  bool            `json:"top_scorers"`
	TopAssists  bool            `json:"top_assists"`
	TopCards    bool            `json:"top_cards"`
	Injuries    bool            `json:"injuries"`
	Predictions bool            `json:"predictions"`
	Odds        bool            `json:"odds"`
}

// CoverageDetails represents fixture coverage details
type CoverageDetails struct {
	Events             bool `json:"events"`
	Lineups            bool `json:"lineups"`
	StatisticsFixtures bool `json:"statistics_fixtures"`
	StatisticsPlayers  bool `json:"statistics_players"`
}

// FootballAPITeamsResponse represents teams API response
type FootballAPITeamsResponse struct {
	Get        string                 `json:"get"`
	Parameters map[string]interface{} `json:"parameters"`
	Errors     interface{}            `json:"errors"` // Can be []string or map[string]interface{}
	Results    int                    `json:"results"`
	Paging     FootballAPIPaging      `json:"paging"`
	Response   []FootballAPITeamData  `json:"response"`
}

// FootballAPITeamData represents individual team data from API
type FootballAPITeamData struct {
	Team  FootballAPITeam  `json:"team"`
	Venue FootballAPIVenue `json:"venue"`
}

// LeagueMapping represents the mapping between internal and external leagues
type LeagueMapping struct {
	ID                  int       `json:"id" db:"id"`
	InternalLeagueID    int       `json:"internal_league_id" db:"internal_league_id"`
	FootballAPILeagueID int       `json:"football_api_league_id" db:"football_api_league_id"`
	Confidence          float64   `json:"confidence" db:"confidence"`
	MappingMethod       string    `json:"mapping_method" db:"mapping_method"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// TeamMapping represents the mapping between internal and external teams
type TeamMapping struct {
	ID                int       `json:"id" db:"id"`
	InternalTeamID    int       `json:"internal_team_id" db:"internal_team_id"`
	FootballAPITeamID int       `json:"football_api_team_id" db:"football_api_team_id"`
	Confidence        float64   `json:"confidence" db:"confidence"`
	MappingMethod     string    `json:"mapping_method" db:"mapping_method"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// Type aliases for league enrichment
type APIFootballLeagueDetail = FootballAPILeagueData
type FootballAPILeagueDetailResponse = FootballAPILeaguesResponse
type Season = FootballAPISeason

// SearchResult represents a search result with similarity score
type SearchResult struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	Country    string  `json:"country,omitempty"`
	Similarity float64 `json:"similarity"`
	Method     string  `json:"method"`
}

// FootballAPIConfig represents Football API configuration
type FootballAPIConfig struct {
	APIKey    string        `json:"api_key"`
	BaseURL   string        `json:"base_url"`
	Timeout   time.Duration `json:"timeout"`
	RateLimit RateLimit     `json:"rate_limit"`
}

// RateLimit represents API rate limiting configuration
type RateLimit struct {
	RequestsPerMinute int           `json:"requests_per_minute"`
	RequestsPerHour   int           `json:"requests_per_hour"`
	RequestsPerDay    int           `json:"requests_per_day"`
	Burst             int           `json:"burst"`
	Refill            time.Duration `json:"refill"`
}

// SyncStats represents synchronization statistics
type SyncStats struct {
	LeaguesProcessed   int       `json:"leagues_processed"`
	LeaguesMapped      int       `json:"leagues_mapped"`
	TeamsProcessed     int       `json:"teams_processed"`
	TeamsMapped        int       `json:"teams_mapped"`
	APIRequestsMade    int       `json:"api_requests_made"`
	ErrorsEncountered  int       `json:"errors_encountered"`
	SyncStartTime      time.Time `json:"sync_start_time"`
	SyncDuration       string    `json:"sync_duration"`
	LastSuccessfulSync time.Time `json:"last_successful_sync"`
}

// MappingMethodType represents different mapping methods
type MappingMethodType string

const (
	MappingMethodExact       MappingMethodType = "exact"
	MappingMethodFuzzy       MappingMethodType = "fuzzy"
	MappingMethodLevenshtein MappingMethodType = "levenshtein"
	MappingMethodKeyword     MappingMethodType = "keyword"
	MappingMethodManual      MappingMethodType = "manual"
)

// ConfidenceThreshold represents minimum confidence for mappings
const (
	HighConfidenceThreshold   = 0.9
	MediumConfidenceThreshold = 0.7
	LowConfidenceThreshold    = 0.5
)

// HasErrors checks if the Football API response contains errors
func (r *FootballAPILeaguesResponse) HasErrors() bool {
	return r.hasErrors()
}

// HasErrors checks if the Football API response contains errors
func (r *FootballAPITeamsResponse) HasErrors() bool {
	return r.hasErrors()
}

// GetErrorMessages extracts error messages from the response
func (r *FootballAPILeaguesResponse) GetErrorMessages() []string {
	return r.getErrorMessages()
}

// GetErrorMessages extracts error messages from the response
func (r *FootballAPITeamsResponse) GetErrorMessages() []string {
	return r.getErrorMessages()
}

// hasErrors checks if there are any errors in the response
func (r *FootballAPILeaguesResponse) hasErrors() bool {
	if r.Errors == nil {
		return false
	}

	switch errors := r.Errors.(type) {
	case []string:
		return len(errors) > 0
	case map[string]interface{}:
		return len(errors) > 0
	case string:
		return errors != ""
	default:
		return false
	}
}

// hasErrors checks if there are any errors in the response
func (r *FootballAPITeamsResponse) hasErrors() bool {
	if r.Errors == nil {
		return false
	}

	switch errors := r.Errors.(type) {
	case []string:
		return len(errors) > 0
	case map[string]interface{}:
		return len(errors) > 0
	case string:
		return errors != ""
	default:
		return false
	}
}

// getErrorMessages extracts error messages as strings
func (r *FootballAPILeaguesResponse) getErrorMessages() []string {
	return extractErrorMessages(r.Errors)
}

// getErrorMessages extracts error messages as strings
func (r *FootballAPITeamsResponse) getErrorMessages() []string {
	return extractErrorMessages(r.Errors)
}

// extractErrorMessages helper function to extract error messages
func extractErrorMessages(errors interface{}) []string {
	if errors == nil {
		return nil
	}

	switch e := errors.(type) {
	case []string:
		return e
	case string:
		return []string{e}
	case map[string]interface{}:
		var messages []string
		for key, value := range e {
			if valueStr, ok := value.(string); ok {
				messages = append(messages, key+": "+valueStr)
			} else {
				messages = append(messages, key+": "+fmt.Sprintf("%v", value))
			}
		}
		return messages
	case []interface{}:
		var messages []string
		for _, item := range e {
			if itemStr, ok := item.(string); ok {
				messages = append(messages, itemStr)
			} else {
				messages = append(messages, fmt.Sprintf("%v", item))
			}
		}
		return messages
	default:
		return []string{fmt.Sprintf("%v", errors)}
	}
}
