-- name: UpdateEventVolume :one
UPDATE events 
SET 
    betting_volume_percentage = sqlc.arg(betting_volume_percentage),
    volume_rank = sqlc.arg(volume_rank),
    volume_updated_at = sqlc.arg(volume_updated_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: CreateVolumeHistory :one
INSERT INTO betting_volume_history (
    event_id,
    volume_percentage,
    rank_position,
    total_events_tracked
) VALUES (
    sqlc.arg(event_id),
    sqlc.arg(volume_percentage),
    sqlc.arg(rank_position),
    sqlc.arg(total_events_tracked)
)
RETURNING *;

-- name: GetHotMovers :many
-- Find events with high betting volume AND significant odds movement
SELECT 
    e.slug,
    ht.name || ' vs ' || at.name as match_name,
    e.betting_volume_percentage,
    e.volume_rank,
    COALESCE(MAX(ABS(co.movement_percentage)), 0) as max_movement,
    CASE 
        WHEN e.betting_volume_percentage > 5 THEN 'HOT'
        WHEN e.betting_volume_percentage > 2 THEN 'POPULAR'
        WHEN e.betting_volume_percentage > 1 THEN 'MODERATE'
        ELSE 'COLD'
    END as popularity_level,
    CASE 
        WHEN e.betting_volume_percentage > 5 AND MAX(ABS(co.movement_percentage)) > 50 THEN 'HOT_MOVER'
        WHEN e.betting_volume_percentage < 1 AND MAX(ABS(co.movement_percentage)) > 50 THEN 'HIDDEN_GEM'
        WHEN e.betting_volume_percentage > 5 AND MAX(ABS(co.movement_percentage)) < 10 THEN 'STABLE_FAVORITE'
        ELSE 'NORMAL'
    END as event_type
FROM events e
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
LEFT JOIN current_odds co ON co.event_id = e.id
WHERE e.betting_volume_percentage >= sqlc.arg(min_volume)
  AND e.event_date > CURRENT_TIMESTAMP
GROUP BY e.id, e.slug, ht.name, at.name, e.betting_volume_percentage, e.volume_rank
HAVING COALESCE(MAX(ABS(co.movement_percentage)), 0) >= sqlc.arg(min_movement)
ORDER BY e.betting_volume_percentage DESC
LIMIT 50;

-- name: GetHiddenGems :many
-- Find low-volume events with big movements (potential sharp money)
SELECT 
    e.slug,
    ht.name || ' vs ' || at.name as match_name,
    e.betting_volume_percentage,
    COALESCE(MAX(ABS(co.movement_percentage)), 0) as max_movement
FROM events e
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
LEFT JOIN current_odds co ON co.event_id = e.id
WHERE e.betting_volume_percentage <= sqlc.arg(max_volume)
  AND e.betting_volume_percentage > 0
  AND e.event_date > CURRENT_TIMESTAMP
GROUP BY e.id, e.slug, ht.name, at.name, e.betting_volume_percentage
HAVING COALESCE(MAX(ABS(co.movement_percentage)), 0) >= sqlc.arg(min_movement)
ORDER BY MAX(ABS(co.movement_percentage)) DESC
LIMIT 20;

-- name: GetVolumeHistory :many
-- Get volume history for a specific event
SELECT 
    bvh.*,
    LAG(bvh.volume_percentage) OVER (ORDER BY bvh.recorded_at) as previous_volume,
    bvh.volume_percentage - LAG(bvh.volume_percentage) OVER (ORDER BY bvh.recorded_at) as volume_change
FROM betting_volume_history bvh
WHERE bvh.event_id = sqlc.arg(event_id)
ORDER BY bvh.recorded_at DESC
LIMIT 100;

-- name: GetTopVolumeEvents :many
-- Get current top events by betting volume
SELECT 
    e.slug,
    ht.name || ' vs ' || at.name as match_name,
    e.event_date,
    e.betting_volume_percentage,
    e.volume_rank,
    COUNT(DISTINCT oh.id) as total_odds_changes,
    COALESCE(MAX(ABS(co.movement_percentage)), 0) as max_movement
FROM events e
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
LEFT JOIN current_odds co ON co.event_id = e.id
LEFT JOIN odds_history oh ON oh.event_id = e.id
WHERE e.volume_rank <= 20
  AND e.event_date > CURRENT_TIMESTAMP
GROUP BY e.id, e.slug, ht.name, at.name, e.event_date, e.betting_volume_percentage, e.volume_rank
ORDER BY e.volume_rank;

-- name: AnalyzeVolumeOddsPattern :one
-- Analyze correlation between volume and odds movement
WITH patterns AS (
    SELECT 
        CASE 
            WHEN e.betting_volume_percentage > 5 AND MAX(ABS(co.movement_percentage)) > 30 THEN 'HIGH_VOLUME_HIGH_MOVEMENT'
            WHEN e.betting_volume_percentage > 5 AND MAX(ABS(co.movement_percentage)) < 10 THEN 'HIGH_VOLUME_STABLE'
            WHEN e.betting_volume_percentage < 1 AND MAX(ABS(co.movement_percentage)) > 30 THEN 'LOW_VOLUME_HIGH_MOVEMENT'
            WHEN e.betting_volume_percentage < 1 AND MAX(ABS(co.movement_percentage)) < 10 THEN 'LOW_VOLUME_STABLE'
            ELSE 'MODERATE'
        END as pattern,
        COUNT(*) as event_count,
        AVG(e.betting_volume_percentage) as avg_volume,
        AVG(MAX(ABS(co.movement_percentage))) as avg_movement
    FROM events e
    LEFT JOIN current_odds co ON co.event_id = e.id
    WHERE e.volume_updated_at > CURRENT_TIMESTAMP - INTERVAL '24 hours'
    GROUP BY e.id, e.betting_volume_percentage
)
SELECT 
    json_object_agg(
        pattern,
        json_build_object(
            'count', event_count,
            'avg_volume', ROUND(avg_volume, 2),
            'avg_movement', ROUND(avg_movement, 2)
        )
    ) as analysis
FROM (
    SELECT 
        pattern,
        SUM(event_count) as event_count,
        AVG(avg_volume) as avg_volume,
        AVG(avg_movement) as avg_movement
    FROM patterns
    GROUP BY pattern
) summary;