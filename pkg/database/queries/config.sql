-- name: GetLatestConfig :one
SELECT * FROM app_config 
WHERE platform = sqlc.arg(platform)
ORDER BY updated_at DESC
LIMIT 1;

-- name: UpsertConfig :one
INSERT INTO app_config (platform, config_data, sportoto_program_name, payin_end_date, next_draw_expected_win)
VALUES (sqlc.arg(platform), sqlc.arg(config_data), sqlc.arg(sportoto_program_name), sqlc.arg(payin_end_date), sqlc.arg(next_draw_expected_win))
ON CONFLICT (platform) DO UPDATE SET
    config_data = EXCLUDED.config_data,
    sportoto_program_name = EXCLUDED.sportoto_program_name,
    payin_end_date = EXCLUDED.payin_end_date,
    next_draw_expected_win = EXCLUDED.next_draw_expected_win,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: CreateConfig :one
INSERT INTO app_config (platform, config_data, sportoto_program_name, payin_end_date, next_draw_expected_win)
VALUES (sqlc.arg(platform), sqlc.arg(config_data), sqlc.arg(sportoto_program_name), sqlc.arg(payin_end_date), sqlc.arg(next_draw_expected_win))
RETURNING *;