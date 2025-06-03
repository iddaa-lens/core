package models

type IddaaConfigResponse struct {
	IsSuccess bool            `json:"isSuccess"`
	Data      IddaaConfigData `json:"data"`
	Message   string          `json:"message"`
}

type IddaaConfigData struct {
	Platform            string       `json:"platform"`
	ConfigValue         ConfigValue  `json:"configValue"`
	SportotoProgramName string       `json:"sportotoProgramName"`
	PayinEndDate        string       `json:"payinEndDate"`
	NextDrawExpectedWin float64      `json:"nextDrawExpectedWin"`
	GlobalConfig        GlobalConfig `json:"globalConfig"`
}

type ConfigValue struct {
	Coupon           CouponConfig           `json:"coupon"`
	KingOddHighLight KingOddHighLightConfig `json:"kingOddHighLight"`
}

type CouponConfig struct {
	MobileFastSelectValues []int `json:"mobileFastSelectValues"`
}

type KingOddHighLightConfig struct {
	Sports    map[string]map[string]KingOddValues `json:"sports"`
	Templates KingOddTemplates                    `json:"templates"`
}

type KingOddValues struct {
	PreValue   string `json:"preValue"`
	PreDetail  string `json:"preDetail"`
	LiveDetail string `json:"liveDetail"`
}

type KingOddTemplates struct {
	Pre        string `json:"pre"`
	PreDetail  string `json:"preDetail"`
	LiveDetail string `json:"liveDetail"`
}

type GlobalConfig struct {
	Coupon     GlobalCouponConfig     `json:"coupon"`
	SoccerPool GlobalSoccerPoolConfig `json:"soccerPool"`
	Other      GlobalOtherConfig      `json:"other"`
}

type GlobalCouponConfig struct {
	MaxPrice          int `json:"maxPrice"`
	MinPrice          int `json:"minPrice"`
	MaxMultiCount     int `json:"maxMultiCount"`
	MaxLiveMultiCount int `json:"maxLiveMultiCount"`
	MaxEarning        int `json:"maxEarning"`
	MaxEventCount     int `json:"maxEventCount"`
	MinMultiPrice     int `json:"minMultiPrice"`
	TaxLimit          int `json:"taxLimit"`
	TaxPercentage     int `json:"taxPercentage"`
}

type GlobalSoccerPoolConfig struct {
	MaxColumn   int `json:"maxColumn"`
	ColumnPrice int `json:"columnPrice"`
}

type GlobalOtherConfig struct {
	Phase        int `json:"phase"`
	BalanceReset int `json:"balanceReset"`
}
