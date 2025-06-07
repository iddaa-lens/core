package services

import (
	"context"
	"fmt"
	"strings"
)

// TranslationMappings contains comprehensive Turkish to English mappings
type TranslationMappings struct {
	Countries map[string]string
	Teams     map[string]string
	Leagues   map[string]string
	Keywords  map[string]string
}

// NewTranslationMappings creates comprehensive translation mappings
func NewTranslationMappings() *TranslationMappings {
	return &TranslationMappings{
		Countries: map[string]string{
			// European countries
			"türkiye":         "Turkey",
			"tr":              "Turkey",
			"ingiltere":       "England",
			"gb":              "England",
			"gb-eng":          "England",
			"iskocya":         "Scotland",
			"galler":          "Wales",
			"kuzey irlanda":   "Northern Ireland",
			"irlanda":         "Ireland",
			"ispanya":         "Spain",
			"es":              "Spain",
			"italya":          "Italy",
			"it":              "Italy",
			"almanya":         "Germany",
			"de":              "Germany",
			"fransa":          "France",
			"fr":              "France",
			"hollanda":        "Netherlands",
			"nl":              "Netherlands",
			"belcika":         "Belgium",
			"be":              "Belgium",
			"portekiz":        "Portugal",
			"pt":              "Portugal",
			"rusya":           "Russia",
			"ru":              "Russia",
			"ukrayna":         "Ukraine",
			"ua":              "Ukraine",
			"polonya":         "Poland",
			"pl":              "Poland",
			"cek cumhuriyeti": "Czech Republic",
			"cz":              "Czech Republic",
			"macaristan":      "Hungary",
			"hu":              "Hungary",
			"romanya":         "Romania",
			"ro":              "Romania",
			"bulgaristan":     "Bulgaria",
			"bg":              "Bulgaria",
			"hirvatistan":     "Croatia",
			"hr":              "Croatia",
			"slovenya":        "Slovenia",
			"si":              "Slovenia",
			"slovakya":        "Slovakia",
			"sk":              "Slovakia",
			"avusturya":       "Austria",
			"at":              "Austria",
			"isvicre":         "Switzerland",
			"ch":              "Switzerland",
			"norves":          "Norway",
			"no":              "Norway",
			"isvec":           "Sweden",
			"se":              "Sweden",
			"danimarka":       "Denmark",
			"dk":              "Denmark",
			"finlandiya":      "Finland",
			"fi":              "Finland",
			"yunanistan":      "Greece",
			"gr":              "Greece",
			"serbistan":       "Serbia",
			"rs":              "Serbia",
			"bosna hersek":    "Bosnia and Herzegovina",
			"ba":              "Bosnia and Herzegovina",
			"karadag":         "Montenegro",
			"me":              "Montenegro",
			"kuzey makedonya": "North Macedonia",
			"mk":              "North Macedonia",
			"arnavutluk":      "Albania",
			"al":              "Albania",
			"kosova":          "Kosovo",
			"xk":              "Kosovo",

			// Americas
			"brezilya":  "Brazil",
			"br":        "Brazil",
			"arjantin":  "Argentina",
			"ar":        "Argentina",
			"kolombiya": "Colombia",
			"co":        "Colombia",
			"sili":      "Chile",
			"cl":        "Chile",
			"meksika":   "Mexico",
			"mx":        "Mexico",
			"amerika":   "United States",
			"abd":       "United States",
			"us":        "United States",
			"kanada":    "Canada",
			"ca":        "Canada",
			"uruguay":   "Uruguay",
			"uy":        "Uruguay",
			"peru":      "Peru",
			"pe":        "Peru",
			"ekvador":   "Ecuador",
			"ec":        "Ecuador",
			"venezuela": "Venezuela",
			"ve":        "Venezuela",
			"paraguay":  "Paraguay",
			"py":        "Paraguay",
			"bolivya":   "Bolivia",
			"bo":        "Bolivia",

			// Asia & Oceania
			"japonya":         "Japan",
			"jp":              "Japan",
			"guney kore":      "South Korea",
			"kr":              "South Korea",
			"cin":             "China",
			"cn":              "China",
			"hindistan":       "India",
			"in":              "India",
			"avustralya":      "Australia",
			"au":              "Australia",
			"yeni zelanda":    "New Zealand",
			"nz":              "New Zealand",
			"suudi arabistan": "Saudi Arabia",
			"sa":              "Saudi Arabia",
			"iran":            "Iran",
			"ir":              "Iran",
			"irak":            "Iraq",
			"iq":              "Iraq",
			"israil":          "Israel",
			"il":              "Israel",
			"tayland":         "Thailand",
			"th":              "Thailand",
			"vietnam":         "Vietnam",
			"vn":              "Vietnam",
			"malezya":         "Malaysia",
			"my":              "Malaysia",
			"singapur":        "Singapore",
			"sg":              "Singapore",
			"filipinler":      "Philippines",
			"ph":              "Philippines",
			"endonezya":       "Indonesia",
			"id":              "Indonesia",

			// Africa
			"misir":          "Egypt",
			"eg":             "Egypt",
			"fas":            "Morocco",
			"ma":             "Morocco",
			"cezayir":        "Algeria",
			"dz":             "Algeria",
			"tunus":          "Tunisia",
			"tn":             "Tunisia",
			"libya":          "Libya",
			"ly":             "Libya",
			"nijerya":        "Nigeria",
			"ng":             "Nigeria",
			"gana":           "Ghana",
			"gh":             "Ghana",
			"kamerun":        "Cameroon",
			"cm":             "Cameroon",
			"senegal":        "Senegal",
			"sn":             "Senegal",
			"guney afrika":   "South Africa",
			"za":             "South Africa",
			"kenya":          "Kenya",
			"ke":             "Kenya",
			"fildisi sahili": "Ivory Coast",
			"ci":             "Ivory Coast",

			// International
			"int":           "World",
			"international": "World",
			"dunya":         "World",
		},

		Teams: map[string]string{
			// Turkish Super League teams
			"galatasaray":               "Galatasaray",
			"galatasaray sk":            "Galatasaray",
			"galatasaray spor kulubu":   "Galatasaray",
			"fenerbahce":                "Fenerbahce",
			"fenerbahce sk":             "Fenerbahce",
			"fenerbahce spor kulubu":    "Fenerbahce",
			"besiktas":                  "Besiktas",
			"besiktas jk":               "Besiktas",
			"besiktas jimnastik kulubu": "Besiktas",
			"trabzonspor":               "Trabzonspor",
			"trabzonspor kulubu":        "Trabzonspor",
			"basaksehir":                "Istanbul Basaksehir",
			"istanbul basaksehir":       "Istanbul Basaksehir",
			"istanbul basaksehir fk":    "Istanbul Basaksehir",
			"antalyaspor":               "Antalyaspor",
			"kayserispor":               "Kayserispor",
			"konyaspor":                 "Konyaspor",
			"sivasspor":                 "Sivasspor",
			"alanyaspor":                "Alanyaspor",
			"gaziantep fk":              "Gaziantep FK",
			"gaziantepspor":             "Gaziantep FK",
			"hatayspor":                 "Hatayspor",
			"adana demirspor":           "Adana Demirspor",
			"kasimpasa":                 "Kasimpasa",
			"fatih karagumruk":          "Fatih Karagumruk",
			"umraniyespor":              "Umraniyespor",
			"istanbulspor":              "Istanbulspor",
			"pendikspor":                "Pendikspor",

			// Major European teams
			"real madrid":         "Real Madrid",
			"barcelona":           "Barcelona",
			"fc barcelona":        "Barcelona",
			"atletico madrid":     "Atletico Madrid",
			"manchester united":   "Manchester United",
			"manchester city":     "Manchester City",
			"liverpool":           "Liverpool",
			"arsenal":             "Arsenal",
			"chelsea":             "Chelsea",
			"tottenham":           "Tottenham",
			"bayern munich":       "Bayern Munich",
			"borussia dortmund":   "Borussia Dortmund",
			"juventus":            "Juventus",
			"ac milan":            "AC Milan",
			"inter milan":         "Inter Milan",
			"napoli":              "Napoli",
			"as roma":             "AS Roma",
			"paris saint germain": "Paris Saint-Germain",
			"psg":                 "Paris Saint-Germain",
			"olympique marseille": "Marseille",
			"olympique lyon":      "Lyon",
			"ajax":                "Ajax",
			"psv eindhoven":       "PSV",
			"feyenoord":           "Feyenoord",
			"benfica":             "Benfica",
			"porto":               "Porto",
			"sporting":            "Sporting CP",
		},

		Leagues: map[string]string{
			// Turkish leagues
			"turkiye super lig":     "Super Lig",
			"super lig":             "Super Lig",
			"trendyol super lig":    "Super Lig",
			"turkiye 1 lig":         "1. Lig",
			"1 lig":                 "1. Lig",
			"tff 1 lig":             "1. Lig",
			"turkiye 2 lig":         "2. Lig",
			"2 lig":                 "2. Lig",
			"tff 2 lig":             "2. Lig",
			"turkiye 3 lig":         "3. Lig",
			"3 lig":                 "3. Lig",
			"tff 3 lig":             "3. Lig",
			"turkiye kupasi":        "Turkish Cup",
			"turkiye kupa":          "Turkish Cup",
			"ziraat turkiye kupasi": "Turkish Cup",

			// Major European leagues
			"ingiltere premier lig":  "Premier League",
			"premier lig":            "Premier League",
			"premier league":         "Premier League",
			"english premier league": "Premier League",
			"ingiltere championship": "Championship",
			"championship":           "Championship",
			"ingiltere lig 1":        "League One",
			"league one":             "League One",
			"ingiltere lig 2":        "League Two",
			"league two":             "League Two",
			"ingiltere fa kupa":      "FA Cup",
			"fa cup":                 "FA Cup",
			"ingiltere lig kupa":     "EFL Cup",
			"efl cup":                "EFL Cup",
			"carabao cup":            "EFL Cup",

			"ispanya la liga":  "La Liga",
			"la liga":          "La Liga",
			"primera division": "La Liga",
			"ispanya 1 lig":    "La Liga",
			"ispanya 2 lig":    "Segunda Division",
			"segunda division": "Segunda Division",
			"ispanya kupa":     "Copa del Rey",
			"copa del rey":     "Copa del Rey",

			"italya serie a": "Serie A",
			"serie a":        "Serie A",
			"italya 1 lig":   "Serie A",
			"italya serie b": "Serie B",
			"serie b":        "Serie B",
			"italya 2 lig":   "Serie B",
			"italya kupa":    "Coppa Italia",
			"coppa italia":   "Coppa Italia",

			"almanya bundesliga":   "Bundesliga",
			"bundesliga":           "Bundesliga",
			"almanya 1 lig":        "Bundesliga",
			"almanya 2 bundesliga": "2. Bundesliga",
			"2 bundesliga":         "2. Bundesliga",
			"almanya 2 lig":        "2. Bundesliga",
			"almanya 3 lig":        "3. Liga",
			"3 liga":               "3. Liga",
			"almanya kupa":         "DFB Pokal",
			"dfb pokal":            "DFB Pokal",

			"fransa ligue 1":  "Ligue 1",
			"ligue 1":         "Ligue 1",
			"fransa 1 lig":    "Ligue 1",
			"fransa ligue 2":  "Ligue 2",
			"ligue 2":         "Ligue 2",
			"fransa 2 lig":    "Ligue 2",
			"fransa kupa":     "Coupe de France",
			"coupe de france": "Coupe de France",

			"hollanda eredivisie": "Eredivisie",
			"eredivisie":          "Eredivisie",
			"hollanda 1 lig":      "Eredivisie",
			"hollanda kupa":       "KNVB Cup",
			"knvb cup":            "KNVB Cup",

			"portekiz primeira liga": "Primeira Liga",
			"primeira liga":          "Primeira Liga",
			"portekiz 1 lig":         "Primeira Liga",
			"portekiz kupa":          "Taca de Portugal",
			"taca de portugal":       "Taca de Portugal",

			"belcika pro lig": "Pro League",
			"pro league":      "Pro League",
			"belcika 1 lig":   "Pro League",
			"belcika kupa":    "Belgian Cup",

			// International competitions
			"uefa sampiyonlar ligi":  "Champions League",
			"sampiyonlar ligi":       "Champions League",
			"champions league":       "Champions League",
			"uefa champions league":  "Champions League",
			"uefa avrupa ligi":       "Europa League",
			"avrupa ligi":            "Europa League",
			"europa league":          "Europa League",
			"uefa europa league":     "Europa League",
			"uefa konferans ligi":    "Conference League",
			"konferans ligi":         "Conference League",
			"conference league":      "Conference League",
			"uefa conference league": "Conference League",
			"uluslar ligi":           "Nations League",
			"nations league":         "Nations League",
			"uefa nations league":    "Nations League",

			"fifa dunya kupasi":        "World Cup",
			"dunya kupasi":             "World Cup",
			"world cup":                "World Cup",
			"fifa world cup":           "World Cup",
			"uefa avrupa sampiyonligi": "European Championship",
			"avrupa sampiyonligi":      "European Championship",
			"euro":                     "European Championship",
			"uefa euro":                "European Championship",
			"european championship":    "European Championship",

			// Other major leagues
			"brezilya serie a":          "Serie A",
			"brasileirao":               "Serie A",
			"arjantin primera division": "Primera Division",
			"kolombiya primera a":       "Primera A",
			"abd major league soccer":   "MLS",
			"mls":                       "MLS",
			"major league soccer":       "MLS",
		},

		Keywords: map[string]string{
			// Generic football terms
			"lig":          "League",
			"ligi":         "League",
			"ligua":        "League",
			"kupa":         "Cup",
			"kupasi":       "Cup",
			"sampiyonlugu": "Championship",
			"sampiyonligi": "Championship",
			"turnuvasi":    "Tournament",
			"turnuva":      "Tournament",
			"futbol":       "Football",
			"spor":         "Sport",
			"kulubu":       "Club",
			"kulübü":       "Club",
			"takimi":       "Team",
			"takim":        "Team",

			// Positional/tier words
			"super":    "Super",
			"premier":  "Premier",
			"birinci":  "First",
			"ikinci":   "Second",
			"ucuncu":   "Third",
			"dorduncu": "Fourth",
			"besinci":  "Fifth",
			"1":        "First",
			"2":        "Second",
			"3":        "Third",
			"4":        "Fourth",
			"5":        "Fifth",

			// Common words
			"uluslararasi": "International",
			"ulusal":       "National",
			"milli":        "National",
			"gencler":      "Youth",
			"genç":         "Young",
			"yari":         "Semi",
			"final":        "Final",
			"grup":         "Group",
			"eleme":        "Qualifying",
			"elemeleri":    "Qualifiers",
			"baraj":        "Playoff",
			"play off":     "Playoff",

			// Seasonal/time words
			"sezon": "Season",
			"yil":   "Year",
			"donem": "Period",
		},
	}
}

