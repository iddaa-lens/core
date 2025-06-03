# API Documentation

This guide covers the external API integration patterns, data models, and client implementation details for the Iddaa Core system.

## 🌐 External APIs

### Iddaa Sportsbook API

Base URL: `https://sportsbookv2.iddaa.com`

#### Competitions Endpoint

**GET** `/sportsbook/competitions`

Returns a list of all available competitions/leagues.

**Response Format:**
```json
{
    "isSuccess": true,
    "data": [
        {
            "i": 1516,          // Competition ID
            "cid": "INT",       // Country code
            "p": 1616,          // Parent ID
            "ic": "https://...", // Icon URL
            "sn": "Avr Şam Ka", // Short name
            "si": "1",          // Sport ID
            "n": "Avrupa Şamp., Kadınlar, Grup B", // Full name
            "cref": 3264        // External reference
        }
    ],
    "message": ""
}
```

**Field Mapping:**
- `i` → `iddaa_id` (Primary identifier)
- `cid` → `country_code` (Country/region)
- `p` → `parent_id` (Parent competition)
- `ic` → `icon_url` (Logo/icon)
- `sn` → `short_name` (Abbreviated name)
- `si` → `sport_id` (Sport type)
- `n` → `full_name` (Display name)
- `cref` → `external_ref` (External reference)

**Sport IDs:**
- `1` - Football
- `2` - Basketball
- `4` - Ice Hockey
- `5` - Tennis
- `6` - Handball
- `11` - Formula 1
- `23` - Other sports

### Iddaa Content API

Base URL: `https://contentv2.iddaa.com`

#### Configuration Endpoint

**GET** `/appconfig?platform=WEB`

Returns platform-specific configuration and system settings.

**Response Format:**
```json
{
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
        "nextDrawExpectedWin": 0.0,
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
}
```

**Key Configuration Fields:**

- **Coupon Limits**: Min/max bet amounts, event limits
- **King Odd Highlights**: Sport-specific odds boosts
- **Sportoto Info**: Current program and deadlines
- **Tax Settings**: Limits and percentage rates

## 🏗️ Client Implementation

### HTTP Client Structure

```go
// pkg/services/iddaa_client.go

type IddaaClient struct {
    baseURL string
    client  *http.Client
}

func NewIddaaClient(cfg *config.Config) *IddaaClient {
    return &IddaaClient{
        baseURL: "https://sportsbookv2.iddaa.com",
        client: &http.Client{
            Timeout: time.Duration(cfg.External.Timeout) * time.Second,
        },
    }
}
```

### Error Handling

```go
func (c *IddaaClient) makeRequest(url string, result interface{}) error {
    resp, err := c.client.Get(url)
    if err != nil {
        return fmt.Errorf("failed to make request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("API returned status %d", resp.StatusCode)
    }

    if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
        return fmt.Errorf("failed to decode response: %w", err)
    }

    return nil
}
```

### Response Validation

```go
type IddaaAPIResponse[T any] struct {
    IsSuccess bool   `json:"isSuccess"`
    Data      []T    `json:"data"`
    Message   string `json:"message"`
}

func (r *IddaaAPIResponse[T]) Validate() error {
    if !r.IsSuccess {
        return fmt.Errorf("API request failed: %s", r.Message)
    }
    return nil
}
```

## 📊 Data Models

### Competition Model

```go
type IddaaCompetition struct {
    ID          int    `json:"i"`          // iddaa_id
    CountryCode string `json:"cid"`        // country_code
    ParentID    int    `json:"p"`          // parent_id
    IconURL     string `json:"ic"`         // icon_url
    ShortName   string `json:"sn"`         // short_name
    SportID     string `json:"si"`         // sport_id
    FullName    string `json:"n"`          // full_name
    ExternalRef int    `json:"cref"`       // external_ref
}
```

### Configuration Model

```go
type IddaaConfigData struct {
    Platform              string            `json:"platform"`
    ConfigValue           ConfigValue       `json:"configValue"`
    SportotoProgramName   string            `json:"sportotoProgramName"`
    PayinEndDate          string            `json:"payinEndDate"`
    NextDrawExpectedWin   float64           `json:"nextDrawExpectedWin"`
    GlobalConfig          GlobalConfig      `json:"globalConfig"`
}

type GlobalConfig struct {
    Coupon     GlobalCouponConfig     `json:"coupon"`
    SoccerPool GlobalSoccerPoolConfig `json:"soccerPool"`
    Other      GlobalOtherConfig      `json:"other"`
}
```

## 🔄 Data Synchronization

### Sync Strategies

#### Competition Sync
- **Frequency**: Every 6 hours
- **Strategy**: Full sync with upsert
- **Conflict Resolution**: Update existing, insert new

