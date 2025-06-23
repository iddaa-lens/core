package apifootball

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/iddaa-lens/core/pkg/models"
)

// Team-related endpoints and functionality

// GetTeams fetches teams with various filter options
func (c *Client) GetTeams(ctx context.Context, params map[string]string) ([]models.FootballAPITeamData, error) {
	// Use retry logic for rate limit handling (max 3 retries)
	response, err := c.makeRequestWithRetry(ctx, "/teams", params, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to get teams: %w", err)
	}

	var teamsData []models.FootballAPITeamData
	if err := json.Unmarshal(response.Response, &teamsData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal teams response: %w", err)
	}

	return teamsData, nil
}

// GetTeamByID fetches a specific team by ID
func (c *Client) GetTeamByID(ctx context.Context, teamID int) (*models.FootballAPITeamData, error) {
	params := ParamID(teamID)
	teams, err := c.GetTeams(ctx, params)
	if err != nil {
		return nil, err
	}

	if len(teams) == 0 {
		return nil, fmt.Errorf("team with ID %d not found", teamID)
	}

	return &teams[0], nil
}

// GetTeamsByName searches teams by name
func (c *Client) GetTeamsByName(ctx context.Context, name string) ([]models.FootballAPITeamData, error) {
	params := ParamName(name)
	return c.GetTeams(ctx, params)
}

// GetTeamsByCountry fetches teams for a specific country
func (c *Client) GetTeamsByCountry(ctx context.Context, country string) ([]models.FootballAPITeamData, error) {
	params := ParamCountry(country)
	return c.GetTeams(ctx, params)
}

// GetTeamsByCode fetches teams by country code
func (c *Client) GetTeamsByCode(ctx context.Context, code string) ([]models.FootballAPITeamData, error) {
	params := ParamCode(code)
	return c.GetTeams(ctx, params)
}

// GetTeamsBySeason fetches teams for a specific season
func (c *Client) GetTeamsBySeason(ctx context.Context, season int) ([]models.FootballAPITeamData, error) {
	params := ParamSeason(season)
	return c.GetTeams(ctx, params)
}

// GetTeamsByLeague fetches teams that play in a specific league
func (c *Client) GetTeamsByLeague(ctx context.Context, leagueID int) ([]models.FootballAPITeamData, error) {
	params := map[string]string{"league": strconv.Itoa(leagueID)}
	return c.GetTeams(ctx, params)
}

// GetTeamsByLeagueAndSeason fetches teams for a specific league and season
func (c *Client) GetTeamsByLeagueAndSeason(ctx context.Context, leagueID, season int) ([]models.FootballAPITeamData, error) {
	params := MergeParams(
		map[string]string{"league": strconv.Itoa(leagueID)},
		ParamSeason(season),
	)
	return c.GetTeams(ctx, params)
}

// GetTeamsByVenue fetches teams that play at a specific venue
func (c *Client) GetTeamsByVenue(ctx context.Context, venueID int) ([]models.FootballAPITeamData, error) {
	params := map[string]string{"venue": strconv.Itoa(venueID)}
	return c.GetTeams(ctx, params)
}

// SearchTeamsByText searches teams by any text (name or country)
func (c *Client) SearchTeamsByText(ctx context.Context, searchText string) ([]models.FootballAPITeamData, error) {
	params := ParamSearch(searchText)
	return c.GetTeams(ctx, params)
}

// SearchTeams searches teams by name with fuzzy matching (alias for SearchTeamsByText)
func (c *Client) SearchTeams(ctx context.Context, searchTerm string) ([]models.FootballAPITeamData, error) {
	return c.SearchTeamsByText(ctx, searchTerm)
}

// GetTeamsAdvanced fetches teams with multiple filter criteria
func (c *Client) GetTeamsAdvanced(ctx context.Context, options TeamSearchOptions) ([]models.FootballAPITeamData, error) {
	params := options.ToParams()
	return c.GetTeams(ctx, params)
}

// TeamSearchOptions represents advanced search options for teams
type TeamSearchOptions struct {
	ID      *int
	Name    string
	League  *int
	Season  *int
	Country string
	Code    string
	Venue   *int
	Search  string
}

// ToParams converts TeamSearchOptions to parameter map
func (o *TeamSearchOptions) ToParams() map[string]string {
	params := make(map[string]string)

	if o.ID != nil {
		params["id"] = strconv.Itoa(*o.ID)
	}
	if o.Name != "" {
		params["name"] = o.Name
	}
	if o.League != nil {
		params["league"] = strconv.Itoa(*o.League)
	}
	if o.Season != nil {
		params["season"] = strconv.Itoa(*o.Season)
	}
	if o.Country != "" {
		params["country"] = o.Country
	}
	if o.Code != "" {
		params["code"] = o.Code
	}
	if o.Venue != nil {
		params["venue"] = strconv.Itoa(*o.Venue)
	}
	if o.Search != "" {
		params["search"] = o.Search
	}

	return params
}

// GetTeamsForCountryAndSeason fetches teams for a specific country and season
func (c *Client) GetTeamsForCountryAndSeason(ctx context.Context, country string, season int) ([]models.FootballAPITeamData, error) {
	params := MergeParams(
		ParamCountry(country),
		ParamSeason(season),
	)
	return c.GetTeams(ctx, params)
}

// GetTeamsInLeagueForCountry fetches teams in a specific league for a country
func (c *Client) GetTeamsInLeagueForCountry(ctx context.Context, leagueID int, country string) ([]models.FootballAPITeamData, error) {
	params := MergeParams(
		map[string]string{"league": strconv.Itoa(leagueID)},
		ParamCountry(country),
	)
	return c.GetTeams(ctx, params)
}