// EnhancedTranslator combines AI translation with comprehensive fallback mappings
type EnhancedTranslator struct {
	aiTranslator *AITranslationService
	mappings     *TranslationMappings
}

// NewEnhancedTranslator creates a new enhanced translator
func NewEnhancedTranslator(openaiKey string) *EnhancedTranslator {
	var aiTranslator *AITranslationService
	if openaiKey != "" {
		aiTranslator = NewAITranslationService(openaiKey)
	}

	return &EnhancedTranslator{
		aiTranslator: aiTranslator,
		mappings:     NewTranslationMappings(),
	}
}

// TranslateTeamName translates a Turkish team name to English with multiple fallback strategies
func (e *EnhancedTranslator) TranslateTeamName(ctx context.Context, turkishName, country string) (string, error) {
	if turkishName == "" {
		return "", fmt.Errorf("team name cannot be empty")
	}

	// Strategy 1: Check static team mappings first (highest confidence)
	normalized := e.normalizeForLookup(turkishName)
	if englishName, exists := e.mappings.Teams[normalized]; exists {
		return englishName, nil
	}

	// Strategy 2: Try AI translation if available
	if e.aiTranslator != nil {
		if translations, err := e.aiTranslator.TranslateTeamName(ctx, turkishName, country); err == nil && len(translations) > 0 {
			return translations[0], nil // Use first (best) translation
		}
	}

	// Strategy 3: Keyword-based translation
	keywordTranslated := e.translateUsingKeywords(turkishName)
	if keywordTranslated != turkishName {
		return keywordTranslated, nil
	}

	// Strategy 4: Return cleaned up version
	return e.cleanupTeamName(turkishName), nil
}

