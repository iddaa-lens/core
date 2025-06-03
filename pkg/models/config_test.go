package models

import (
	"encoding/json"
	"testing"
)

func TestIddaaConfigResponse_Unmarshal(t *testing.T) {
	jsonData := `{
    "isSuccess": true,
    "data": {
        "platform": "WEB",
        "configValue": {
            "coupon": {
                "mobileFastSelectValues": [50, 100, 250, 500, 1000]
            },
            "kingOddHighLight": {
                "sports": {
                    "1": {
                        "-1": {
                            "preValue": "4",
                            "preDetail": "4",
                            "liveDetail": "5"
                        }
                    }
                },
                "templates": {
                    "pre": "Kral Oran ile Her Maçta %{value} Daha Yüksek Oran",
                    "preDetail": "Kral Oran ile Bu Maçta %{value} Daha Yüksek Oranlar",
                    "liveDetail": "Canlıda Bu Maçta Oranlar %{value} Daha Yüksek"
                }
            }
        },
        "sportotoProgramName": "06-09 Haziran",
        "payinEndDate": "2025-06-06T21:40:00",
        "nextDrawExpectedWin": 12500.75,
        "globalConfig": {
            "coupon": {
                "maxPrice": 20000,
                "minPrice": 50,
                "maxMultiCount": 100,
                "maxLiveMultiCount": 100,
                "maxEarning": 12500000,
                "maxEventCount": 20,
                "minMultiPrice": 2500,
                "taxLimit": 53339,
                "taxPercentage": 20
            },
            "soccerPool": {
                "maxColumn": 2500,
                "columnPrice": 10
            },
            "other": {
                "phase": 4,
                "balanceReset": 50
            }
        }
    },
    "message": ""
}`

	var response IddaaConfigResponse
	err := json.Unmarshal([]byte(jsonData), &response)

	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if !response.IsSuccess {
		t.Error("Expected IsSuccess to be true")
	}

	if response.Message != "" {
		t.Errorf("Expected empty message, got '%s'", response.Message)
	}

	data := response.Data
	if data.Platform != "WEB" {
		t.Errorf("Expected platform 'WEB', got '%s'", data.Platform)
	}

	if data.SportotoProgramName != "06-09 Haziran" {
		t.Errorf("Expected sportoto program name '06-09 Haziran', got '%s'", data.SportotoProgramName)
	}

	if data.PayinEndDate != "2025-06-06T21:40:00" {
		t.Errorf("Expected payin end date '2025-06-06T21:40:00', got '%s'", data.PayinEndDate)
	}

	if data.NextDrawExpectedWin != 12500.75 {
		t.Errorf("Expected next draw expected win 12500.75, got %f", data.NextDrawExpectedWin)
	}

	// Test coupon config
	coupon := data.ConfigValue.Coupon
	if len(coupon.MobileFastSelectValues) != 5 {
		t.Errorf("Expected 5 mobile fast select values, got %d", len(coupon.MobileFastSelectValues))
	}
	if coupon.MobileFastSelectValues[0] != 50 {
		t.Errorf("Expected first mobile fast select value 50, got %d", coupon.MobileFastSelectValues[0])
	}

	// Test king odd highlight
	kingOdd := data.ConfigValue.KingOddHighLight
	if kingOdd.Templates.Pre != "Kral Oran ile Her Maçta %{value} Daha Yüksek Oran" {
		t.Errorf("Expected pre template, got '%s'", kingOdd.Templates.Pre)
	}

	sport1, exists := kingOdd.Sports["1"]
	if !exists {
		t.Error("Expected sport '1' to exist in kingOddHighLight.sports")
	}

	defaultOdd, exists := sport1["-1"]
	if !exists {
		t.Error("Expected default odd '-1' to exist in sport '1'")
	}

	if defaultOdd.PreValue != "4" {
		t.Errorf("Expected preValue '4', got '%s'", defaultOdd.PreValue)
	}

	// Test global config
	globalConfig := data.GlobalConfig
	if globalConfig.Coupon.MaxPrice != 20000 {
		t.Errorf("Expected max price 20000, got %d", globalConfig.Coupon.MaxPrice)
	}

	if globalConfig.SoccerPool.MaxColumn != 2500 {
		t.Errorf("Expected max column 2500, got %d", globalConfig.SoccerPool.MaxColumn)
	}

	if globalConfig.Other.Phase != 4 {
		t.Errorf("Expected phase 4, got %d", globalConfig.Other.Phase)
	}
}

func TestIddaaConfigResponse_MarshalUnmarshal(t *testing.T) {
	// Test round-trip marshalling
	original := IddaaConfigResponse{
		IsSuccess: true,
		Data: IddaaConfigData{
			Platform: "WEB",
			ConfigValue: ConfigValue{
				Coupon: CouponConfig{
					MobileFastSelectValues: []int{50, 100, 250},
				},
			},
			SportotoProgramName: "Test Program",
			NextDrawExpectedWin: 1000.50,
		},
		Message: "success",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var result IddaaConfigResponse
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Compare
	if result.IsSuccess != original.IsSuccess {
		t.Errorf("IsSuccess mismatch: expected %v, got %v", original.IsSuccess, result.IsSuccess)
	}

	if result.Data.Platform != original.Data.Platform {
		t.Errorf("Platform mismatch: expected %s, got %s", original.Data.Platform, result.Data.Platform)
	}

	if result.Data.NextDrawExpectedWin != original.Data.NextDrawExpectedWin {
		t.Errorf("NextDrawExpectedWin mismatch: expected %f, got %f", original.Data.NextDrawExpectedWin, result.Data.NextDrawExpectedWin)
	}
}
