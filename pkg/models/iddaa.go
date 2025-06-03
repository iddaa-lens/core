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

type IddaaCompetition struct {
	ID          int    `json:"i"`
	CountryCode string `json:"cid"`
	ParentID    int    `json:"p"`
	IconURL     string `json:"ic"`
	ShortName   string `json:"sn"`
	SportID     string `json:"si"`
	FullName    string `json:"n"`
	ExternalRef int    `json:"cref"`
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
	HasDigitalContent bool `json:"hd"` // Has digital content
}

type IddaaMarketConfig struct {
	ID          int    `json:"i"`  // Market ID
	Name        string `json:"n"`  // Market name
	ShortName   string `json:"sn"` // Short name
	Type        string `json:"t"`  // Market type
	SubType     string `json:"st"` // Market sub type
	SportID     int    `json:"si"` // Sport ID
	IsActive    bool   `json:"ia"` // Is active
	DisplayName string `json:"dn"` // Display name
	Description string `json:"d"`  // Description
	SortOrder   int    `json:"so"` // Sort order
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
