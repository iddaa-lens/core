# Odds Tracking Design Decision

## Recommended Approach: Hybrid Model with Two Tables

### 1. Current Odds Table (Fast Lookups)
```sql
CREATE TABLE current_odds (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id),
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(100),
    odds_value DECIMAL(10, 3),
    last_updated TIMESTAMP,
    UNIQUE(event_id, market_type_id, outcome)
);
```

### 2. Odds History Table (Changes Only)
```sql
CREATE TABLE odds_history (
    id SERIAL PRIMARY KEY,
    event_id INTEGER REFERENCES events(id),
    market_type_id INTEGER REFERENCES market_types(id),
    outcome VARCHAR(100),
    odds_value DECIMAL(10, 3),
    previous_value DECIMAL(10, 3),
    change_amount DECIMAL(10, 3),
    change_percentage DECIMAL(5, 2),
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_odds_history_event_time (event_id, recorded_at DESC)
);
```

### Benefits of This Hybrid Approach

1. **Performance**:
   - Current odds lookups are O(1) with unique constraint
   - No need to scan history for latest values
   - Smaller table for real-time queries

2. **Storage Efficiency**:
   - Only store changes in history
   - Current table stays small (one row per event/market/outcome)
   - History grows only when odds actually change

3. **Query Simplicity**:
   - Getting current odds: `SELECT * FROM current_odds WHERE event_id = ?`
   - Getting changes: `SELECT * FROM odds_history WHERE event_id = ? ORDER BY recorded_at`
   - No complex window functions for basic queries

4. **Analytics Ready**:
   - Pre-calculated change amounts and percentages
   - Easy to query trends and patterns
   - Can aggregate changes without recalculation

### Implementation Logic

```go
func (s *EventsService) processOdds(ctx context.Context, eventID int, marketTypeID int32, outcome string, newOdds float64) error {
    // 1. Get current odds
    current, err := s.db.GetCurrentOdds(ctx, eventID, marketTypeID, outcome)
    
    // 2. If odds changed or new
    if err == sql.ErrNoRows || math.Abs(current.OddsValue - newOdds) > 0.001 {
        // 3. Insert into history if changed
        if err != sql.ErrNoRows && current.OddsValue != newOdds {
            s.db.InsertOddsHistory(ctx, OddsHistoryParams{
                EventID:          eventID,
                MarketTypeID:     marketTypeID,
                Outcome:          outcome,
                OddsValue:        newOdds,
                PreviousValue:    current.OddsValue,
                ChangeAmount:     newOdds - current.OddsValue,
                ChangePercentage: ((newOdds - current.OddsValue) / current.OddsValue) * 100,
            })
        }
        
        // 4. Upsert current odds
        s.db.UpsertCurrentOdds(ctx, CurrentOddsParams{
            EventID:      eventID,
            MarketTypeID: marketTypeID,
            Outcome:      outcome,
            OddsValue:    newOdds,
        })
    }
}
```

### Sample Queries

```sql
-- Get current odds for an event
SELECT * FROM current_odds WHERE event_id = 123;

-- Get odds movement history
SELECT * FROM odds_history 
WHERE event_id = 123 
ORDER BY recorded_at DESC;

-- Get biggest movers in last hour
SELECT 
    e.slug,
    oh.outcome,
    oh.change_percentage,
    oh.odds_value,
    oh.previous_value
FROM odds_history oh
JOIN events e ON oh.event_id = e.id
WHERE oh.recorded_at > NOW() - INTERVAL '1 hour'
ORDER BY ABS(oh.change_percentage) DESC
LIMIT 20;

-- Get volatility score for events
SELECT 
    event_id,
    COUNT(*) as number_of_changes,
    SUM(ABS(change_percentage)) as total_movement,
    AVG(ABS(change_percentage)) as avg_movement
FROM odds_history
WHERE recorded_at > NOW() - INTERVAL '24 hours'
GROUP BY event_id
ORDER BY total_movement DESC;
```

## Recommendation

**Use the hybrid two-table approach** because:

1. Your sync runs every 30 minutes - you want to efficiently detect and store only changes
2. You'll likely have many more reads than writes
3. Current odds queries should be lightning fast for API/frontend
4. Historical analysis queries can afford to be slightly slower
5. Pre-calculated changes make analytics much easier

This design scales well and provides the best balance of performance, storage efficiency, and query simplicity.