```go
func (s *CompetitionService) SyncCompetitions(ctx context.Context) error {
    resp, err := s.client.GetCompetitions()
    if err != nil {
        return fmt.Errorf("failed to fetch competitions: %w", err)
    }

    for _, comp := range resp.Data {
        if err := s.saveCompetition(ctx, comp); err != nil {
            log.Printf("Failed to save competition %d: %v", comp.ID, err)
            continue
        }
    }
    return nil
}
```

#### Configuration Sync
- **Frequency**: Every 4 hours
- **Strategy**: Platform-specific replacement
- **Storage**: JSONB for flexible querying

```go
func (s *ConfigService) SyncConfig(ctx context.Context, platform string) error {
    resp, err := s.client.GetAppConfig(platform)
    if err != nil {
        return fmt.Errorf("failed to fetch config: %w", err)
    }

    return s.saveConfig(ctx, resp)
}
```

### Rate Limiting

- **Built-in Timeouts**: 30-second request timeout
- **Retry Logic**: Not implemented (fail-fast approach)
- **Respect API Limits**: Conservative sync frequencies

## 🧪 Testing Strategies

### API Client Testing

```go
func TestIddaaClient_GetCompetitions(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        response := IddaaAPIResponse[IddaaCompetition]{
            IsSuccess: true,
            Data: []IddaaCompetition{
                {ID: 1, CountryCode: "TR", FullName: "Test League"},
            },
        }
        json.NewEncoder(w).Encode(response)
    }))
    defer server.Close()

    client := NewIddaaClient(config)
    client.baseURL = server.URL

    result, err := client.GetCompetitions()
    assert.NoError(t, err)
    assert.Len(t, result.Data, 1)
}
```

### Mock Implementations

```go
type MockIddaaClient struct {
    competitions *models.IddaaAPIResponse[models.IddaaCompetition]
    config       *models.IddaaConfigResponse
    shouldError  bool
}

func (m *MockIddaaClient) GetCompetitions() (*models.IddaaAPIResponse[models.IddaaCompetition], error) {
    if m.shouldError {
        return nil, errors.New("mock error")
    }
    return m.competitions, nil
}
```

## 🔧 Configuration

### Environment Variables

```bash
# API Configuration
EXTERNAL_API_TIMEOUT=30      # Request timeout in seconds
EXTERNAL_API_URL=            # Override base URL (optional)
EXTERNAL_API_KEY=            # API key (if required)

# Retry Configuration
MAX_RETRIES=3                # Max retry attempts
RETRY_DELAY=1000            # Delay between retries (ms)
```

### Client Configuration

```go
type ExternalAPIConfig struct {
    BaseURL string
    APIKey  string
    Timeout int
}

func (c *ExternalAPIConfig) Validate() error {
    if c.Timeout <= 0 {
        return errors.New("timeout must be positive")
    }
    return nil
}
```

## 🚨 Error Handling

### Error Types

```go
type APIError struct {
    StatusCode int
    Message    string
    URL        string
}

func (e *APIError) Error() string {
    return fmt.Sprintf("API error %d: %s (URL: %s)", e.StatusCode, e.Message, e.URL)
}

type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}
```

### Recovery Strategies

1. **Network Errors**: Log and continue with next sync cycle
2. **API Errors**: Log error details, continue with partial data
3. **Validation Errors**: Skip invalid records, log for investigation
4. **Database Errors**: Fail fast, require manual intervention

## 📈 Monitoring

### Metrics to Track

- API response times
- Success/failure rates
- Data freshness (last successful sync)
- Record counts (competitions, config updates)

### Logging Examples

```go
log.Printf("Starting %s sync...", job.Name())
start := time.Now()

if err := job.Execute(ctx); err != nil {
    log.Printf("Job %s failed after %v: %v", job.Name(), time.Since(start), err)
} else {
    log.Printf("Job %s completed successfully in %v", job.Name(), time.Since(start))
}
```

## 🔮 Future Enhancements

### Planned API Integrations

1. **Events API**: `/sportsbook/competitions/{id}/events`
2. **Odds API**: `/sportsbook/events/{id}/odds`
3. **Live Data**: WebSocket connections for real-time updates
4. **Team Data**: Team profiles and statistics

### Optimization Opportunities

- **Caching**: Redis for frequently accessed data
- **Incremental Sync**: Delta updates instead of full sync
- **Parallel Processing**: Concurrent API calls
- **Circuit Breaker**: Fail-fast for downstream protection

---

This API integration provides a robust foundation for real-time sports betting data management with proper error handling, testing, and monitoring capabilities.

**Next**: [Deployment Guide](deployment.md)