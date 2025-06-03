# Finding Big Odds Movements

## Overview

The system is designed to efficiently track and identify significant odds movements, which can indicate:
- Market sentiment shifts
- Insider information
- Team news impact
- Betting patterns

## Database Design for Movement Detection

### Two-Table Architecture

1. **`current_odds`** - Stores latest odds with movement tracking:
   - `opening_value` - First recorded odds (e.g., 1.20)
   - `odds_value` - Current odds (e.g., 3.70)
   - `highest_value` - Maximum odds seen
   - `lowest_value` - Minimum odds seen
   - `movement_percentage` - Auto-calculated: (current - opening) / opening * 100
   - `total_movement` - Auto-calculated: highest - lowest

2. **`odds_history`** - Stores only when odds change:
   - `previous_value` - What the odds were
   - `odds_value` - What they changed to
   - `change_percentage` - Auto-calculated percentage change
   - `multiplier` - Auto-calculated: new / old (e.g., 3.70 / 1.20 = 3.08x)

## Query Examples

### 1. Find Events with Big Movements (Your Use Case)

```sql
-- Find events where odds moved significantly (like 1.20 → 3.70)
SELECT * FROM big_movers
WHERE multiplier > 2  -- Odds more than doubled
   OR movement_percentage > 100;  -- More than 100% increase

-- Example output:
-- event_slug: fenerbahce-vs-galatasaray-2025-06-05
-- market: 1X2
-- outcome: 1 (Home Win)
-- opening_value: 1.20
-- current_value: 3.70
-- multiplier: 3.08
-- movement_percentage: 208.33%
-- trend_direction: DRIFTING
```

### 2. Get Biggest Movers in Last 24 Hours

```sql
-- Using the built-in function
SELECT * FROM get_biggest_movers(
    hours_back := 24,
    min_movement_pct := 50
);

-- Direct query for more control
SELECT 
    e.slug,
    ht.name || ' vs ' || at.name as match,
    mt.name as market,
    co.outcome,
    co.opening_value || ' → ' || co.odds_value as movement,
    co.movement_percentage || '%' as change,
    ROUND(co.odds_value / co.opening_value, 2) || 'x' as multiplier
FROM current_odds co
JOIN events e ON co.event_id = e.id
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
JOIN market_types mt ON co.market_type_id = mt.id
WHERE co.last_updated > NOW() - INTERVAL '24 hours'
  AND (
    co.odds_value / co.opening_value > 2  -- Doubled
    OR co.opening_value / co.odds_value > 2  -- Halved
  )
ORDER BY ABS(co.movement_percentage) DESC;
```

### 3. Find Suspicious Movements

```sql
-- Rapid changes that might indicate insider activity
SELECT * FROM suspicious_movements
WHERE changes_last_hour > 3  -- More than 3 changes in an hour
  AND multiplier > 1.5;  -- And significant movement

-- Events with extreme volatility
SELECT 
    event_slug,
    SUM(changes_count) as total_changes,
    AVG(volatility_score) as avg_volatility,
    MAX(max_single_change) as biggest_swing
FROM (
    SELECT 
        e.slug as event_slug,
        analyze_event_volatility(e.id).*
    FROM events e
    WHERE e.event_date > NOW()
) analysis
GROUP BY event_slug
HAVING SUM(changes_count) > 10
ORDER BY AVG(volatility_score) DESC;
```

### 4. Track Specific Event Movement History

```sql
-- See how odds changed over time for a specific event
SELECT 
    oh.recorded_at,
    mt.name as market,
    oh.outcome,
    oh.previous_value || ' → ' || oh.odds_value as change,
    oh.change_percentage || '%' as pct_change,
    oh.multiplier || 'x' as multiplier
FROM odds_history oh
JOIN events e ON oh.event_id = e.id
JOIN market_types mt ON oh.market_type_id = mt.id
WHERE e.slug = 'fenerbahce-vs-galatasaray-2025-06-05'
  AND ABS(oh.change_percentage) > 10  -- Only significant changes
ORDER BY oh.recorded_at DESC;
```

### 5. Market Trend Analysis

```sql
-- Which markets are most volatile?
SELECT 
    mt.name as market,
    COUNT(DISTINCT oh.event_id) as events_affected,
    AVG(ABS(oh.change_percentage)) as avg_movement,
    MAX(ABS(oh.change_percentage)) as max_movement,
    SUM(CASE WHEN oh.change_percentage > 0 THEN 1 ELSE 0 END) as increases,
    SUM(CASE WHEN oh.change_percentage < 0 THEN 1 ELSE 0 END) as decreases
FROM odds_history oh
JOIN market_types mt ON oh.market_type_id = mt.id
WHERE oh.recorded_at > NOW() - INTERVAL '7 days'
GROUP BY mt.name
ORDER BY avg_movement DESC;
```

## API Integration Ideas

### Alerts Endpoint
```go
// GET /api/alerts/big-movers?threshold=50&hours=24
type BigMoverAlert struct {
    EventSlug         string  `json:"event_slug"`
    Match            string  `json:"match"`
    Market           string  `json:"market"`
    Outcome          string  `json:"outcome"`
    OpeningOdds      float64 `json:"opening_odds"`
    CurrentOdds      float64 `json:"current_odds"`
    ChangePercentage float64 `json:"change_percentage"`
    Multiplier       float64 `json:"multiplier"`
    Direction        string  `json:"direction"` // DRIFTING or SHORTENING
}
```

### Movement History Endpoint
```go
// GET /api/events/{slug}/movements
type OddsMovement struct {
    Timestamp        time.Time `json:"timestamp"`
    Market          string    `json:"market"`
    Outcome         string    `json:"outcome"`
    OldValue        float64   `json:"old_value"`
    NewValue        float64   `json:"new_value"`
    ChangePercent   float64   `json:"change_percent"`
}
```

## Performance Optimizations

1. **Indexed Searches**: 
   - `idx_current_odds_movement_pct` - Fast lookup by movement percentage
   - `idx_odds_history_big_changes` - Pre-filtered index for > 20% changes
   - `idx_odds_history_multiplier` - Pre-filtered index for > 1.5x multipliers

2. **Generated Columns**:
   - Movement calculations are done at insert time, not query time
   - No need for complex window functions in queries

3. **Efficient Storage**:
   - Only store odds when they change
   - Current state readily available without aggregation

## Example Scenario

**Team A was favorite at 1.20, now drifted to 3.70:**

1. Initial sync creates:
   ```sql
   INSERT INTO current_odds (event_id, market_type_id, outcome, odds_value, opening_value, highest_value, lowest_value)
   VALUES (123, 1, '1', 1.20, 1.20, 1.20, 1.20);
   ```

2. 30 minutes later, odds change to 3.70:
   ```sql
   -- Insert into history
   INSERT INTO odds_history (event_id, market_type_id, outcome, odds_value, previous_value)
   VALUES (123, 1, '1', 3.70, 1.20);
   -- Auto-calculates: change_percentage = 208.33%, multiplier = 3.08
   
   -- Update current
   UPDATE current_odds 
   SET odds_value = 3.70, 
       highest_value = 3.70,
       last_updated = NOW()
   WHERE event_id = 123 AND market_type_id = 1 AND outcome = '1';
   -- Auto-calculates: movement_percentage = 208.33%
   ```

3. Query finds it instantly:
   ```sql
   SELECT * FROM big_movers;  -- Shows this event at the top
   ```

This design makes finding big movements extremely fast and efficient!