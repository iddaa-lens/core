package utils

import (
	"regexp"
	"strings"
	"unicode"
)

// TeamNameNormalizer handles normalization of team names for better matching
type TeamNameNormalizer struct {
	commonSuffixes []string
	commonPrefixes []string
	replacements   map[string]string
	spaceRegex     *regexp.Regexp
}

// NewTeamNameNormalizer creates a new team name normalizer
func NewTeamNameNormalizer() *TeamNameNormalizer {
	return &TeamNameNormalizer{
		commonSuffixes: []string{
			// Football club suffixes
			"FC", "SC", "AC", "BC", "SK", "JK", "FK", "GK", "TK", "BK", "IK",
			"CF", "IF", "AIK", "IFK", "HJK", "VfB", "VfL", "SV", "TSV",

			// English suffixes
			"United", "City", "Town", "Rovers", "Wanderers", "Athletic",
			"Albion", "Villa", "County", "Borough", "Rangers", "Hotspur",

			// Generic suffixes
			"Sports", "Club", "Team", "Football", "Soccer", "Futbol",

			// Turkish suffixes
			"Spor", "Kulübü", "Kulübu", "Kulubu", "Spor Kulübü", "Spor Kulubu",
			"SK", "FK", "GS", "FB", "BJK", "TS", "AS", "KS", "BS", "ES",

			// Other languages
			"Deportivo", "Sporting", "Olympique", "Real", "Atletico",
			"Internacional", "Nacional", "Central", "Oriental",
		},
		commonPrefixes: []string{
			"FC", "AC", "SC", "CF", "Club", "Real", "Athletic", "Sporting",
			"Deportivo", "Olympique", "AS", "US", "AJ", "NK", "HNK", "GNK",
		},
		replacements: map[string]string{
			"&":             "and",
			"Sankt":         "St",
			"Saint":         "St",
			"Futbol Kulübü": "",
			"Spor Kulübü":   "",
			"Kulübü":        "",
			"Kulübu":        "",
			"Kulubu":        "",
			"Spor":          "",

			// Turkish character normalization
			"ç": "c", "Ç": "C",
			"ğ": "g", "Ğ": "G",
			"ı": "i", "I": "I",
			"ö": "o", "Ö": "O",
			"ş": "s", "Ş": "S",
			"ü": "u", "Ü": "U",
			"â": "a", "Â": "A",
			"î": "i", "Î": "I",
			"û": "u", "Û": "U",

			// Common abbreviations
			"Fußball": "Football",
			"Calcio":  "Football",
			"Futebol": "Football",
		},
		spaceRegex: regexp.MustCompile(`\s+`),
	}
}

// Normalize normalizes a team name for better matching
func (n *TeamNameNormalizer) Normalize(teamName string) string {
	if teamName == "" {
		return ""
	}

	normalized := strings.TrimSpace(teamName)

	// Apply character replacements (including Turkish characters)
	for old, new := range n.replacements {
		normalized = strings.ReplaceAll(normalized, old, new)
	}

	// Remove common prefixes
	normalized = n.removePrefixes(normalized)

	// Remove common suffixes
	normalized = n.removeSuffixes(normalized)

	// Clean up extra spaces and normalize case
	normalized = n.spaceRegex.ReplaceAllString(normalized, " ")
	normalized = strings.TrimSpace(normalized)

	return normalized
}

// NormalizeCountry normalizes country names for matching
func (n *TeamNameNormalizer) NormalizeCountry(country string) string {
	if country == "" {
		return ""
	}

	normalized := strings.TrimSpace(country)

	// Apply character replacements
	for old, new := range n.replacements {
		if len(old) == 1 && len(new) == 1 { // Only character replacements
			normalized = strings.ReplaceAll(normalized, old, new)
		}
	}

	return normalized
}

// removePrefixes removes common team name prefixes
func (n *TeamNameNormalizer) removePrefixes(name string) string {
	for _, prefix := range n.commonPrefixes {
		patterns := []string{
			prefix + " ",                  // "FC "
			strings.ToLower(prefix) + " ", // "fc "
			strings.ToUpper(prefix) + " ", // "FC "
		}

		for _, pattern := range patterns {
			if strings.HasPrefix(name, pattern) {
				return strings.TrimSpace(strings.TrimPrefix(name, pattern))
			}
		}
	}
	return name
}

