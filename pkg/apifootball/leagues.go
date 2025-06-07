package apifootball

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/iddaa-lens/core/pkg/models"
)

// League-related endpoints and functionality

// GetLeagues fetches leagues with various filter options
func (c *Client) GetLeagues(ctx context.Context, params map[string]string) ([]models.FootballAPILeagueData, error) {
	response, err := c.makeRequest(ctx, "/leagues", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get leagues: %w", err)
	}

	var leaguesData []models.FootballAPILeagueData
	if err := json.Unmarshal(response.Response, &leaguesData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal leagues response: %w", err)
	}

	return leaguesData, nil
}

// GetLeagueByID fetches a specific league by ID
func (c *Client) GetLeagueByID(ctx context.Context, leagueID int) (*models.FootballAPILeagueData, error) {
	params := ParamID(leagueID)
	leagues, err := c.GetLeagues(ctx, params)
	if err != nil {
		return nil, err
	}

	if len(leagues) == 0 {
		return nil, fmt.Errorf("league with ID %d not found", leagueID)
	}

	return &leagues[0], nil
}

// GetLeaguesByName searches leagues by name
func (c *Client) GetLeaguesByName(ctx context.Context, name string) ([]models.FootballAPILeagueData, error) {
	params := ParamName(name)
	return c.GetLeagues(ctx, params)
}

// GetLeaguesByCountry fetches leagues for a specific country
func (c *Client) GetLeaguesByCountry(ctx context.Context, country string) ([]models.FootballAPILeagueData, error) {
	params := ParamCountry(country)
	return c.GetLeagues(ctx, params)
}

// GetLeaguesByCode fetches leagues by country code
func (c *Client) GetLeaguesByCode(ctx context.Context, code string) ([]models.FootballAPILeagueData, error) {
	params := ParamCode(code)
	return c.GetLeagues(ctx, params)
}

// GetLeaguesBySeason fetches leagues for a specific season
func (c *Client) GetLeaguesBySeason(ctx context.Context, season int) ([]models.FootballAPILeagueData, error) {
	params := ParamSeason(season)
	return c.GetLeagues(ctx, params)
}

// GetLeaguesByTeam fetches leagues that contain a specific team
func (c *Client) GetLeaguesByTeam(ctx context.Context, teamID int) ([]models.FootballAPILeagueData, error) {
	params := ParamTeam(teamID)
	return c.GetLeagues(ctx, params)
}

// GetLeaguesByType fetches leagues by type (e.g., "League", "Cup")
func (c *Client) GetLeaguesByType(ctx context.Context, leagueType string) ([]models.FootballAPILeagueData, error) {
	params := ParamType(leagueType)
	return c.GetLeagues(ctx, params)
}

// GetCurrentLeagues fetches currently active leagues
func (c *Client) GetCurrentLeagues(ctx context.Context) ([]models.FootballAPILeagueData, error) {
	params := ParamCurrent(true)
	return c.GetLeagues(ctx, params)
}

// GetLastAddedLeagues fetches the most recently added leagues
func (c *Client) GetLastAddedLeagues(ctx context.Context, count int) ([]models.FootballAPILeagueData, error) {
	params := ParamLast(count)
	return c.GetLeagues(ctx, params)
}

// SearchLeagues searches leagues by name or country with fuzzy matching
func (c *Client) SearchLeagues(ctx context.Context, searchTerm string) ([]models.FootballAPILeagueData, error) {
	params := ParamSearch(searchTerm)
	return c.GetLeagues(ctx, params)
}

// GetLeaguesAdvanced fetches leagues with multiple filter criteria
func (c *Client) GetLeaguesAdvanced(ctx context.Context, options LeagueSearchOptions) ([]models.FootballAPILeagueData, error) {
	params := options.ToParams()
	return c.GetLeagues(ctx, params)
}

// LeagueSearchOptions represents advanced search options for leagues
type LeagueSearchOptions struct {
	ID      *int
	Name    string
	Country string
	Code    string
	Season  *int
	Team    *int
	Type    string
	Current *bool
	Search  string
	Last    *int
}

// ToParams converts LeagueSearchOptions to parameter map
func (o *LeagueSearchOptions) ToParams() map[string]string {
	params := make(map[string]string)

	if o.ID != nil {
		params["id"] = strconv.Itoa(*o.ID)
	}
	if o.Name != "" {
		params["name"] = o.Name
	}
	if o.Country != "" {
		params["country"] = o.Country
	}
	if o.Code != "" {
		params["code"] = o.Code
	}
	if o.Season != nil {
		params["season"] = strconv.Itoa(*o.Season)
	}
	if o.Team != nil {
		params["team"] = strconv.Itoa(*o.Team)
	}
	if o.Type != "" {
		params["type"] = o.Type
	}
	if o.Current != nil {
		params["current"] = strconv.FormatBool(*o.Current)
	}
	if o.Search != "" {
		params["search"] = o.Search
	}
	if o.Last != nil {
		params["last"] = strconv.Itoa(*o.Last)
	}

	return params
}

