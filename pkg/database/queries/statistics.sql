-- name: UpdateEventLiveData :one
UPDATE events 
SET 
    is_live = sqlc.arg(is_live),
    status = sqlc.arg(status),
    home_score = sqlc.arg(home_score),
    away_score = sqlc.arg(away_score),
    minute_of_match = sqlc.arg(minute_of_match),
    half = sqlc.arg(half),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpsertMatchStatistics :one
INSERT INTO match_statistics (
    event_id, is_home, shots, shots_on_target, possession, corners,
    yellow_cards, red_cards, fouls, offsides, free_kicks, throw_ins, 
    goal_kicks, saves
)
VALUES (
    sqlc.arg(event_id), sqlc.arg(is_home), sqlc.arg(shots), sqlc.arg(shots_on_target),
    sqlc.arg(possession), sqlc.arg(corners), sqlc.arg(yellow_cards), sqlc.arg(red_cards),
    sqlc.arg(fouls), sqlc.arg(offsides), sqlc.arg(free_kicks), sqlc.arg(throw_ins),
    sqlc.arg(goal_kicks), sqlc.arg(saves)
)
ON CONFLICT (event_id, is_home) DO UPDATE SET
    shots = EXCLUDED.shots,
    shots_on_target = EXCLUDED.shots_on_target,
    possession = EXCLUDED.possession,
    corners = EXCLUDED.corners,
    yellow_cards = EXCLUDED.yellow_cards,
    red_cards = EXCLUDED.red_cards,
    fouls = EXCLUDED.fouls,
    offsides = EXCLUDED.offsides,
    free_kicks = EXCLUDED.free_kicks,
    throw_ins = EXCLUDED.throw_ins,
    goal_kicks = EXCLUDED.goal_kicks,
    saves = EXCLUDED.saves,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: CreateMatchEvent :one
INSERT INTO match_events (event_id, minute, event_type, team, player, description, is_home)
VALUES (sqlc.arg(event_id), sqlc.arg(minute), sqlc.arg(event_type), sqlc.arg(team), sqlc.arg(player), sqlc.arg(description), sqlc.arg(is_home))
ON CONFLICT (event_id, minute, event_type, team, player) DO NOTHING
RETURNING *;

-- name: GetLiveEvents :many
SELECT 
    e.slug,
    ht.name as home_team,
    at.name as away_team,
    e.home_score,
    e.away_score,
    e.minute_of_match,
    e.half,
    e.status
FROM events e
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
WHERE e.is_live = true
ORDER BY e.event_date ASC;

-- name: GetMatchStatistics :many
SELECT * FROM match_statistics
WHERE event_id = sqlc.arg(event_id)
ORDER BY is_home DESC;

-- name: GetMatchEvents :many
SELECT * FROM match_events
WHERE event_id = sqlc.arg(event_id)
ORDER BY minute ASC, id ASC;

-- name: GetEventStatisticsSummary :one
SELECT 
    e.slug,
    ht.name as home_team,
    at.name as away_team,
    e.home_score,
    e.away_score,
    e.minute_of_match,
    e.half,
    e.status,
    e.is_live,
    (SELECT COUNT(*) FROM match_events me WHERE me.event_id = e.id) as total_events,
    (SELECT COUNT(*) FROM match_statistics ms WHERE ms.event_id = e.id) as has_statistics
FROM events e
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
WHERE e.id = sqlc.arg(event_id);