// TranslateCountryName translates a Turkish country name to English
func (e *EnhancedTranslator) TranslateCountryName(countryName string) string {
	if countryName == "" {
		return ""
	}

	normalized := e.normalizeForLookup(countryName)
	if englishName, exists := e.mappings.Countries[normalized]; exists {
		return englishName
	}

	return countryName
}

// TranslateLeagueName translates a Turkish league name to English
func (e *EnhancedTranslator) TranslateLeagueName(ctx context.Context, leagueName, country string) (string, error) {
	if leagueName == "" {
		return "", fmt.Errorf("league name cannot be empty")
	}

	// Strategy 1: Check static league mappings first
	normalized := e.normalizeForLookup(leagueName)
	if englishName, exists := e.mappings.Leagues[normalized]; exists {
		return englishName, nil
	}

	// Strategy 2: Try AI translation if available
	if e.aiTranslator != nil {
		if translations, err := e.aiTranslator.TranslateLeagueName(ctx, leagueName, country); err == nil && len(translations) > 0 {
			return translations[0], nil
		}
	}

	// Strategy 3: Keyword-based translation
	keywordTranslated := e.translateUsingKeywords(leagueName)
	if keywordTranslated != leagueName {
		return keywordTranslated, nil
	}

	// Strategy 4: Return cleaned up version
	return e.cleanupLeagueName(leagueName), nil
}

