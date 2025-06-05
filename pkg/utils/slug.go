package utils

import (
	"github.com/gosimple/slug"
)

// NormalizeSlug creates a URL-friendly slug using the gosimple/slug library
// This handles all Unicode characters including Turkish, European, and other languages
func NormalizeSlug(text string) string {
	if text == "" {
		return ""
	}

	// Use gosimple/slug which handles all international characters properly
	return slug.Make(text)
}

// GenerateEventSlug creates a slug for an event from team names and external ID
func GenerateEventSlug(homeTeam, awayTeam, externalID string) string {
	if homeTeam == "" {
		homeTeam = "team"
	}
	if awayTeam == "" {
		awayTeam = "team"
	}
	if externalID == "" {
		externalID = "event"
	}

	text := homeTeam + " vs " + awayTeam + " " + externalID
	return NormalizeSlug(text)
}

// GenerateTeamSlug creates a slug for a team name
func GenerateTeamSlug(teamName string) string {
	if teamName == "" {
		return "team"
	}
	return NormalizeSlug(teamName)
}

// GenerateLeagueSlug creates a slug for a league name and country
func GenerateLeagueSlug(leagueName, country string) string {
	if leagueName == "" {
		leagueName = "league"
	}

	text := leagueName
	if country != "" {
		text += " " + country
	}

	return NormalizeSlug(text)
}

// GenerateMarketTypeSlug creates a slug for a market type
func GenerateMarketTypeSlug(code, name string) string {
	if code == "" && name == "" {
		return "market"
	}

	text := code
	if name != "" {
		if text != "" {
			text += " "
		}
		text += name
	}

	return NormalizeSlug(text)
}
