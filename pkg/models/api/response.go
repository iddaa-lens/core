package api

import "time"

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// EventResponse represents an event in API responses
type EventResponse struct {
	ID                      int32     `json:"id"`
	ExternalID              string    `json:"external_id"`
	Slug                    string    `json:"slug"`
	EventDate               time.Time `json:"event_date"`
	Status                  string    `json:"status"`
	HomeScore               *int32    `json:"home_score,omitempty"`
	AwayScore               *int32    `json:"away_score,omitempty"`
	IsLive                  bool      `json:"is_live"`
	MinuteOfMatch           *int32    `json:"minute_of_match,omitempty"`
	Half                    *int32    `json:"half,omitempty"`
	BettingVolumePercentage *float64  `json:"betting_volume_percentage,omitempty"`
	VolumeRank              *int32    `json:"volume_rank,omitempty"`
	HasKingOdd              bool      `json:"has_king_odd"`
	OddsCount               *int32    `json:"odds_count,omitempty"`
	HasCombine              bool      `json:"has_combine"`
	HomeTeam                string    `json:"home_team"`
	HomeTeamCountry         string    `json:"home_team_country"`
	AwayTeam                string    `json:"away_team"`
	AwayTeamCountry         string    `json:"away_team_country"`
	League                  string    `json:"league"`
	LeagueCountry           string    `json:"league_country"`
	Sport                   string    `json:"sport"`
	SportCode               string    `json:"sport_code"`
	Match                   string    `json:"match"`
}

// BigMoverResponse represents odds movement data
type BigMoverResponse struct {
	EventSlug            string    `json:"event_slug"`
	Match                string    `json:"match"`
	Sport                string    `json:"sport"`
	SportCode            string    `json:"sport_code"`
	League               string    `json:"league"`
	LeagueCountry        string    `json:"league_country"`
	Market               string    `json:"market"`
	MarketDescription    string    `json:"market_description"`
	Outcome              string    `json:"outcome"`
	OpeningOdds          float64   `json:"opening_odds"`
	CurrentOdds          float64   `json:"current_odds"`
	ChangePercentage     float64   `json:"change_percentage"`
	Multiplier           float64   `json:"multiplier"`
	Direction            string    `json:"direction"`
	LastUpdated          time.Time `json:"last_updated"`
	EventTime            time.Time `json:"event_time"`
	EventStatus          string    `json:"event_status"`
	IsLive               bool      `json:"is_live"`
	HomeScore            *int32    `json:"home_score,omitempty"`
	AwayScore            *int32    `json:"away_score,omitempty"`
	MinuteOfMatch        *int32    `json:"minute_of_match,omitempty"`
	BettingVolumePercent *float64  `json:"betting_volume_percent,omitempty"`
	HomeTeamCountry      string    `json:"home_team_country"`
	AwayTeamCountry      string    `json:"away_team_country"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Page        int  `json:"page"`
	PerPage     int  `json:"per_page"`
	Total       int  `json:"total"`
	TotalPages  int  `json:"total_pages"`
	HasNext     bool `json:"has_next"`
	HasPrevious bool `json:"has_previous"`
}

// PaginatedEventsResponse represents paginated events response
type PaginatedEventsResponse struct {
	Data       []EventResponse `json:"data"`
	Pagination PaginationInfo  `json:"pagination"`
}

// Response represents a general API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
	Message string      `json:"message,omitempty"`
}
