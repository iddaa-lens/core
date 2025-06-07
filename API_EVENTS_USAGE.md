# Events API Endpoint

The new `/api/events` endpoint provides filtered access to sports events with comprehensive information including teams, leagues, sports, and betting data.

## Endpoint

```
GET /api/events
```

## Query Parameters

| Parameter     | Type   | Default | Max | Description |
|---------------|--------|---------|-----|-------------|
| `hours_before` | int    | 24      | 168 | Hours before current time to include events |
| `hours_after`  | int    | 24      | 168 | Hours after current time to include events |
| `sport`       | string | -       | -   | Filter by sport code (e.g., "1" for football) |
| `league`      | string | -       | -   | Filter by league name (partial match, case insensitive) |
| `status`      | string | -       | -   | Filter by status: "live", "scheduled", or "finished" |
| `limit`       | int    | 100     | 500 | Maximum number of events to return |

## Example Requests

### Get all events in the next 24 hours
```bash
curl "http://localhost:8080/api/events"
```

### Get live events only
```bash
curl "http://localhost:8080/api/events?status=live"
```

### Get football events in the next 48 hours
```bash
curl "http://localhost:8080/api/events?sport=1&hours_after=48"
```

### Get events from a specific league
```bash
curl "http://localhost:8080/api/events?league=Premier%20League"
```

### Get events from the last 6 hours to next 12 hours, limited to 50 results
```bash
curl "http://localhost:8080/api/events?hours_before=6&hours_after=12&limit=50"
```

## Response Format

The endpoint returns a JSON array of event objects:

```json
[
  {
    "id": 123,
    "external_id": "evt_12345",
    "slug": "manchester-united-vs-liverpool-2025-06-06",
    "event_date": "2025-06-06T20:00:00Z",
    "status": "scheduled",
    "home_score": null,
    "away_score": null,
    "is_live": false,
    "minute_of_match": null,
    "half": null,
    "betting_volume_percentage": 15.7,
    "volume_rank": 3,
    "has_king_odd": true,
    "odds_count": 127,
    "has_combine": true,
    "home_team": "Manchester United",
    "home_team_country": "England",
    "away_team": "Liverpool",
    "away_team_country": "England",
    "league": "Premier League",
    "league_country": "England",
    "sport": "Football",
    "sport_code": "1",
    "match": "Manchester United vs Liverpool"
  }
]
```

## Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | int | Internal event ID |
| `external_id` | string | External system event ID |
| `slug` | string | URL-friendly event identifier |
| `event_date` | string | ISO 8601 formatted event date/time |
| `status` | string | Event status (live, scheduled, finished) |
| `home_score` | int/null | Home team score (null if not started) |
| `away_score` | int/null | Away team score (null if not started) |
| `is_live` | boolean | Whether the event is currently live |
| `minute_of_match` | int/null | Current minute of the match (live events) |
| `half` | int/null | Current half/period (live events) |
| `betting_volume_percentage` | float/null | Betting volume percentage |
| `volume_rank` | int/null | Rank by betting volume |
| `has_king_odd` | boolean | Whether event has king odds |
| `odds_count` | int/null | Number of available odds |
| `has_combine` | boolean | Whether combine bets are available |
| `home_team` | string | Home team name |
| `home_team_country` | string | Home team country |
| `away_team` | string | Away team name |
| `away_team_country` | string | Away team country |
| `league` | string | League name |
| `league_country` | string | League country |
| `sport` | string | Sport name |
| `sport_code` | string | Sport code |
| `match` | string | Formatted match name (Home vs Away) |

## Error Responses

- **400 Bad Request**: Invalid parameter values
- **500 Internal Server Error**: Database or server error

## Implementation Notes

- The endpoint follows the same patterns as the existing `/api/odds/big-movers` endpoint
- Uses database connection pooling and proper error handling
- Includes comprehensive logging for monitoring and debugging
- Supports CORS for frontend integration
- All timestamp parameters are relative to the current server time
- League name filtering uses case-insensitive partial matching
- Results are ordered by event date (ascending)