// Convenience methods for common team searches

// GetTurkishTeams fetches all Turkish teams
func (c *Client) GetTurkishTeams(ctx context.Context) ([]models.FootballAPITeamData, error) {
	return c.GetTeamsByCountry(ctx, "Turkey")
}

// GetTurkishTeamsInSuperLig fetches Turkish teams in Super Lig (assuming league ID 203)
func (c *Client) GetTurkishTeamsInSuperLig(ctx context.Context, season int) ([]models.FootballAPITeamData, error) {
	// Note: This assumes Turkish Super Lig has league ID 203, adjust as needed
	return c.GetTeamsByLeagueAndSeason(ctx, 203, season)
}

// GetNationalTeams fetches national teams
func (c *Client) GetNationalTeams(ctx context.Context) ([]models.FootballAPITeamData, error) {
	// National teams typically don't have a league parameter, so we search for them differently
	// This might require a different approach depending on the API structure
	params := map[string]string{}
	teams, err := c.GetTeams(ctx, params)
	if err != nil {
		return nil, err
	}

	// Filter for national teams
	var nationalTeams []models.FootballAPITeamData
	for _, team := range teams {
		if team.Team.National {
			nationalTeams = append(nationalTeams, team)
		}
	}

	return nationalTeams, nil
}

// GetTeamsFoundedAfter fetches teams founded after a specific year
func (c *Client) GetTeamsFoundedAfter(ctx context.Context, year int, country string) ([]models.FootballAPITeamData, error) {
	teams, err := c.GetTeamsByCountry(ctx, country)
	if err != nil {
		return nil, err
	}

	// Filter teams founded after the specified year
	var filteredTeams []models.FootballAPITeamData
	for _, team := range teams {
		if team.Team.Founded >= year {
			filteredTeams = append(filteredTeams, team)
		}
	}

	return filteredTeams, nil
}

// GetTeamsWithVenueInfo fetches teams that have venue information
func (c *Client) GetTeamsWithVenueInfo(ctx context.Context, country string) ([]models.FootballAPITeamData, error) {
	teams, err := c.GetTeamsByCountry(ctx, country)
	if err != nil {
		return nil, err
	}

	// Filter teams with venue information
	var teamsWithVenue []models.FootballAPITeamData
	for _, team := range teams {
		if team.Venue.ID > 0 && team.Venue.Name != "" {
			teamsWithVenue = append(teamsWithVenue, team)
		}
	}

	return teamsWithVenue, nil
}

// Helper functions for building complex queries

// NewTeamSearchOptions creates a new TeamSearchOptions instance
func NewTeamSearchOptions() *TeamSearchOptions {
	return &TeamSearchOptions{}
}

// WithID sets the team ID filter
func (o *TeamSearchOptions) WithID(id int) *TeamSearchOptions {
	o.ID = &id
	return o
}

// WithName sets the name filter
func (o *TeamSearchOptions) WithName(name string) *TeamSearchOptions {
	o.Name = name
	return o
}

// WithLeague sets the league filter
func (o *TeamSearchOptions) WithLeague(leagueID int) *TeamSearchOptions {
	o.League = &leagueID
	return o
}

// WithSeason sets the season filter
func (o *TeamSearchOptions) WithSeason(season int) *TeamSearchOptions {
	o.Season = &season
	return o
}

// WithCountry sets the country filter
func (o *TeamSearchOptions) WithCountry(country string) *TeamSearchOptions {
	o.Country = country
	return o
}

// WithCode sets the country code filter
func (o *TeamSearchOptions) WithCode(code string) *TeamSearchOptions {
	o.Code = code
	return o
}

// WithVenue sets the venue filter
func (o *TeamSearchOptions) WithVenue(venueID int) *TeamSearchOptions {
	o.Venue = &venueID
	return o
}

// WithSearch sets the search term
func (o *TeamSearchOptions) WithSearch(search string) *TeamSearchOptions {
	o.Search = search
	return o
}

// Team statistics and information helpers

// GetTeamStatistics returns a summary of team information
func GetTeamStatistics(teamData models.FootballAPITeamData) map[string]any {
	return map[string]any{
		"id":             teamData.Team.ID,
		"name":           teamData.Team.Name,
		"code":           teamData.Team.Code,
		"country":        teamData.Team.Country,
		"founded":        teamData.Team.Founded,
		"is_national":    teamData.Team.National,
		"has_logo":       teamData.Team.Logo != "",
		"venue_id":       teamData.Venue.ID,
		"venue_name":     teamData.Venue.Name,
		"venue_city":     teamData.Venue.City,
		"venue_capacity": teamData.Venue.Capacity,
		"has_venue":      teamData.Venue.ID > 0,
	}
}

// IsClubTeam checks if the team is a club team (not national)
func IsClubTeam(teamData models.FootballAPITeamData) bool {
	return !teamData.Team.National
}

// HasVenue checks if the team has venue information
func HasVenue(teamData models.FootballAPITeamData) bool {
	return teamData.Venue.ID > 0 && teamData.Venue.Name != ""
}

// GetFoundedAge returns how many years ago the team was founded
func GetFoundedAge(teamData models.FootballAPITeamData, currentYear int) int {
	if teamData.Team.Founded > 0 && currentYear > teamData.Team.Founded {
		return currentYear - teamData.Team.Founded
	}
	return 0
}
