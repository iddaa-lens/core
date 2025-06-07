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
LEFT JOIN leagues l ON e.league_id = l.id
WHERE e.external_id = sqlc.arg(external_id);

-- name: GetEventByExternalIDSimple :one
SELECT * FROM events WHERE external_id = sqlc.arg(external_id);

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

-- name: GetActiveEventsForDetailedSync :many
SELECT * FROM events 
WHERE (status = 'live' OR status = 'scheduled') 
  AND event_date >= NOW() - INTERVAL '2 hours'
  AND event_date <= NOW() + INTERVAL '24 hours'
ORDER BY 
  CASE WHEN status = 'live' THEN 1 ELSE 2 END,
  event_date ASC
LIMIT sqlc.arg(limit_count);

-- name: UpsertEvent :one
INSERT INTO events (external_id, league_id, home_team_id, away_team_id, event_date, status, home_score, away_score, 
                   bulletin_id, version, sport_id, bet_program, mbc, has_king_odd, odds_count, has_combine, is_live)
VALUES (sqlc.arg(external_id), sqlc.arg(league_id), sqlc.arg(home_team_id), sqlc.arg(away_team_id), sqlc.arg(event_date), 
        sqlc.arg(status), sqlc.arg(home_score), sqlc.arg(away_score), sqlc.arg(bulletin_id), sqlc.arg(version), 
        sqlc.arg(sport_id), sqlc.arg(bet_program), sqlc.arg(mbc), sqlc.arg(has_king_odd), sqlc.arg(odds_count), 
        sqlc.arg(has_combine), sqlc.arg(is_live))
ON CONFLICT (external_id) DO UPDATE SET
    league_id = EXCLUDED.league_id,
    home_team_id = EXCLUDED.home_team_id,
    away_team_id = EXCLUDED.away_team_id,
    event_date = EXCLUDED.event_date,
    status = EXCLUDED.status,
    home_score = EXCLUDED.home_score,
    away_score = EXCLUDED.away_score,
    bulletin_id = EXCLUDED.bulletin_id,
    version = EXCLUDED.version,
    sport_id = EXCLUDED.sport_id,
    bet_program = EXCLUDED.bet_program,
    mbc = EXCLUDED.mbc,
    has_king_odd = EXCLUDED.has_king_odd,
    odds_count = EXCLUDED.odds_count,
    has_combine = EXCLUDED.has_combine,
    is_live = EXCLUDED.is_live,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: ListEventsFiltered :many
SELECT e.id, e.external_id, e.league_id, e.home_team_id, e.away_team_id, e.slug, e.event_date, e.status, e.home_score, e.away_score, e.is_live, e.minute_of_match, e.half, e.betting_volume_percentage, e.volume_rank, e.volume_updated_at, e.bulletin_id, e.version, e.sport_id, e.bet_program, e.mbc, e.has_king_odd, e.odds_count, e.has_combine, e.created_at, e.updated_at,
       ht.name as home_team_name,
       ht.country as home_team_country,
       at.name as away_team_name,
       at.country as away_team_country,
       l.name as league_name,
       l.country as league_country,
       s.name as sport_name,
       s.code as sport_code
FROM events e
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
JOIN leagues l ON e.league_id = l.id
JOIN sports s ON e.sport_id = s.id
WHERE e.event_date >= sqlc.arg(time_after)
  AND e.event_date <= sqlc.arg(time_before)
  AND (sqlc.arg(sport_code) = '' OR s.code = sqlc.arg(sport_code))
  AND (sqlc.arg(league_name) = '' OR l.name ILIKE '%' || sqlc.arg(league_name) || '%')
  AND (sqlc.arg(status) = '' OR e.status = sqlc.arg(status))
ORDER BY e.event_date ASC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountEventsFiltered :one
SELECT COUNT(*)
FROM events e
JOIN teams ht ON e.home_team_id = ht.id
JOIN teams at ON e.away_team_id = at.id
JOIN leagues l ON e.league_id = l.id
JOIN sports s ON e.sport_id = s.id
WHERE e.event_date >= sqlc.arg(time_after)
  AND e.event_date <= sqlc.arg(time_before)
  AND (sqlc.arg(sport_code) = '' OR s.code = sqlc.arg(sport_code))
  AND (sqlc.arg(league_name) = '' OR l.name ILIKE '%' || sqlc.arg(league_name) || '%')
  AND (sqlc.arg(status) = '' OR e.status = sqlc.arg(status));