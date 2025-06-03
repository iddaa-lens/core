# Events API Integration

## Overview

This document describes the integration with the iddaa events API to collect and store live sports events, teams, and historical odds data.

## Implementation Summary

### 1. Data Models

**Events API Response Structure:**
- `IddaaEventsResponse` - Root response wrapper
- `IddaaEventsData` - Contains events array and metadata
- `IddaaEvent` - Individual event (match) data
- `IddaaMarket` - Betting market within an event
- `IddaaOutcome` - Individual betting option with odds

### 2. Database Integration

**Tables Updated:**
- `events` - Stores match information (teams, date, status)
- `teams` - Stores team information (auto-created from events)
- `odds` - Historical odds data with timestamps
- `market_types` - Betting market definitions

**Key Features:**
- **Upsert Strategy**: Events and teams are upserted to handle duplicates
- **Historical Tracking**: Odds are stored with timestamps for trend analysis
- **Foreign Key Relations**: Proper relationships between events, teams, competitions

### 3. Sync Job Implementation

**EventsSyncJob:**
- **Schedule**: Every 30 minutes for historical odds tracking
- **API Endpoint**: `https://sportsbookv2.iddaa.com/sportsbook/events?st=1&type=0&version=0`
- **Processing**: Extracts events, teams, markets, and odds from API response

**Data Flow:**
1. Fetch events from iddaa API
2. Parse and validate JSON response
3. Upsert teams (home/away) 
4. Upsert events with team references
5. Process markets and store odds with timestamps

### 4. Market Type Mapping

The system maps iddaa market subtypes to standardized codes:

| Iddaa Code | Market Type | Description |
|------------|-------------|-------------|
| 1 | 1X2 | Match Result |
| 60 | OU_0_5 | Over/Under 0.5 Goals |
| 101 | OU_2_5 | Over/Under 2.5 Goals |
| 89 | BTTS | Both Teams to Score |
| 88 | HT | Half Time Result |
| 92 | DC | Double Chance |
| 720 | RED_CARD | Red Card in Match |

### 5. Usage Examples

**Run events sync manually:**
```bash
DATABASE_URL="postgres://user:pass@localhost:5433/iddaa_core?sslmode=disable" \
./bin/cron -job events -once
```

**Run all jobs continuously:**
```bash
DATABASE_URL="postgres://user:pass@localhost:5433/iddaa_core?sslmode=disable" \
./bin/cron
```

### 6. API Response Sample

See `docs/iddaa-api-events.md` for complete API documentation including:
- Full JSON response format
- Field meanings and translations
- Status codes and market types
- Turkish-English outcome translations

### 7. Next Steps

**For Database Setup:**
1. Ensure PostgreSQL is running on port 5433
2. Create `iddaa_core` database
3. Run migrations: `DATABASE_URL="postgresql://iddaa:iddaa123@localhost:5433/iddaa_core?sslmode=disable" make migrate`
4. Test with: `DATABASE_URL="postgresql://iddaa:iddaa123@localhost:5433/iddaa_core?sslmode=disable" ./bin/cron -job events -once`

**✅ Test Results:**
- Successfully processed 105 events from iddaa API
- All teams, events, markets, and odds stored correctly
- Takes ~8 seconds to fetch and process full events data

**For API Integration:**
1. Monitor API rate limits
2. Add error handling for API failures
3. Consider adding competition sync before events
4. Add logging for odds changes and trends

**For Historical Analysis:**
1. Odds data accumulates over time for trend analysis
2. Query historical odds by event and market type
3. Build analytics on odds movements
4. Potential ML features for prediction models