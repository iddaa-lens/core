// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: sports.sql

package generated

import (
	"context"
)

const bulkUpsertSports = `-- name: BulkUpsertSports :execrows
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
    unnest($1::int[]),
    unnest($2::text[]),
    unnest($3::text[]),
    unnest($4::text[]),
    unnest($5::int[]),
    unnest($6::int[]),
    unnest($7::int[]),
    unnest($8::int[]),
    unnest($9::boolean[]),
    unnest($10::boolean[]),
    unnest($11::boolean[]),
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
`

type BulkUpsertSportsParams struct {
	Ids                []int32  `db:"ids" json:"ids"`
	Names              []string `db:"names" json:"names"`
	Codes              []string `db:"codes" json:"codes"`
	Slugs              []string `db:"slugs" json:"slugs"`
	LiveCounts         []int32  `db:"live_counts" json:"live_counts"`
	UpcomingCounts     []int32  `db:"upcoming_counts" json:"upcoming_counts"`
	EventsCounts       []int32  `db:"events_counts" json:"events_counts"`
	OddsCounts         []int32  `db:"odds_counts" json:"odds_counts"`
	HasResults         []bool   `db:"has_results" json:"has_results"`
	HasKingOdds        []bool   `db:"has_king_odds" json:"has_king_odds"`
	HasDigitalContents []bool   `db:"has_digital_contents" json:"has_digital_contents"`
}

func (q *Queries) BulkUpsertSports(ctx context.Context, arg BulkUpsertSportsParams) (int64, error) {
	result, err := q.db.Exec(ctx, bulkUpsertSports,
		arg.Ids,
		arg.Names,
		arg.Codes,
		arg.Slugs,
		arg.LiveCounts,
		arg.UpcomingCounts,
		arg.EventsCounts,
		arg.OddsCounts,
		arg.HasResults,
		arg.HasKingOdds,
		arg.HasDigitalContents,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

const getSport = `-- name: GetSport :one
SELECT id, name, code, slug, live_count, upcoming_count, events_count, odds_count, has_results, has_king_odd, has_digital_content, created_at, updated_at FROM sports WHERE id = $1
`

func (q *Queries) GetSport(ctx context.Context, id int32) (Sport, error) {
	row := q.db.QueryRow(ctx, getSport, id)
	var i Sport
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Code,
		&i.Slug,
		&i.LiveCount,
		&i.UpcomingCount,
		&i.EventsCount,
		&i.OddsCount,
		&i.HasResults,
		&i.HasKingOdd,
		&i.HasDigitalContent,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const listSports = `-- name: ListSports :many
SELECT id, name, code, slug, live_count, upcoming_count, events_count, odds_count, has_results, has_king_odd, has_digital_content, created_at, updated_at FROM sports ORDER BY id
`

func (q *Queries) ListSports(ctx context.Context) ([]Sport, error) {
	rows, err := q.db.Query(ctx, listSports)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Sport{}
	for rows.Next() {
		var i Sport
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Code,
			&i.Slug,
			&i.LiveCount,
			&i.UpcomingCount,
			&i.EventsCount,
			&i.OddsCount,
			&i.HasResults,
			&i.HasKingOdd,
			&i.HasDigitalContent,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateSport = `-- name: UpdateSport :one
UPDATE sports SET
    name = $1,
    code = $2,
    slug = $3,
    live_count = $4,
    upcoming_count = $5,
    events_count = $6,
    odds_count = $7,
    has_results = $8,
    has_king_odd = $9,
    has_digital_content = $10,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $11
RETURNING id, name, code, slug, live_count, upcoming_count, events_count, odds_count, has_results, has_king_odd, has_digital_content, created_at, updated_at
`

type UpdateSportParams struct {
	Name              string `db:"name" json:"name"`
	Code              string `db:"code" json:"code"`
	Slug              string `db:"slug" json:"slug"`
	LiveCount         *int32 `db:"live_count" json:"live_count"`
	UpcomingCount     *int32 `db:"upcoming_count" json:"upcoming_count"`
	EventsCount       *int32 `db:"events_count" json:"events_count"`
	OddsCount         *int32 `db:"odds_count" json:"odds_count"`
	HasResults        *bool  `db:"has_results" json:"has_results"`
	HasKingOdd        *bool  `db:"has_king_odd" json:"has_king_odd"`
	HasDigitalContent *bool  `db:"has_digital_content" json:"has_digital_content"`
	ID                int32  `db:"id" json:"id"`
}

func (q *Queries) UpdateSport(ctx context.Context, arg UpdateSportParams) (Sport, error) {
	row := q.db.QueryRow(ctx, updateSport,
		arg.Name,
		arg.Code,
		arg.Slug,
		arg.LiveCount,
		arg.UpcomingCount,
		arg.EventsCount,
		arg.OddsCount,
		arg.HasResults,
		arg.HasKingOdd,
		arg.HasDigitalContent,
		arg.ID,
	)
	var i Sport
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Code,
		&i.Slug,
		&i.LiveCount,
		&i.UpcomingCount,
		&i.EventsCount,
		&i.OddsCount,
		&i.HasResults,
		&i.HasKingOdd,
		&i.HasDigitalContent,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const upsertSport = `-- name: UpsertSport :one
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
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
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
RETURNING id, name, code, slug, live_count, upcoming_count, events_count, odds_count, has_results, has_king_odd, has_digital_content, created_at, updated_at
`

type UpsertSportParams struct {
	ID                int32  `db:"id" json:"id"`
	Name              string `db:"name" json:"name"`
	Code              string `db:"code" json:"code"`
	Slug              string `db:"slug" json:"slug"`
	LiveCount         *int32 `db:"live_count" json:"live_count"`
	UpcomingCount     *int32 `db:"upcoming_count" json:"upcoming_count"`
	EventsCount       *int32 `db:"events_count" json:"events_count"`
	OddsCount         *int32 `db:"odds_count" json:"odds_count"`
	HasResults        *bool  `db:"has_results" json:"has_results"`
	HasKingOdd        *bool  `db:"has_king_odd" json:"has_king_odd"`
	HasDigitalContent *bool  `db:"has_digital_content" json:"has_digital_content"`
}

func (q *Queries) UpsertSport(ctx context.Context, arg UpsertSportParams) (Sport, error) {
	row := q.db.QueryRow(ctx, upsertSport,
		arg.ID,
		arg.Name,
		arg.Code,
		arg.Slug,
		arg.LiveCount,
		arg.UpcomingCount,
		arg.EventsCount,
		arg.OddsCount,
		arg.HasResults,
		arg.HasKingOdd,
		arg.HasDigitalContent,
	)
	var i Sport
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Code,
		&i.Slug,
		&i.LiveCount,
		&i.UpcomingCount,
		&i.EventsCount,
		&i.OddsCount,
		&i.HasResults,
		&i.HasKingOdd,
		&i.HasDigitalContent,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
