package models

type IddaaAPIResponse[T any] struct {
	IsSuccess bool   `json:"isSuccess"`
	Data      []T    `json:"data"`
	Message   string `json:"message"`
}

type IddaaEventsResponse struct {
	IsSuccess bool             `json:"isSuccess"`
	Data      *IddaaEventsData `json:"data"`
	Message   string           `json:"message"`
}

type IddaaEventsData struct {
	IsDiff  bool         `json:"isdiff"`
	Version int64        `json:"version"`
	Events  []IddaaEvent `json:"events"`
}

type IddaaEvent struct {
	ID            int           `json:"i"`
	BulletinID    int           `json:"bri"`
	Version       int           `json:"v"`
	HomeTeam      string        `json:"hn"`
	AwayTeam      string        `json:"an"`
	SportID       int           `json:"sid"`
	Status        int           `json:"s"`
	BetProgram    int           `json:"bp"`
	IsLive        bool          `json:"il"`
	MBC           int           `json:"mbc"`
	HasKingOdd    bool          `json:"kOdd"`
	Markets       []IddaaMarket `json:"m"`
	CompetitionID int           `json:"ci"`
	OddsCount     int           `json:"oc"`
	Date          int64         `json:"d"`
	HasCombine    bool          `json:"hc"`
}

type IddaaMarket struct {
	ID           int            `json:"i"`
	Type         int            `json:"t"`
	SubType      int            `json:"st"`
	Version      int            `json:"v"`
	Status       int            `json:"s"`
	MBC          int            `json:"mbc"`
	SpecialValue string         `json:"sov,omitempty"`
	Outcomes     []IddaaOutcome `json:"o"`
}

type IddaaOutcome struct {
	Number      int     `json:"no"`
	Odds        float64 `json:"odd"`
	WinningOdds float64 `json:"wodd"`
	Name        string  `json:"n"`
}

type IddaaOdds struct {
	EventID    int                    `json:"ei"`
	MarketType string                 `json:"mt"`
	Outcomes   map[string]interface{} `json:"outcomes"`
	Timestamp  string                 `json:"ts"`
}

type IddaaSportInfo struct {
	SportID           int  `json:"i"`  // Sport ID
	LiveCount         int  `json:"lc"` // Live events count
	UpcomingCount     int  `json:"uc"` // Upcoming events count
	EventsCount       int  `json:"ec"` // Total events count
	OddsCount         int  `json:"oc"` // Odds count
	HasResults        bool `json:"hr"` // Has results
	HasKingOdd        bool `json:"hk"` // Has king odd
	HasDigitalContent bool `json:"hd"` // Has digital content.
}

// IddaaMarketConfigResponse represents the response from get_market_config endpoint
type IddaaMarketConfigResponse struct {
	IsSuccess bool                  `json:"isSuccess"`
	Data      IddaaMarketConfigData `json:"data"`
	Message   string                `json:"message"`
}

// IddaaMarketConfigData contains the market configurations
type IddaaMarketConfigData struct {
	Markets map[string]IddaaMarketConfig `json:"m"`
}

// IddaaMarketConfig represents a single market configuration
type IddaaMarketConfig struct {
	ID             int    `json:"i"`    // Market ID
	Name           string `json:"n"`    // Market name (Turkish)
	Description    string `json:"d"`    // Market description (Turkish)
	IsLive         bool   `json:"il"`   // Is live market
	MarketType     int    `json:"mt"`   // Market type
	MinMarketValue int    `json:"mmdv"` // Min market default value
	MaxMarketValue int    `json:"mmlv"` // Max market limit value
	Priority       int    `json:"p"`    // Priority
	SportType      int    `json:"st"`   // Sport type
	MarketSubType  int    `json:"mst"`  // Market sub type
	MinValue       int    `json:"mdv"`  // Min default value
	MaxValue       int    `json:"mlv"`  // Max limit value
	IsActive       bool   `json:"in"`   // Is active
}

