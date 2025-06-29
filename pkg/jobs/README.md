# Cron Jobs Documentation

This document describes all available cron jobs in the iddaa-core backend services.

## Overview

The system includes 15 distinct cron jobs that handle data synchronization, analytics, and maintenance operations. All jobs support individual execution using the `--job` flag for testing and troubleshooting.

## Job List

### 1. Config Sync (`config`)

- **Schedule**: `0 12 * * *` (Daily at 12 PM)
- **Summary**: Syncs app configuration settings from Iddaa API
- **Implementation**: `config_sync.go`
- **Dependencies**: Iddaa API access
- **API Endpoint**: `https://contentv2.iddaa.com/appconfig?platform=WEB`
- **Database Tables**: `app_config`
- **Test Command**: `./cron --job=config --once`

### 2. Sports Sync (`sports`)

- **Schedule**: `*/30 * * * *` (Every 30 minutes)
- **Summary**: Updates sports list with live/upcoming event counts
- **Implementation**: `sports_sync.go`
- **Dependencies**: Iddaa API access
- **API Endpoint**: `https://sportsbookv2.iddaa.com/sportsbook/info`
- **Database Tables**: `sports`
- **Test Command**: `./cron --job=sports --once`
- **Notes**: Foundation job - other jobs depend on sports data

### 3. Events Sync (`events`)

- **Schedule**: `*/5 * * * *` (Every 5 minutes)
- **Summary**: Syncs match events and basic odds for all sports
- **Implementation**: `events_sync.go`
- **Dependencies**: Iddaa API access, requires sports data
- **API Endpoint**: `https://sportsbookv2.iddaa.com/sportsbook/events?st={sport_id}&type=0&version=0`
- **Database Tables**: `events`, `current_odds`, `odds_history`
- **Test Command**: `./cron --job=events --once`
- **Notes**: High frequency job for real-time data capture

### 4. Volume Sync (`volume`)

- **Schedule**: `*/15 * * * *` (Every 15 minutes)
- **Summary**: Tracks betting volume percentages and rankings
- **Implementation**: `volume_sync.go`
- **Dependencies**: Iddaa API access, requires sports data
- **Database Tables**: `betting_volume_history`
- **Test Command**: `./cron --job=volume --once`

### 5. Distribution Sync (`distribution`)

- **Schedule**: `*/15 * * * *` (Every 15 minutes)
- **Summary**: Updates outcome betting distribution percentages
- **Implementation**: `distribution_sync.go`
- **Dependencies**: Iddaa API access, requires sports data
- **Database Tables**: `outcome_distributions`, `outcome_distribution_history`
- **Test Command**: `./cron --job=distribution --once`

### 6. Analytics Refresh (`analytics`)

- **Schedule**: `*/15 * * * *` (Every 15 minutes)
- **Summary**: Refreshes materialized views for analytics
- **Implementation**: `analytics_refresh.go`
- **Dependencies**: Database access only (no external APIs)
- **Database Operations**: Refreshes materialized views (e.g., contrarian bets)
- **Test Command**: `./cron --job=analytics --once`
- **Notes**: Currently has issues with missing materialized views

### 7. Market Config Sync (`market_config`)

- **Schedule**: `*/15 * * * *` (Every 15 minutes)
- **Summary**: Syncs 600+ market types and betting options
- **Implementation**: `market_config_sync.go`
- **Dependencies**: Iddaa API access
- **Database Tables**: `market_types`
- **Test Command**: `./cron --job=market_config --once`
- **Notes**: Syncs 600+ market configurations

### 8. Statistics Sync (`statistics`)

- **Schedule**: `*/15 * * * *` (Every 15 minutes)
- **Summary**: Fetches match statistics during active hours
- **Implementation**: `statistics_sync.go`
- **Dependencies**: Statistics service API access
- **Database Tables**: `match_statistics`
- **Test Command**: `./cron --job=statistics --once`
- **Notes**: Only runs during active hours, currently has JSON parsing issues

### 9. Detailed Odds Sync (`detailed_odds`)

