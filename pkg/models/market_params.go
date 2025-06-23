package models

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// MarketParams represents generic parameters for any market type
// Uses a simple array to store values that map to {0}, {1}, {2}, etc.
type MarketParams struct {
	// Values contains the parameter values in order
	// {0} maps to Values[0], {1} maps to Values[1], etc.
	Values []string `json:"values"`
}

// FormatMarketName replaces placeholders {0}, {1}, {2}, etc. with actual values
func FormatMarketName(template string, params MarketParams) string {
	result := template

	// Replace each placeholder with corresponding value
	for i, value := range params.Values {
		placeholder := fmt.Sprintf("{%d}", i)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// ExtractMarketParams extracts parameters from Iddaa's SpecialValue field
// The templates handle all formatting, we just need to extract the values
// Examples:
// - "5.5" -> ["5.5"]
// - "15-30:2.5" -> ["15", "30", "2.5"]
// - "1,2,3" -> ["1", "2", "3"]
func ExtractMarketParams(specialValue string) MarketParams {
	if specialValue == "" {
		return MarketParams{Values: []string{}}
	}

	params := MarketParams{Values: []string{}}

	// Split by common delimiters to extract all values
	// First split by colon to separate period from line values
	colonParts := strings.Split(specialValue, ":")

	for _, part := range colonParts {
		// Then split by dash for period ranges
		dashParts := strings.Split(part, "-")
		for _, dashPart := range dashParts {
			// Finally split by comma for multiple values
			commaParts := strings.Split(dashPart, ",")
			for _, value := range commaParts {
				trimmed := strings.TrimSpace(value)
				if trimmed != "" {
					params.Values = append(params.Values, formatNumber(trimmed))
				}
			}
		}
	}

	return params
}

// formatNumber formats numeric strings consistently
// Removes unnecessary decimal zeros: "2.0" -> "2", "2.5" -> "2.5"
func formatNumber(value string) string {
	// Try to parse as float
	if num, err := strconv.ParseFloat(value, 64); err == nil {
		// Check if it's a whole number
		if num == float64(int(num)) {
			return fmt.Sprintf("%.0f", num)
		}
		// Otherwise keep one decimal place
		return fmt.Sprintf("%.1f", num)
	}
	// Return as-is if not a number
	return value
}

// GetPlaceholderCount returns the number of placeholders in a template string
func GetPlaceholderCount(template string) int {
	// Find all {n} patterns
	re := regexp.MustCompile(`\{(\d+)\}`)
	matches := re.FindAllStringSubmatch(template, -1)

	maxIndex := -1
	for _, match := range matches {
		if len(match) > 1 {
			if index, err := strconv.Atoi(match[1]); err == nil && index > maxIndex {
				maxIndex = index
			}
		}
	}

	return maxIndex + 1
}
