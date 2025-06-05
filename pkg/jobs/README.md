# Cron Jobs Documentation

This document describes all available cron jobs in the iddaa-core backend services.

## Overview

The system includes 10 distinct cron jobs that handle data synchronization, analytics, and maintenance operations. All jobs support individual execution using the `--job` flag for testing and troubleshooting.

## Job List

### 1. Config Sync (`config`)
- **Schedule**: `0 6 * * 1` (Weekly on Mondays at 6 AM)
- **Purpose**: Syncs application configuration from Iddaa's content API
- **Implementation**: `config_sync.go`
- **Dependencies**: Iddaa API access
- **API Endpoint**: `https://contentv2.iddaa.com/appconfig?platform=WEB`
- **Database Tables**: `app_config`
- **Test Command**: `./cron --job=config --once`

### 2. Sports Sync (`sports`)
- **Schedule**: `*/30 * * * *` (Every 30 minutes)
- **Purpose**: Keeps sports information up to date with live/upcoming event counts
- **Implementation**: `sports_sync.go` 
- **Dependencies**: Iddaa API access
- **API Endpoint**: `https://sportsbookv2.iddaa.com/sportsbook/info`
- **Database Tables**: `sports`
- **Test Command**: `./cron --job=sports --once`
- **Notes**: Foundation job - other jobs depend on sports data

### 3. Events Sync (`events`)
- **Schedule**: `*/5 * * * *` (Every 5 minutes)
- **Purpose**: Captures rapid odds movements and event data for all active sports
- **Implementation**: `events_sync.go`
- **Dependencies**: Iddaa API access, requires sports data
- **API Endpoint**: `https://sportsbookv2.iddaa.com/sportsbook/events?st={sport_id}&type=0&version=0`
- **Database Tables**: `events`, `current_odds`, `odds_history`
- **Test Command**: `./cron --job=events --once`
- **Notes**: High frequency job for real-time data capture

### 4. Volume Sync (`volume`)
- **Schedule**: `*/20 * * * *` (Every 20 minutes)
- **Purpose**: Tracks betting volume changes for all sports
- **Implementation**: `volume_sync.go`
- **Dependencies**: Iddaa API access, requires sports data
- **Database Tables**: `betting_volume_history`
- **Test Command**: `./cron --job=volume --once`

### 5. Distribution Sync (`distribution`)
- **Schedule**: `0 * * * *` (Every hour)
- **Purpose**: Tracks betting distribution changes for all sports
- **Implementation**: `distribution_sync.go`
- **Dependencies**: Iddaa API access, requires sports data
- **Database Tables**: `outcome_distributions`, `outcome_distribution_history`
- **Test Command**: `./cron --job=distribution --once`

### 6. Analytics Refresh (`analytics`)
- **Schedule**: `0 */6 * * *` (Every 6 hours)
- **Purpose**: Refreshes materialized views and analytics
- **Implementation**: `analytics_refresh.go`
- **Dependencies**: Database access only (no external APIs)
- **Database Operations**: Refreshes materialized views (e.g., contrarian bets)
- **Test Command**: `./cron --job=analytics --once`
- **Notes**: Currently has issues with missing materialized views

### 7. Market Config Sync (`market_config`)
- **Schedule**: `0 6 * * *` (Daily at 6 AM)
- **Purpose**: Syncs market configurations and betting types
- **Implementation**: `market_config_sync.go`
- **Dependencies**: Iddaa API access
- **Database Tables**: `market_types`
- **Test Command**: `./cron --job=market_config --once`
- **Notes**: Syncs 600+ market configurations

### 8. Statistics Sync (`statistics`)
- **Schedule**: `*/15 8-23 * * *` (Every 15 minutes during 8 AM to 11 PM)
- **Purpose**: Syncs event statistics for football covering European match times
- **Implementation**: `statistics_sync.go`
- **Dependencies**: Statistics service API access
- **Database Tables**: `match_statistics`
- **Test Command**: `./cron --job=statistics --once`
- **Notes**: Only runs during active hours, currently has JSON parsing issues

### 9. Detailed Odds Sync (`detailed_odds`)
- **Schedule**: `*/2 * * * *` (Every 2 minutes)
- **Purpose**: High-frequency detailed odds tracking for live and near-live events using individual event endpoint
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
- **Purpose**: Syncs leagues and teams with Football API integration and AI translation
- **Implementation**: `leagues_sync.go`
- **Dependencies**: 
  - Iddaa API access (required)
  - `FOOTBALL_API_KEY` environment variable (optional)
  - `OPENAI_API_KEY` environment variable (optional)
- **Database Tables**: `leagues`, `teams`, `league_mappings`, `team_mappings`
- **Test Command**: `./cron --job=leagues --once`
- **Features**:
  - Multi-step process: Iddaa leagues → Football API mapping → Teams sync
  - AI-powered translation for league names
  - Graceful degradation if external APIs unavailable

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
10. `analytics` - Analytics refresh

### External API Dependencies
- **Iddaa API**: All jobs except `analytics`
- **Football API**: `leagues` job (optional)
- **OpenAI API**: `leagues` job for translation (optional)

## Environment Variables

Required for full functionality:
```bash
export DATABASE_URL="postgresql://user:pass@host:port/db?sslmode=disable"
export FOOTBALL_API_KEY="your_football_api_key"  # Optional for leagues job
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

# Analytics jobs  
./cron --job=statistics --once
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