type IddaaEventStatistics struct {
	EventID       int                  `json:"EventId"`
	BulletinId    int                  `json:"BulletinId"`
	EventNo       string               `json:"EventNo"`
	League        string               `json:"League"`
	HomeTeam      string               `json:"HomeTeam"`
	AwayTeam      string               `json:"AwayTeam"`
	MatchDate     string               `json:"MatchDate"`
	Status        int                  `json:"Status"`
	Half          int                  `json:"Half"`
	MinuteOfMatch int                  `json:"MinuteOfMatch"`
	HomeScore     int                  `json:"HomeScore"`
	AwayScore     int                  `json:"AwayScore"`
	HalfTimeScore string               `json:"HalfTimeScore"`
	FullTimeScore string               `json:"FullTimeScore"`
	Statistics    IddaaMatchStatistics `json:"Statistics"`
	Events        []IddaaMatchEvent    `json:"Events"`
	IsLive        bool                 `json:"IsLive"`
	HasStatistics bool                 `json:"HasStatistics"`
	SportID       int                  `json:"SportId"`
}

type IddaaMatchStatistics struct {
	HomeStats IddaaTeamStats `json:"HomeTeam"`
	AwayStats IddaaTeamStats `json:"AwayTeam"`
}

type IddaaTeamStats struct {
	Shots         int `json:"Shots"`
	ShotsOnTarget int `json:"ShotsOnTarget"`
	Possession    int `json:"Possession"`
	Corners       int `json:"Corners"`
	YellowCards   int `json:"YellowCards"`
	RedCards      int `json:"RedCards"`
	Fouls         int `json:"Fouls"`
	Offsides      int `json:"Offsides"`
	FreeKicks     int `json:"FreeKicks"`
	ThrowIns      int `json:"ThrowIns"`
	GoalKicks     int `json:"GoalKicks"`
	Saves         int `json:"Saves"`
}

type IddaaMatchEvent struct {
	Minute      int    `json:"Minute"`
	EventType   string `json:"EventType"`
	Team        string `json:"Team"`
	Player      string `json:"Player"`
	Description string `json:"Description"`
	IsHome      bool   `json:"IsHome"`
}

// IddaaCompetitionsResponse represents the response from competitions endpoint
type IddaaCompetitionsResponse struct {
	IsSuccess bool               `json:"isSuccess"`
	Data      []IddaaCompetition `json:"data"`
	Message   string             `json:"message"`
}

// IddaaCompetition represents a single competition/league from Iddaa
type IddaaCompetition struct {
	ID        int    `json:"i"`    // Competition ID
	CountryID string `json:"cid"`  // Country code
	Priority  int    `json:"p"`    // Priority
	IconURL   string `json:"ic"`   // Icon URL
	ShortName string `json:"sn"`   // Short name
	SportID   string `json:"si"`   // Sport ID
	Name      string `json:"n"`    // Full name
	Reference int    `json:"cref"` // Reference ID
}

type IddaaSingleEventResponse struct {
	IsSuccess bool             `json:"isSuccess"`
	Data      IddaaSingleEvent `json:"data"`
	Message   string           `json:"message"`
}

type IddaaSingleEvent struct {
	ID            int                   `json:"i"`
	BulletinID    int                   `json:"bri"`
	Version       int                   `json:"v"`
	HomeTeam      string                `json:"hn"`
	AwayTeam      string                `json:"an"`
	SportID       int                   `json:"sid"`
	Status        int                   `json:"s"`
	BetProgram    int                   `json:"bp"`
	IsLive        bool                  `json:"il"`
	MBC           int                   `json:"mbc"`
	HasKingOdd    bool                  `json:"kOdd"`
	Markets       []IddaaDetailedMarket `json:"m"`
	CompetitionID int                   `json:"ci"`
	OddsCount     int                   `json:"oc"`
	Date          int64                 `json:"d"`
	HasCombine    bool                  `json:"hc"`
}

type IddaaDetailedMarket struct {
	ID           int                    `json:"i"`
	Type         int                    `json:"t"`
	SubType      int                    `json:"st"`
	Version      int                    `json:"v"`
	Status       int                    `json:"s"`
	MBC          int                    `json:"mbc"`
	SpecialValue string                 `json:"sov,omitempty"`
	Outcomes     []IddaaDetailedOutcome `json:"o"`
}

type IddaaDetailedOutcome struct {
	Number      int     `json:"no"`
	Odds        float64 `json:"odd"`
	WinningOdds float64 `json:"wodd"`
	Name        string  `json:"n"`
}