// removeSuffixes removes common team name suffixes
func (n *TeamNameNormalizer) removeSuffixes(name string) string {
	for _, suffix := range n.commonSuffixes {
		patterns := []string{
			" " + suffix,                  // " FC"
			" " + strings.ToLower(suffix), // " fc"
			" " + strings.ToUpper(suffix), // " FC"
		}

		for _, pattern := range patterns {
			if strings.HasSuffix(name, pattern) {
				return strings.TrimSpace(strings.TrimSuffix(name, pattern))
			}
		}
	}
	return name
}

// ExtractKeywords extracts meaningful keywords from a team name
func (n *TeamNameNormalizer) ExtractKeywords(teamName string) []string {
	// Normalize first
	normalized := n.Normalize(teamName)

	// Common stop words to ignore
	stopWords := map[string]bool{
		"the": true, "of": true, "and": true, "in": true, "at": true,
		"de": true, "del": true, "la": true, "le": true, "les": true,
		"el": true, "los": true, "das": true, "der": true, "die": true,
		"een": true, "het": true, "van": true, "von": true, "zu": true,
	}

	words := strings.Fields(strings.ToLower(normalized))
	var keywords []string

	for _, word := range words {
		// Clean word of punctuation
		cleaned := n.cleanWord(word)

		// Include if it's meaningful (length > 2 and not a stop word)
		if len(cleaned) > 2 && !stopWords[cleaned] {
			keywords = append(keywords, cleaned)
		}
	}

	return keywords
}

// cleanWord removes punctuation and non-letter characters from a word
func (n *TeamNameNormalizer) cleanWord(word string) string {
	var result strings.Builder
	for _, r := range word {
		if unicode.IsLetter(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// CompareNormalized compares two normalized team names and returns similarity score
func (n *TeamNameNormalizer) CompareNormalized(name1, name2 string) float64 {
	norm1 := n.Normalize(name1)
	norm2 := n.Normalize(name2)

	// Exact match
	if strings.EqualFold(norm1, norm2) {
		return 1.0
	}

	// Contains match
	if strings.Contains(strings.ToLower(norm1), strings.ToLower(norm2)) ||
		strings.Contains(strings.ToLower(norm2), strings.ToLower(norm1)) {
		return 0.9
	}

	// Keyword-based similarity
	keywords1 := n.ExtractKeywords(norm1)
	keywords2 := n.ExtractKeywords(norm2)

	if len(keywords1) == 0 || len(keywords2) == 0 {
		return 0.0
	}

	// Count common keywords
	common := 0
	keywordMap := make(map[string]bool)
	for _, kw := range keywords1 {
		keywordMap[strings.ToLower(kw)] = true
	}

	for _, kw := range keywords2 {
		if keywordMap[strings.ToLower(kw)] {
			common++
		}
	}

	// Calculate Jaccard similarity
	union := len(keywords1) + len(keywords2) - common
	if union == 0 {
		return 0.0
	}

	return float64(common) / float64(union)
}

// GetNormalizedVariations returns multiple normalized variations of a team name
func (n *TeamNameNormalizer) GetNormalizedVariations(teamName string) []string {
	if teamName == "" {
		return nil
	}

	variations := []string{
		teamName,              // Original
		n.Normalize(teamName), // Fully normalized
	}

	// Add variation without suffixes only
	withoutSuffixes := n.removeSuffixes(teamName)
	if withoutSuffixes != teamName {
		variations = append(variations, withoutSuffixes)
	}

	// Add variation without prefixes only
	withoutPrefixes := n.removePrefixes(teamName)
	if withoutPrefixes != teamName {
		variations = append(variations, withoutPrefixes)
	}

	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, v := range variations {
		normalized := strings.TrimSpace(v)
		if normalized != "" && !seen[normalized] {
			seen[normalized] = true
			unique = append(unique, normalized)
		}
	}

	return unique
}
