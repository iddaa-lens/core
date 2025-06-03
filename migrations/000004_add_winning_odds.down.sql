-- Remove winning_odds column from current_odds table
ALTER TABLE current_odds 
DROP COLUMN IF EXISTS winning_odds;

-- Remove winning_odds column from odds_history table
ALTER TABLE odds_history 
DROP COLUMN IF EXISTS winning_odds;

-- Restore the original big_movers view without winning_odds
DROP VIEW IF EXISTS big_movers;
CREATE OR REPLACE VIEW big_movers AS
SELECT 
    e.id as event_id,
    e.slug as event_slug,
    ht.name as home_team,
    at.name as away_team,
    e.event_date,
    e.status,
    mt.code as market_code,
    mt.name as market_name,
    co.outcome,
    co.opening_value,
    co.odds_value as current_value,
    co.highest_value,
    co.lowest_value,
    co.movement_percentage,
    co.odds_value / NULLIF(co.opening_value, 0) as multiplier,
    CASE 
        WHEN co.odds_value > co.opening_value THEN 'DRIFTING'
        WHEN co.odds_value < co.opening_value THEN 'SHORTENING'
        ELSE 'STABLE'
    END as trend_direction,
    co.last_updated
FROM current_odds co
JOIN events e ON co.event_id = e.id
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
JOIN market_types mt ON co.market_type_id = mt.id
WHERE ABS(co.movement_percentage) > 20
   OR co.odds_value / NULLIF(co.opening_value, 0) > 2
   OR co.opening_value / NULLIF(co.odds_value, 0) > 2
ORDER BY ABS(co.movement_percentage) DESC;