- **Schedule**: `*/2 * * * *` (Every 2 minutes)
- **Summary**: High-frequency tracking of all markets for active events
- **Implementation**: `detailed_odds_sync.go`
- **Dependencies**: Iddaa API access, requires existing events data
- **API Endpoint**: `https://sportsbookv2.iddaa.com/sportsbook/event/{external_id}`
- **Database Tables**: `current_odds`, `odds_history`, `market_types`
- **Test Command**: `./cron --job=detailed_odds --once`
- **Features**:
  - Targets live and scheduled events within 24-hour window
  - Complete market coverage (15-30+ markets per event vs 3-5 in bulk sync)
  - Enhanced odds data with written odds (`wodd`) vs current odds (`odd`)
  - Rate limited to prevent API overload (100ms delay between requests)
  - Live event prioritization for real-time tracking

### 10. Leagues Sync (`leagues`)

- **Schedule**: `0 2 * * *` (Daily at 2 AM)
- **Summary**: Syncs leagues/teams with API-Football matching
- **Implementation**: `leagues_sync.go`
- **Dependencies**:
  - Iddaa API access (required)
  - `API_FOOTBALL_API_KEY` environment variable (optional)
  - `OPENAI_API_KEY` environment variable (optional)
- **Database Tables**: `leagues`, `teams`, `league_mappings`, `team_mappings`
- **Test Command**: `./cron --job=leagues --once`
- **Features**:
  - Multi-step process: Iddaa leagues → Football API mapping → Teams sync
  - AI-powered translation for league names
  - Graceful degradation if external APIs unavailable

### 11. Smart Money Processor (`smart_money_processor`)

- **Schedule**: `*/15 * * * *` (Every 15 minutes)
- **Summary**: Detects sharp money movements and generates alerts
- **Implementation**: `smart_money_processor.go`
- **Dependencies**: Database access, requires existing odds and distribution data
- **Database Tables**: `smart_money_alerts`, `smart_money_movements`
- **Test Command**: `./cron --job=smart_money_processor --once`
- **Features**:
  - Calculates confidence scores based on multiple factors
  - Identifies sharp vs public money movements
  - Tracks historical smart money performance

### 12. API Football League Matching (`api_football_league_matching`)

- **Schedule**: `0 3 * * *` (Daily at 3 AM)
- **Summary**: Maps Turkish leagues to API-Football IDs
- **Implementation**: `api_football_league_matching.go`
- **Dependencies**: API-Football API key required
- **Database Tables**: `league_mappings`
- **Test Command**: `./cron --job=api_football_league_matching --once`
- **Features**:
  - Uses similarity matching with 70%+ confidence threshold
  - Handles Turkish to English translations
  - Creates mapping entries for data enrichment

### 13. API Football Team Matching (`api_football_team_matching`)

- **Schedule**: `0 4 * * *` (Daily at 4 AM)
- **Summary**: Maps Turkish teams to API-Football IDs
- **Implementation**: `api_football_team_matching.go`
- **Dependencies**: API-Football API key required
- **Database Tables**: `team_mappings`
- **Test Command**: `./cron --job=api_football_team_matching --once`
- **Features**:
  - League-aware team matching
  - Confidence scoring for match quality
  - Prevents duplicate mappings

### 14. API Football League Enrichment (`api_football_league_enrichment`)

- **Schedule**: `0 5 * * 0` (Weekly on Sundays at 5 AM)
- **Summary**: Adds logos and metadata to mapped leagues
- **Implementation**: `api_football_league_enrichment.go`
- **Dependencies**: API-Football API key required, requires league mappings
- **Database Tables**: `leagues` (updates enrichment fields)
- **Test Command**: `./cron --job=api_football_league_enrichment --once`
- **Features**:
  - Fetches league logos and country flags
  - Updates league metadata (type, available features)
  - Only processes mapped leagues

### 15. API Football Team Enrichment (`api_football_team_enrichment`)

- **Schedule**: `0 6 * * 0` (Weekly on Sundays at 6 AM)
- **Summary**: Adds logos and venue data to mapped teams
- **Implementation**: `api_football_team_enrichment.go`
- **Dependencies**: API-Football API key required, requires team mappings
- **Database Tables**: `teams` (updates enrichment fields)
- **Test Command**: `./cron --job=api_football_team_enrichment --once`
- **Features**:
  - Fetches team logos and venue details
  - Updates team metadata (founded year, capacity)
  - Only processes mapped teams

## Job Dependencies