// normalizeForLookup normalizes text for dictionary lookup
func (e *EnhancedTranslator) normalizeForLookup(text string) string {
	normalized := strings.ToLower(text)

	// Turkish character normalization
	replacements := map[string]string{
		"ç": "c", "ğ": "g", "ı": "i", "ö": "o", "ş": "s", "ü": "u",
	}

	for turkish, latin := range replacements {
		normalized = strings.ReplaceAll(normalized, turkish, latin)
	}

	// Remove extra spaces
	normalized = strings.Join(strings.Fields(normalized), " ")

	return strings.TrimSpace(normalized)
}

// translateUsingKeywords translates text using keyword mappings
func (e *EnhancedTranslator) translateUsingKeywords(text string) string {
	words := strings.Fields(text)
	var translatedWords []string

	for _, word := range words {
		normalized := e.normalizeForLookup(word)
		if englishWord, exists := e.mappings.Keywords[normalized]; exists {
			translatedWords = append(translatedWords, englishWord)
		} else {
			// Keep original word but clean it up
			cleaned := e.cleanupWord(word)
			translatedWords = append(translatedWords, cleaned)
		}
	}

	return strings.Join(translatedWords, " ")
}

// cleanupTeamName cleans up a team name by removing Turkish characters
func (e *EnhancedTranslator) cleanupTeamName(teamName string) string {
	cleaned := teamName

	// Turkish character replacements
	replacements := map[string]string{
		"ç": "c", "Ç": "C",
		"ğ": "g", "Ğ": "G",
		"ı": "i", "I": "I",
		"ö": "o", "Ö": "O",
		"ş": "s", "Ş": "S",
		"ü": "u", "Ü": "U",
	}

	for turkish, latin := range replacements {
		cleaned = strings.ReplaceAll(cleaned, turkish, latin)
	}

	return strings.TrimSpace(cleaned)
}

