-- name: GetSport :one
SELECT * FROM sports WHERE id = sqlc.arg(id);

-- name: ListSports :many
SELECT * FROM sports ORDER BY id;

-- name: UpsertSport :one
INSERT INTO sports (
    id, 
    name, 
    code, 
    slug,
    live_count,
    upcoming_count,
    events_count,
    odds_count,
    has_results,
    has_king_odd,
    has_digital_content,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(name),
    sqlc.arg(code),
    sqlc.arg(slug),
    sqlc.arg(live_count),
    sqlc.arg(upcoming_count),
    sqlc.arg(events_count),
    sqlc.arg(odds_count),
    sqlc.arg(has_results),
    sqlc.arg(has_king_odd),
    sqlc.arg(has_digital_content),
    CURRENT_TIMESTAMP
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    code = EXCLUDED.code,
    slug = EXCLUDED.slug,
    live_count = EXCLUDED.live_count,
    upcoming_count = EXCLUDED.upcoming_count,
    events_count = EXCLUDED.events_count,
    odds_count = EXCLUDED.odds_count,
    has_results = EXCLUDED.has_results,
    has_king_odd = EXCLUDED.has_king_odd,
    has_digital_content = EXCLUDED.has_digital_content,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpdateSport :one
UPDATE sports SET
    name = sqlc.arg(name),
    code = sqlc.arg(code),
    slug = sqlc.arg(slug),
    live_count = sqlc.arg(live_count),
    upcoming_count = sqlc.arg(upcoming_count),
    events_count = sqlc.arg(events_count),
    odds_count = sqlc.arg(odds_count),
    has_results = sqlc.arg(has_results),
    has_king_odd = sqlc.arg(has_king_odd),
    has_digital_content = sqlc.arg(has_digital_content),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: BulkUpsertSports :execrows
INSERT INTO sports (
    id,
    name,
    code,
    slug,
    live_count,
    upcoming_count,
    events_count,
    odds_count,
    has_results,
    has_king_odd,
    has_digital_content,
    updated_at
) VALUES (
    unnest(sqlc.arg(ids)::int[]),
    unnest(sqlc.arg(names)::text[]),
    unnest(sqlc.arg(codes)::text[]),
    unnest(sqlc.arg(slugs)::text[]),
    unnest(sqlc.arg(live_counts)::int[]),
    unnest(sqlc.arg(upcoming_counts)::int[]),
    unnest(sqlc.arg(events_counts)::int[]),
    unnest(sqlc.arg(odds_counts)::int[]),
    unnest(sqlc.arg(has_results)::boolean[]),
    unnest(sqlc.arg(has_king_odds)::boolean[]),
    unnest(sqlc.arg(has_digital_contents)::boolean[]),
    CURRENT_TIMESTAMP
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    code = EXCLUDED.code,
    slug = EXCLUDED.slug,
    live_count = EXCLUDED.live_count,
    upcoming_count = EXCLUDED.upcoming_count,
    events_count = EXCLUDED.events_count,
    odds_count = EXCLUDED.odds_count,
    has_results = EXCLUDED.has_results,
    has_king_odd = EXCLUDED.has_king_odd,
    has_digital_content = EXCLUDED.has_digital_content,
    updated_at = CURRENT_TIMESTAMP;