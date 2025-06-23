-- name: GetOddsHistory :many
-- Get full odds history for a specific event
SELECT 
    oh.*,
    mt.code as market_code,
    mt.name as market_name
FROM odds_history oh
JOIN market_types mt ON oh.market_type_id = mt.id
WHERE oh.event_id = sqlc.arg(event_id)
ORDER BY oh.market_type_id, oh.outcome, oh.recorded_at DESC;

-- name: GetOddsChangesByMarket :many
-- Get odds changes for a specific market
SELECT 
    oh.*,
    mt.code as market_code,
    mt.name as market_name
FROM odds_history oh
JOIN market_types mt ON oh.market_type_id = mt.id
WHERE oh.event_id = sqlc.arg(event_id)
AND oh.market_type_id = sqlc.arg(market_type_id)
AND ABS(oh.change_percentage) > sqlc.arg(min_change_percentage)::float8
ORDER BY oh.recorded_at DESC;

-- name: GetRecentMovements :many
-- Get recent significant odds movements across all events
SELECT 
    oh.*,
    e.slug as event_slug,
    e.event_date,
    e.status as event_status,
    e.is_live,
    e.home_score,
    e.away_score,
    e.minute_of_match,
    e.betting_volume_percentage,
    mt.code as market_code,
    mt.name as market_name,
    mt.description as market_description,
    ht.name as home_team,
    ht.country as home_team_country,
    at.name as away_team,
    at.country as away_team_country,
    l.name as league_name,
    l.country as league_country,
    s.name as sport_name,
    s.code as sport_code
FROM odds_history oh
JOIN events e ON oh.event_id = e.id
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
JOIN market_types mt ON oh.market_type_id = mt.id
JOIN leagues l ON e.league_id = l.id
JOIN sports s ON e.sport_id = s.id
WHERE oh.recorded_at > sqlc.arg(since_time)
AND ABS(oh.change_percentage) > sqlc.arg(min_change_percentage)::float8
ORDER BY oh.recorded_at DESC
LIMIT sqlc.arg(limit_count);

-- name: GetSuspiciousMovements :many
-- Get potentially suspicious odds movements (sharp money indicators)
SELECT 
    oh.*,
    e.slug as event_slug,
    mt.code as market_code
FROM odds_history oh
JOIN events e ON oh.event_id = e.id
JOIN market_types mt ON oh.market_type_id = mt.id
WHERE oh.multiplier > 1.5 OR oh.multiplier < 0.67
AND oh.recorded_at > sqlc.arg(since_time)
ORDER BY CASE 
    WHEN oh.multiplier > 1.0 THEN oh.multiplier
    ELSE (1.0 / oh.multiplier)
END DESC
LIMIT sqlc.arg(limit_count);