### Execution Order

Jobs should typically be run in this order for initial setup:

1. `sports` - Foundation data
2. `leagues` - League/team data  
3. `market_config` - Market configurations
4. `config` - Application config
5. `events` - Live event data
6. `detailed_odds` - Detailed odds tracking (requires events data)
7. `volume` - Volume data
8. `distribution` - Distribution data
9. `statistics` - Event statistics
10. `api_football_league_matching` - Match leagues with API-Football
11. `api_football_team_matching` - Match teams with API-Football
12. `api_football_league_enrichment` - Enrich league data
13. `api_football_team_enrichment` - Enrich team data
14. `smart_money_processor` - Smart money detection
15. `analytics` - Analytics refresh

### External API Dependencies

- **Iddaa API**: All jobs except `analytics`, `smart_money_processor`, and API-Football enrichment jobs
- **Football API**: `leagues`, `api_football_league_matching`, `api_football_team_matching`, `api_football_league_enrichment`, `api_football_team_enrichment`
- **OpenAI API**: `leagues` job for translation (optional)

## Environment Variables

Required for full functionality:

```bash
export DATABASE_URL="postgresql://user:pass@host:port/db?sslmode=disable"
export API_FOOTBALL_API_KEY="your_API_FOOTBALL_API_KEY"  # Optional for leagues job
export OPENAI_API_KEY="your_openai_api_key"      # Optional for AI translation
```

## Testing All Jobs

To test all jobs in sequence:

```bash
export DATABASE_URL="postgresql://iddaa:iddaa123@localhost:5433/iddaa_core?sslmode=disable"

# Core data jobs
./cron --job=sports --once
./cron --job=leagues --once  
./cron --job=market_config --once
./cron --job=config --once

# Live data jobs
./cron --job=events --once
./cron --job=detailed_odds --once  # High-frequency odds tracking
./cron --job=volume --once
./cron --job=distribution --once

# API-Football enrichment jobs (if API key is set)
./cron --job=api_football_league_matching --once
./cron --job=api_football_team_matching --once
./cron --job=api_football_league_enrichment --once
./cron --job=api_football_team_enrichment --once

# Analytics jobs  
./cron --job=statistics --once
./cron --job=smart_money_processor --once
./cron --job=analytics --once
```

## Production Considerations

### Anti-Bot Headers

All Iddaa API jobs include comprehensive browser-like headers to prevent bot detection:

- Realistic User-Agent strings
- Standard browser headers (Sec-Ch-Ua, Sec-Fetch-*, etc.)
- Iddaa-specific headers (Client-Transaction-Id, Platform, Timestamp)

### Error Handling

- Jobs continue processing even if individual items fail
- Comprehensive logging for troubleshooting
- Graceful degradation when external APIs are unavailable

### Database Performance

- Uses efficient upsert operations (`ON CONFLICT DO UPDATE`)
- Bulk operations where possible
- Transaction management for data consistency

## Monitoring

Key metrics to monitor in production:

- Job execution success/failure rates
- API response times and error rates
- Database connection pool usage
- Data freshness (time since last successful sync)
- Iddaa API rate limiting and potential blocking

## Known Issues

1. **Statistics Sync**: JSON parsing issues with Iddaa statistics API
2. **Analytics Refresh**: Missing materialized views in database schema
3. **Leagues Sync**: One league fails with foreign key constraint (sport_id 124 not found)

## Troubleshooting

### Common Issues

- **Gzip Compression**: Ensure HTTP client doesn't manually set Accept-Encoding header
- **Database Connections**: Verify DATABASE_URL is correctly formatted
- **API Keys**: Check environment variables are set for optional services
- **Time Zones**: Jobs run in UTC, ensure clock synchronization

### Debug Commands

```bash
# Test individual job with verbose logging
./cron --job=sports --once
./cron --job=detailed_odds --once

# Check database connection
psql $DATABASE_URL -c "SELECT 1;"

# Check recent odds data
psql $DATABASE_URL -c "SELECT COUNT(*) FROM current_odds;"
psql $DATABASE_URL -c "SELECT COUNT(*) FROM odds_history WHERE created_at > NOW() - INTERVAL '1 hour';"

# Verify job registration (should show all 10 jobs)
./cron --help
```