// GetLeaguesForCountryAndSeason fetches leagues for a specific country and season
func (c *Client) GetLeaguesForCountryAndSeason(ctx context.Context, country string, season int) ([]models.FootballAPILeagueData, error) {
	params := MergeParams(
		ParamCountry(country),
		ParamSeason(season),
	)
	return c.GetLeagues(ctx, params)
}

// GetCurrentLeaguesForCountry fetches current leagues for a specific country
func (c *Client) GetCurrentLeaguesForCountry(ctx context.Context, country string) ([]models.FootballAPILeagueData, error) {
	params := MergeParams(
		ParamCountry(country),
		ParamCurrent(true),
	)
	return c.GetLeagues(ctx, params)
}

// GetLeaguesByTypeAndCountry fetches leagues by type and country
func (c *Client) GetLeaguesByTypeAndCountry(ctx context.Context, leagueType, country string) ([]models.FootballAPILeagueData, error) {
	params := MergeParams(
		ParamType(leagueType),
		ParamCountry(country),
	)
	return c.GetLeagues(ctx, params)
}

// GetLeaguesWithTeamInSeason fetches leagues containing a specific team in a season
func (c *Client) GetLeaguesWithTeamInSeason(ctx context.Context, teamID, season int) ([]models.FootballAPILeagueData, error) {
	params := MergeParams(
		ParamTeam(teamID),
		ParamSeason(season),
	)
	return c.GetLeagues(ctx, params)
}

// Convenience methods for common league searches

// GetTurkishLeagues fetches all Turkish leagues
func (c *Client) GetTurkishLeagues(ctx context.Context) ([]models.FootballAPILeagueData, error) {
	return c.GetLeaguesByCountry(ctx, "Turkey")
}

// GetCurrentTurkishLeagues fetches current Turkish leagues
func (c *Client) GetCurrentTurkishLeagues(ctx context.Context) ([]models.FootballAPILeagueData, error) {
	return c.GetCurrentLeaguesForCountry(ctx, "Turkey")
}

// GetMajorEuropeanLeagues fetches major European leagues for current season
func (c *Client) GetMajorEuropeanLeagues(ctx context.Context) ([]models.FootballAPILeagueData, error) {
	majorCountries := []string{"England", "Spain", "Germany", "Italy", "France"}
	var allLeagues []models.FootballAPILeagueData

	for _, country := range majorCountries {
		leagues, err := c.GetCurrentLeaguesForCountry(ctx, country)
		if err != nil {
			return nil, fmt.Errorf("failed to get leagues for %s: %w", country, err)
		}
		allLeagues = append(allLeagues, leagues...)
	}

	return allLeagues, nil
}

// GetLeagueTypeOptions returns common league type options
func GetLeagueTypeOptions() []string {
	return []string{
		"League",
		"Cup",
		"Championship",
		"Playoff",
		"Qualification",
		"Friendly",
	}
}

// Helper functions for building complex queries

// NewLeagueSearchOptions creates a new LeagueSearchOptions instance
func NewLeagueSearchOptions() *LeagueSearchOptions {
	return &LeagueSearchOptions{}
}

// WithID sets the league ID filter
func (o *LeagueSearchOptions) WithID(id int) *LeagueSearchOptions {
	o.ID = &id
	return o
}

// WithName sets the name filter
func (o *LeagueSearchOptions) WithName(name string) *LeagueSearchOptions {
	o.Name = name
	return o
}

// WithCountry sets the country filter
func (o *LeagueSearchOptions) WithCountry(country string) *LeagueSearchOptions {
	o.Country = country
	return o
}

// WithCode sets the country code filter
func (o *LeagueSearchOptions) WithCode(code string) *LeagueSearchOptions {
	o.Code = code
	return o
}

// WithSeason sets the season filter
func (o *LeagueSearchOptions) WithSeason(season int) *LeagueSearchOptions {
	o.Season = &season
	return o
}

// WithTeam sets the team filter
func (o *LeagueSearchOptions) WithTeam(teamID int) *LeagueSearchOptions {
	o.Team = &teamID
	return o
}

// WithType sets the league type filter
func (o *LeagueSearchOptions) WithType(leagueType string) *LeagueSearchOptions {
	o.Type = leagueType
	return o
}

// WithCurrent sets the current season filter
func (o *LeagueSearchOptions) WithCurrent(current bool) *LeagueSearchOptions {
	o.Current = &current
	return o
}

// WithSearch sets the search term
func (o *LeagueSearchOptions) WithSearch(search string) *LeagueSearchOptions {
	o.Search = search
	return o
}

// WithLast sets the limit for recently added leagues
func (o *LeagueSearchOptions) WithLast(count int) *LeagueSearchOptions {
	o.Last = &count
	return o
}
