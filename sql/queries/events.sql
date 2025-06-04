-- name: GetEvent :one
SELECT e.*, 
       ht.name as home_team_name,
       at.name as away_team_name,
       l.name as league_name
FROM events e
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
JOIN leagues l ON e.league_id = l.id
WHERE e.id = sqlc.arg(id);

-- name: GetEventByExternalID :one
SELECT e.*, 
       ht.name as home_team_name,
       at.name as away_team_name,
       l.name as league_name
FROM events e
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
JOIN leagues l ON e.league_id = l.id
WHERE e.external_id = sqlc.arg(external_id);

-- name: ListEventsByDate :many
SELECT e.*, 
       ht.name as home_team_name,
       at.name as away_team_name,
       l.name as league_name
FROM events e
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
JOIN leagues l ON e.league_id = l.id
WHERE DATE(e.event_date) = DATE(sqlc.arg(event_date))
ORDER BY e.event_date;

-- name: CreateEvent :one
INSERT INTO events (external_id, league_id, home_team_id, away_team_id, event_date, status)
VALUES (sqlc.arg(external_id), sqlc.arg(league_id), sqlc.arg(home_team_id), sqlc.arg(away_team_id), sqlc.arg(event_date), sqlc.arg(status))
RETURNING *;

-- name: UpdateEventStatus :one
UPDATE events 
SET status = sqlc.arg(status), home_score = sqlc.arg(home_score), away_score = sqlc.arg(away_score), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpsertEvent :one
INSERT INTO events (external_id, league_id, home_team_id, away_team_id, event_date, status, home_score, away_score)
VALUES (sqlc.arg(external_id), sqlc.arg(league_id), sqlc.arg(home_team_id), sqlc.arg(away_team_id), sqlc.arg(event_date), sqlc.arg(status), sqlc.arg(home_score), sqlc.arg(away_score))
ON CONFLICT (external_id) DO UPDATE SET
    league_id = EXCLUDED.league_id,
    home_team_id = EXCLUDED.home_team_id,
    away_team_id = EXCLUDED.away_team_id,
    event_date = EXCLUDED.event_date,
    status = EXCLUDED.status,
    home_score = EXCLUDED.home_score,
    away_score = EXCLUDED.away_score,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;