// cleanupLeagueName cleans up a league name
func (e *EnhancedTranslator) cleanupLeagueName(leagueName string) string {
	return e.cleanupTeamName(leagueName) // Same logic for now
}

// cleanupWord cleans up a single word
func (e *EnhancedTranslator) cleanupWord(word string) string {
	return e.cleanupTeamName(word) // Same logic for now
}

// GetTeamNameVariations returns multiple English variations of a Turkish team name
func (e *EnhancedTranslator) GetTeamNameVariations(ctx context.Context, turkishName, country string) ([]string, error) {
	variations := make([]string, 0)
	seen := make(map[string]bool)

	// Add main translation
	if mainTranslation, err := e.TranslateTeamName(ctx, turkishName, country); err == nil && mainTranslation != "" {
		if !seen[mainTranslation] {
			variations = append(variations, mainTranslation)
			seen[mainTranslation] = true
		}
	}

	// Add AI variations if available
	if e.aiTranslator != nil {
		if aiTranslations, err := e.aiTranslator.TranslateLeagueName(ctx, turkishName, country); err == nil {
			for _, translation := range aiTranslations {
				if !seen[translation] {
					variations = append(variations, translation)
					seen[translation] = true
				}
			}
		}
	}

	// Add keyword translation
	keywordTranslation := e.translateUsingKeywords(turkishName)
	if !seen[keywordTranslation] {
		variations = append(variations, keywordTranslation)
		seen[keywordTranslation] = true
	}

	// Add cleaned up original
	cleaned := e.cleanupTeamName(turkishName)
	if !seen[cleaned] {
		variations = append(variations, cleaned)
		seen[cleaned] = true
	}

	return variations, nil
}
