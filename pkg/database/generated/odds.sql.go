// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: odds.sql

package generated

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const batchGetCurrentOdds = `-- name: BatchGetCurrentOdds :many
SELECT
    id, event_id, market_type_id, outcome, odds_value, opening_value, highest_value, lowest_value, winning_odds, total_movement, movement_percentage, last_updated, market_params
FROM
    current_odds
WHERE
    event_id = $1::int
    AND market_type_id = ANY($2::int)
    AND outcome = ANY($3::text[])
`

type BatchGetCurrentOddsParams struct {
	EventID       int32    `db:"event_id" json:"event_id"`
	MarketTypeIds int32    `db:"market_type_ids" json:"market_type_ids"`
	Outcomes      []string `db:"outcomes" json:"outcomes"`
}

func (q *Queries) BatchGetCurrentOdds(ctx context.Context, arg BatchGetCurrentOddsParams) ([]CurrentOdd, error) {
	rows, err := q.db.Query(ctx, batchGetCurrentOdds, arg.EventID, arg.MarketTypeIds, arg.Outcomes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []CurrentOdd{}
	for rows.Next() {
		var i CurrentOdd
		if err := rows.Scan(
			&i.ID,
			&i.EventID,
			&i.MarketTypeID,
			&i.Outcome,
			&i.OddsValue,
			&i.OpeningValue,
			&i.HighestValue,
			&i.LowestValue,
			&i.WinningOdds,
			&i.TotalMovement,
			&i.MovementPercentage,
			&i.LastUpdated,
			&i.MarketParams,
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

const bulkGetCurrentOddsForComparison = `-- name: BulkGetCurrentOddsForComparison :many
SELECT
    co.event_id,
    co.market_type_id,
    co.outcome,
    co.odds_value,
    co.opening_value,
    co.highest_value,
    co.lowest_value,
    e.event_date
FROM
    current_odds co
    JOIN events e ON e.id = co.event_id
WHERE
    (co.event_id, co.market_type_id, co.outcome) IN (
        SELECT
            unnest($1::int[]),
            unnest($2::int[]),
            unnest($3::text[])
    )
`

type BulkGetCurrentOddsForComparisonParams struct {
	EventIds      []int32  `db:"event_ids" json:"event_ids"`
	MarketTypeIds []int32  `db:"market_type_ids" json:"market_type_ids"`
	Outcomes      []string `db:"outcomes" json:"outcomes"`
}

type BulkGetCurrentOddsForComparisonRow struct {
	EventID      *int32           `db:"event_id" json:"event_id"`
	MarketTypeID *int32           `db:"market_type_id" json:"market_type_id"`
	Outcome      string           `db:"outcome" json:"outcome"`
	OddsValue    float64          `db:"odds_value" json:"odds_value"`
	OpeningValue *float64         `db:"opening_value" json:"opening_value"`
	HighestValue *float64         `db:"highest_value" json:"highest_value"`
	LowestValue  *float64         `db:"lowest_value" json:"lowest_value"`
	EventDate    pgtype.Timestamp `db:"event_date" json:"event_date"`
}

// Helper query to get current odds for comparison
func (q *Queries) BulkGetCurrentOddsForComparison(ctx context.Context, arg BulkGetCurrentOddsForComparisonParams) ([]BulkGetCurrentOddsForComparisonRow, error) {
	rows, err := q.db.Query(ctx, bulkGetCurrentOddsForComparison, arg.EventIds, arg.MarketTypeIds, arg.Outcomes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []BulkGetCurrentOddsForComparisonRow{}
	for rows.Next() {
		var i BulkGetCurrentOddsForComparisonRow
		if err := rows.Scan(
			&i.EventID,
			&i.MarketTypeID,
			&i.Outcome,
			&i.OddsValue,
			&i.OpeningValue,
			&i.HighestValue,
			&i.LowestValue,
			&i.EventDate,
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

const bulkInsertOddsHistory = `-- name: BulkInsertOddsHistory :exec
WITH input_data AS (
    SELECT
        unnest($1::int[]) as event_id,
        unnest($2::int[]) as market_type_id,
        unnest($3::text[]) as outcome,
        unnest($4::float8[]) as odds_value,
        unnest($5::float8[]) as previous_value,
        unnest($6::float8[]) as change_amount,
        unnest($7::float8[]) as change_percentage,
        unnest($8::float8[]) as multiplier,
        unnest($9::boolean[]) as is_reverse_movement,
        unnest($10::text[]) as significance_level,
        unnest($11::int[]) as minutes_to_kickoff,
        unnest($12::jsonb[]) as market_params
)
INSERT INTO
    odds_history (
        event_id,
        market_type_id,
        outcome,
        odds_value,
        previous_value,
        change_amount,
        change_percentage,
        multiplier,
        is_reverse_movement,
        significance_level,
        minutes_to_kickoff,
        market_params,
        recorded_at
    )
SELECT
    event_id,
    market_type_id,
    outcome,
    odds_value,
    previous_value,
    change_amount,
    change_percentage,
    multiplier,
    is_reverse_movement,
    significance_level,
    minutes_to_kickoff,
    market_params,
    NOW()
FROM
    input_data
`

type BulkInsertOddsHistoryParams struct {
	EventIds           []int32   `db:"event_ids" json:"event_ids"`
	MarketTypeIds      []int32   `db:"market_type_ids" json:"market_type_ids"`
	Outcomes           []string  `db:"outcomes" json:"outcomes"`
	OddsValues         []float64 `db:"odds_values" json:"odds_values"`
	PreviousValues     []float64 `db:"previous_values" json:"previous_values"`
	ChangeAmounts      []float64 `db:"change_amounts" json:"change_amounts"`
	ChangePercentages  []float64 `db:"change_percentages" json:"change_percentages"`
	Multipliers        []float64 `db:"multipliers" json:"multipliers"`
	IsReverseMovements []bool    `db:"is_reverse_movements" json:"is_reverse_movements"`
	SignificanceLevels []string  `db:"significance_levels" json:"significance_levels"`
	MinutesToKickoffs  []int32   `db:"minutes_to_kickoffs" json:"minutes_to_kickoffs"`
	MarketParams       [][]byte  `db:"market_params" json:"market_params"`
}

func (q *Queries) BulkInsertOddsHistory(ctx context.Context, arg BulkInsertOddsHistoryParams) error {
	_, err := q.db.Exec(ctx, bulkInsertOddsHistory,
		arg.EventIds,
		arg.MarketTypeIds,
		arg.Outcomes,
		arg.OddsValues,
		arg.PreviousValues,
		arg.ChangeAmounts,
		arg.ChangePercentages,
		arg.Multipliers,
		arg.IsReverseMovements,
		arg.SignificanceLevels,
		arg.MinutesToKickoffs,
		arg.MarketParams,
	)
	return err
}

const bulkUpsertCurrentOdds = `-- name: BulkUpsertCurrentOdds :exec
WITH input_data AS (
    SELECT
        unnest($1::int[]) as event_id,
        unnest($2::int[]) as market_type_id,
        unnest($3::text[]) as outcome,
        unnest($4::float8[]) as odds_value,
        unnest($5::jsonb[]) as market_params
)
INSERT INTO
    current_odds (
        event_id,
        market_type_id,
        outcome,
        odds_value,
        opening_value,
        highest_value,
        lowest_value,
        market_params,
        last_updated
    )
SELECT
    event_id,
    market_type_id,
    outcome,
    odds_value,
    odds_value,
    odds_value,
    odds_value,
    market_params,
    NOW()
FROM
    input_data ON CONFLICT (event_id, market_type_id, outcome) DO
UPDATE
SET
    odds_value = EXCLUDED.odds_value,
    highest_value = GREATEST(current_odds.highest_value, EXCLUDED.odds_value),
    lowest_value = LEAST(current_odds.lowest_value, EXCLUDED.odds_value),
    last_updated = NOW()
WHERE
    current_odds.odds_value IS DISTINCT
FROM
    EXCLUDED.odds_value
`

type BulkUpsertCurrentOddsParams struct {
	EventIds      []int32   `db:"event_ids" json:"event_ids"`
	MarketTypeIds []int32   `db:"market_type_ids" json:"market_type_ids"`
	Outcomes      []string  `db:"outcomes" json:"outcomes"`
	OddsValues    []float64 `db:"odds_values" json:"odds_values"`
	MarketParams  [][]byte  `db:"market_params" json:"market_params"`
}

func (q *Queries) BulkUpsertCurrentOdds(ctx context.Context, arg BulkUpsertCurrentOddsParams) error {
	_, err := q.db.Exec(ctx, bulkUpsertCurrentOdds,
		arg.EventIds,
		arg.MarketTypeIds,
		arg.Outcomes,
		arg.OddsValues,
		arg.MarketParams,
	)
	return err
}

const createOddsHistory = `-- name: CreateOddsHistory :one
INSERT INTO
    odds_history (
        event_id,
        market_type_id,
        outcome,
        odds_value,
        previous_value,
        winning_odds,
        change_amount,
        change_percentage,
        multiplier,
        market_params
    )
VALUES
    (
        $1::int,
        $2::int,
        $3::text,
        $4::decimal,
        $5::decimal,
        $6::decimal,
        -- Calculate change amount: new_odds - previous_odds
        $4::decimal - $5::decimal,
        -- Calculate change percentage: ((new_odds - previous_odds) / previous_odds) * 100
        CASE
            WHEN $5::decimal > 0 THEN (
                (
                    (
                        $4::decimal - $5::decimal
                    ) / $5::decimal * 100
                )
            )::REAL
            ELSE 0
        END,
        -- Calculate multiplier: new_odds / previous_odds
        CASE
            WHEN $5::decimal > 0 THEN (
                $4::decimal / $5::decimal
            )::DOUBLE PRECISION
            ELSE 1
        END,
        $7::jsonb
    ) RETURNING id, event_id, market_type_id, outcome, odds_value, previous_value, winning_odds, change_amount, change_percentage, multiplier, sharp_money_indicator, is_reverse_movement, significance_level, minutes_to_kickoff, market_params, recorded_at
`

type CreateOddsHistoryParams struct {
	EventID       int32   `db:"event_id" json:"event_id"`
	MarketTypeID  int32   `db:"market_type_id" json:"market_type_id"`
	Outcome       string  `db:"outcome" json:"outcome"`
	OddsValue     float64 `db:"odds_value" json:"odds_value"`
	PreviousValue float64 `db:"previous_value" json:"previous_value"`
	WinningOdds   float64 `db:"winning_odds" json:"winning_odds"`
	MarketParams  []byte  `db:"market_params" json:"market_params"`
}

func (q *Queries) CreateOddsHistory(ctx context.Context, arg CreateOddsHistoryParams) (OddsHistory, error) {
	row := q.db.QueryRow(ctx, createOddsHistory,
		arg.EventID,
		arg.MarketTypeID,
		arg.Outcome,
		arg.OddsValue,
		arg.PreviousValue,
		arg.WinningOdds,
		arg.MarketParams,
	)
	var i OddsHistory
	err := row.Scan(
		&i.ID,
		&i.EventID,
		&i.MarketTypeID,
		&i.Outcome,
		&i.OddsValue,
		&i.PreviousValue,
		&i.WinningOdds,
		&i.ChangeAmount,
		&i.ChangePercentage,
		&i.Multiplier,
		&i.SharpMoneyIndicator,
		&i.IsReverseMovement,
		&i.SignificanceLevel,
		&i.MinutesToKickoff,
		&i.MarketParams,
		&i.RecordedAt,
	)
	return i, err
}

const getBigMovers = `-- name: GetBigMovers :many
SELECT
    oh.id, oh.event_id, oh.market_type_id, oh.outcome, oh.odds_value, oh.previous_value, oh.winning_odds, oh.change_amount, oh.change_percentage, oh.multiplier, oh.sharp_money_indicator, oh.is_reverse_movement, oh.significance_level, oh.minutes_to_kickoff, oh.market_params, oh.recorded_at,
    e.slug as event_slug,
    mt.code as market_code
FROM
    odds_history oh
    JOIN events e ON oh.event_id = e.id
    JOIN market_types mt ON oh.market_type_id = mt.id
WHERE
    ABS(oh.change_percentage) > $1::float8
    AND oh.recorded_at > $2::timestamp
ORDER BY
    ABS(oh.change_percentage) DESC
LIMIT
    $3::int
`

type GetBigMoversParams struct {
	MinChangePct float64          `db:"min_change_pct" json:"min_change_pct"`
	SinceTime    pgtype.Timestamp `db:"since_time" json:"since_time"`
	LimitCount   int32            `db:"limit_count" json:"limit_count"`
}

type GetBigMoversRow struct {
	ID                  int32            `db:"id" json:"id"`
	EventID             *int32           `db:"event_id" json:"event_id"`
	MarketTypeID        *int32           `db:"market_type_id" json:"market_type_id"`
	Outcome             string           `db:"outcome" json:"outcome"`
	OddsValue           float64          `db:"odds_value" json:"odds_value"`
	PreviousValue       *float64         `db:"previous_value" json:"previous_value"`
	WinningOdds         *float64         `db:"winning_odds" json:"winning_odds"`
	ChangeAmount        *float64         `db:"change_amount" json:"change_amount"`
	ChangePercentage    *float32         `db:"change_percentage" json:"change_percentage"`
	Multiplier          *float64         `db:"multiplier" json:"multiplier"`
	SharpMoneyIndicator *float32         `db:"sharp_money_indicator" json:"sharp_money_indicator"`
	IsReverseMovement   *bool            `db:"is_reverse_movement" json:"is_reverse_movement"`
	SignificanceLevel   *string          `db:"significance_level" json:"significance_level"`
	MinutesToKickoff    *int32           `db:"minutes_to_kickoff" json:"minutes_to_kickoff"`
	MarketParams        []byte           `db:"market_params" json:"market_params"`
	RecordedAt          pgtype.Timestamp `db:"recorded_at" json:"recorded_at"`
	EventSlug           string           `db:"event_slug" json:"event_slug"`
	MarketCode          string           `db:"market_code" json:"market_code"`
}

func (q *Queries) GetBigMovers(ctx context.Context, arg GetBigMoversParams) ([]GetBigMoversRow, error) {
	rows, err := q.db.Query(ctx, getBigMovers, arg.MinChangePct, arg.SinceTime, arg.LimitCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetBigMoversRow{}
	for rows.Next() {
		var i GetBigMoversRow
		if err := rows.Scan(
			&i.ID,
			&i.EventID,
			&i.MarketTypeID,
			&i.Outcome,
			&i.OddsValue,
			&i.PreviousValue,
			&i.WinningOdds,
			&i.ChangeAmount,
			&i.ChangePercentage,
			&i.Multiplier,
			&i.SharpMoneyIndicator,
			&i.IsReverseMovement,
			&i.SignificanceLevel,
			&i.MinutesToKickoff,
			&i.MarketParams,
			&i.RecordedAt,
			&i.EventSlug,
			&i.MarketCode,
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

const getCurrentOdds = `-- name: GetCurrentOdds :many
SELECT
    co.id, co.event_id, co.market_type_id, co.outcome, co.odds_value, co.opening_value, co.highest_value, co.lowest_value, co.winning_odds, co.total_movement, co.movement_percentage, co.last_updated, co.market_params,
    mt.name as market_name,
    mt.code as market_code
FROM
    current_odds co
    JOIN market_types mt ON co.market_type_id = mt.id
WHERE
    co.event_id = $1::int
`

type GetCurrentOddsRow struct {
	ID                 int32            `db:"id" json:"id"`
	EventID            *int32           `db:"event_id" json:"event_id"`
	MarketTypeID       *int32           `db:"market_type_id" json:"market_type_id"`
	Outcome            string           `db:"outcome" json:"outcome"`
	OddsValue          float64          `db:"odds_value" json:"odds_value"`
	OpeningValue       *float64         `db:"opening_value" json:"opening_value"`
	HighestValue       *float64         `db:"highest_value" json:"highest_value"`
	LowestValue        *float64         `db:"lowest_value" json:"lowest_value"`
	WinningOdds        *float64         `db:"winning_odds" json:"winning_odds"`
	TotalMovement      *float64         `db:"total_movement" json:"total_movement"`
	MovementPercentage *float32         `db:"movement_percentage" json:"movement_percentage"`
	LastUpdated        pgtype.Timestamp `db:"last_updated" json:"last_updated"`
	MarketParams       []byte           `db:"market_params" json:"market_params"`
	MarketName         string           `db:"market_name" json:"market_name"`
	MarketCode         string           `db:"market_code" json:"market_code"`
}

func (q *Queries) GetCurrentOdds(ctx context.Context, eventID int32) ([]GetCurrentOddsRow, error) {
	rows, err := q.db.Query(ctx, getCurrentOdds, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetCurrentOddsRow{}
	for rows.Next() {
		var i GetCurrentOddsRow
		if err := rows.Scan(
			&i.ID,
			&i.EventID,
			&i.MarketTypeID,
			&i.Outcome,
			&i.OddsValue,
			&i.OpeningValue,
			&i.HighestValue,
			&i.LowestValue,
			&i.WinningOdds,
			&i.TotalMovement,
			&i.MovementPercentage,
			&i.LastUpdated,
			&i.MarketParams,
			&i.MarketName,
			&i.MarketCode,
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

const getCurrentOddsByMarket = `-- name: GetCurrentOddsByMarket :many
SELECT
    co.id, co.event_id, co.market_type_id, co.outcome, co.odds_value, co.opening_value, co.highest_value, co.lowest_value, co.winning_odds, co.total_movement, co.movement_percentage, co.last_updated, co.market_params,
    mt.name as market_name,
    mt.code as market_code
FROM
    current_odds co
    JOIN market_types mt ON co.market_type_id = mt.id
WHERE
    co.event_id = $1::int
    AND co.market_type_id = $2::int
`

type GetCurrentOddsByMarketParams struct {
	EventID      int32 `db:"event_id" json:"event_id"`
	MarketTypeID int32 `db:"market_type_id" json:"market_type_id"`
}

type GetCurrentOddsByMarketRow struct {
	ID                 int32            `db:"id" json:"id"`
	EventID            *int32           `db:"event_id" json:"event_id"`
	MarketTypeID       *int32           `db:"market_type_id" json:"market_type_id"`
	Outcome            string           `db:"outcome" json:"outcome"`
	OddsValue          float64          `db:"odds_value" json:"odds_value"`
	OpeningValue       *float64         `db:"opening_value" json:"opening_value"`
	HighestValue       *float64         `db:"highest_value" json:"highest_value"`
	LowestValue        *float64         `db:"lowest_value" json:"lowest_value"`
	WinningOdds        *float64         `db:"winning_odds" json:"winning_odds"`
	TotalMovement      *float64         `db:"total_movement" json:"total_movement"`
	MovementPercentage *float32         `db:"movement_percentage" json:"movement_percentage"`
	LastUpdated        pgtype.Timestamp `db:"last_updated" json:"last_updated"`
	MarketParams       []byte           `db:"market_params" json:"market_params"`
	MarketName         string           `db:"market_name" json:"market_name"`
	MarketCode         string           `db:"market_code" json:"market_code"`
}

func (q *Queries) GetCurrentOddsByMarket(ctx context.Context, arg GetCurrentOddsByMarketParams) ([]GetCurrentOddsByMarketRow, error) {
	rows, err := q.db.Query(ctx, getCurrentOddsByMarket, arg.EventID, arg.MarketTypeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetCurrentOddsByMarketRow{}
	for rows.Next() {
		var i GetCurrentOddsByMarketRow
		if err := rows.Scan(
			&i.ID,
			&i.EventID,
			&i.MarketTypeID,
			&i.Outcome,
			&i.OddsValue,
			&i.OpeningValue,
			&i.HighestValue,
			&i.LowestValue,
			&i.WinningOdds,
			&i.TotalMovement,
			&i.MovementPercentage,
			&i.LastUpdated,
			&i.MarketParams,
			&i.MarketName,
			&i.MarketCode,
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

const getCurrentOddsByOutcome = `-- name: GetCurrentOddsByOutcome :one
SELECT
    co.id, co.event_id, co.market_type_id, co.outcome, co.odds_value, co.opening_value, co.highest_value, co.lowest_value, co.winning_odds, co.total_movement, co.movement_percentage, co.last_updated, co.market_params,
    mt.name as market_name,
    mt.code as market_code
FROM
    current_odds co
    JOIN market_types mt ON co.market_type_id = mt.id
WHERE
    co.event_id = $1::int
    AND co.market_type_id = $2::int
    AND co.outcome = $3::text
`

type GetCurrentOddsByOutcomeParams struct {
	EventID      int32  `db:"event_id" json:"event_id"`
	MarketTypeID int32  `db:"market_type_id" json:"market_type_id"`
	Outcome      string `db:"outcome" json:"outcome"`
}

type GetCurrentOddsByOutcomeRow struct {
	ID                 int32            `db:"id" json:"id"`
	EventID            *int32           `db:"event_id" json:"event_id"`
	MarketTypeID       *int32           `db:"market_type_id" json:"market_type_id"`
	Outcome            string           `db:"outcome" json:"outcome"`
	OddsValue          float64          `db:"odds_value" json:"odds_value"`
	OpeningValue       *float64         `db:"opening_value" json:"opening_value"`
	HighestValue       *float64         `db:"highest_value" json:"highest_value"`
	LowestValue        *float64         `db:"lowest_value" json:"lowest_value"`
	WinningOdds        *float64         `db:"winning_odds" json:"winning_odds"`
	TotalMovement      *float64         `db:"total_movement" json:"total_movement"`
	MovementPercentage *float32         `db:"movement_percentage" json:"movement_percentage"`
	LastUpdated        pgtype.Timestamp `db:"last_updated" json:"last_updated"`
	MarketParams       []byte           `db:"market_params" json:"market_params"`
	MarketName         string           `db:"market_name" json:"market_name"`
	MarketCode         string           `db:"market_code" json:"market_code"`
}

func (q *Queries) GetCurrentOddsByOutcome(ctx context.Context, arg GetCurrentOddsByOutcomeParams) (GetCurrentOddsByOutcomeRow, error) {
	row := q.db.QueryRow(ctx, getCurrentOddsByOutcome, arg.EventID, arg.MarketTypeID, arg.Outcome)
	var i GetCurrentOddsByOutcomeRow
	err := row.Scan(
		&i.ID,
		&i.EventID,
		&i.MarketTypeID,
		&i.Outcome,
		&i.OddsValue,
		&i.OpeningValue,
		&i.HighestValue,
		&i.LowestValue,
		&i.WinningOdds,
		&i.TotalMovement,
		&i.MovementPercentage,
		&i.LastUpdated,
		&i.MarketParams,
		&i.MarketName,
		&i.MarketCode,
	)
	return i, err
}

const getOddsHistoryByID = `-- name: GetOddsHistoryByID :one
SELECT
    id, event_id, market_type_id, outcome, odds_value, previous_value, winning_odds, change_amount, change_percentage, multiplier, sharp_money_indicator, is_reverse_movement, significance_level, minutes_to_kickoff, market_params, recorded_at
FROM
    odds_history
WHERE
    id = $1::bigint
`

func (q *Queries) GetOddsHistoryByID(ctx context.Context, id int64) (OddsHistory, error) {
	row := q.db.QueryRow(ctx, getOddsHistoryByID, id)
	var i OddsHistory
	err := row.Scan(
		&i.ID,
		&i.EventID,
		&i.MarketTypeID,
		&i.Outcome,
		&i.OddsValue,
		&i.PreviousValue,
		&i.WinningOdds,
		&i.ChangeAmount,
		&i.ChangePercentage,
		&i.Multiplier,
		&i.SharpMoneyIndicator,
		&i.IsReverseMovement,
		&i.SignificanceLevel,
		&i.MinutesToKickoff,
		&i.MarketParams,
		&i.RecordedAt,
	)
	return i, err
}

const getOddsMovements = `-- name: GetOddsMovements :many
SELECT
    oh.id, oh.event_id, oh.market_type_id, oh.outcome, oh.odds_value, oh.previous_value, oh.winning_odds, oh.change_amount, oh.change_percentage, oh.multiplier, oh.sharp_money_indicator, oh.is_reverse_movement, oh.significance_level, oh.minutes_to_kickoff, oh.market_params, oh.recorded_at,
    mt.name as market_name,
    mt.code as market_code
FROM
    odds_history oh
    JOIN market_types mt ON oh.market_type_id = mt.id
WHERE
    oh.event_id = $1::int
ORDER BY
    oh.recorded_at DESC
LIMIT
    $2::int
`

type GetOddsMovementsParams struct {
	EventID    int32 `db:"event_id" json:"event_id"`
	LimitCount int32 `db:"limit_count" json:"limit_count"`
}

type GetOddsMovementsRow struct {
	ID                  int32            `db:"id" json:"id"`
	EventID             *int32           `db:"event_id" json:"event_id"`
	MarketTypeID        *int32           `db:"market_type_id" json:"market_type_id"`
	Outcome             string           `db:"outcome" json:"outcome"`
	OddsValue           float64          `db:"odds_value" json:"odds_value"`
	PreviousValue       *float64         `db:"previous_value" json:"previous_value"`
	WinningOdds         *float64         `db:"winning_odds" json:"winning_odds"`
	ChangeAmount        *float64         `db:"change_amount" json:"change_amount"`
	ChangePercentage    *float32         `db:"change_percentage" json:"change_percentage"`
	Multiplier          *float64         `db:"multiplier" json:"multiplier"`
	SharpMoneyIndicator *float32         `db:"sharp_money_indicator" json:"sharp_money_indicator"`
	IsReverseMovement   *bool            `db:"is_reverse_movement" json:"is_reverse_movement"`
	SignificanceLevel   *string          `db:"significance_level" json:"significance_level"`
	MinutesToKickoff    *int32           `db:"minutes_to_kickoff" json:"minutes_to_kickoff"`
	MarketParams        []byte           `db:"market_params" json:"market_params"`
	RecordedAt          pgtype.Timestamp `db:"recorded_at" json:"recorded_at"`
	MarketName          string           `db:"market_name" json:"market_name"`
	MarketCode          string           `db:"market_code" json:"market_code"`
}

func (q *Queries) GetOddsMovements(ctx context.Context, arg GetOddsMovementsParams) ([]GetOddsMovementsRow, error) {
	rows, err := q.db.Query(ctx, getOddsMovements, arg.EventID, arg.LimitCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetOddsMovementsRow{}
	for rows.Next() {
		var i GetOddsMovementsRow
		if err := rows.Scan(
			&i.ID,
			&i.EventID,
			&i.MarketTypeID,
			&i.Outcome,
			&i.OddsValue,
			&i.PreviousValue,
			&i.WinningOdds,
			&i.ChangeAmount,
			&i.ChangePercentage,
			&i.Multiplier,
			&i.SharpMoneyIndicator,
			&i.IsReverseMovement,
			&i.SignificanceLevel,
			&i.MinutesToKickoff,
			&i.MarketParams,
			&i.RecordedAt,
			&i.MarketName,
			&i.MarketCode,
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

const getRecentOddsHistory = `-- name: GetRecentOddsHistory :many
SELECT
    oh.id, oh.event_id, oh.market_type_id, oh.outcome, oh.odds_value, oh.previous_value, oh.winning_odds, oh.change_amount, oh.change_percentage, oh.multiplier, oh.sharp_money_indicator, oh.is_reverse_movement, oh.significance_level, oh.minutes_to_kickoff, oh.market_params, oh.recorded_at,
    e.event_date,
    e.is_live,
    mt.name as market_name,
    mt.code as market_code
FROM
    odds_history oh
    JOIN events e ON oh.event_id = e.id
    JOIN market_types mt ON oh.market_type_id = mt.id
WHERE
    oh.recorded_at >= $1::timestamp
    AND e.event_date > NOW()
    AND ABS(oh.change_percentage) >= $2::float8
ORDER BY
    oh.recorded_at DESC
LIMIT
    $3::int
`

type GetRecentOddsHistoryParams struct {
	SinceTime    pgtype.Timestamp `db:"since_time" json:"since_time"`
	MinChangePct float64          `db:"min_change_pct" json:"min_change_pct"`
	LimitCount   int32            `db:"limit_count" json:"limit_count"`
}

type GetRecentOddsHistoryRow struct {
	ID                  int32            `db:"id" json:"id"`
	EventID             *int32           `db:"event_id" json:"event_id"`
	MarketTypeID        *int32           `db:"market_type_id" json:"market_type_id"`
	Outcome             string           `db:"outcome" json:"outcome"`
	OddsValue           float64          `db:"odds_value" json:"odds_value"`
	PreviousValue       *float64         `db:"previous_value" json:"previous_value"`
	WinningOdds         *float64         `db:"winning_odds" json:"winning_odds"`
	ChangeAmount        *float64         `db:"change_amount" json:"change_amount"`
	ChangePercentage    *float32         `db:"change_percentage" json:"change_percentage"`
	Multiplier          *float64         `db:"multiplier" json:"multiplier"`
	SharpMoneyIndicator *float32         `db:"sharp_money_indicator" json:"sharp_money_indicator"`
	IsReverseMovement   *bool            `db:"is_reverse_movement" json:"is_reverse_movement"`
	SignificanceLevel   *string          `db:"significance_level" json:"significance_level"`
	MinutesToKickoff    *int32           `db:"minutes_to_kickoff" json:"minutes_to_kickoff"`
	MarketParams        []byte           `db:"market_params" json:"market_params"`
	RecordedAt          pgtype.Timestamp `db:"recorded_at" json:"recorded_at"`
	EventDate           pgtype.Timestamp `db:"event_date" json:"event_date"`
	IsLive              *bool            `db:"is_live" json:"is_live"`
	MarketName          string           `db:"market_name" json:"market_name"`
	MarketCode          string           `db:"market_code" json:"market_code"`
}

func (q *Queries) GetRecentOddsHistory(ctx context.Context, arg GetRecentOddsHistoryParams) ([]GetRecentOddsHistoryRow, error) {
	rows, err := q.db.Query(ctx, getRecentOddsHistory, arg.SinceTime, arg.MinChangePct, arg.LimitCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetRecentOddsHistoryRow{}
	for rows.Next() {
		var i GetRecentOddsHistoryRow
		if err := rows.Scan(
			&i.ID,
			&i.EventID,
			&i.MarketTypeID,
			&i.Outcome,
			&i.OddsValue,
			&i.PreviousValue,
			&i.WinningOdds,
			&i.ChangeAmount,
			&i.ChangePercentage,
			&i.Multiplier,
			&i.SharpMoneyIndicator,
			&i.IsReverseMovement,
			&i.SignificanceLevel,
			&i.MinutesToKickoff,
			&i.MarketParams,
			&i.RecordedAt,
			&i.EventDate,
			&i.IsLive,
			&i.MarketName,
			&i.MarketCode,
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

const upsertCurrentOdds = `-- name: UpsertCurrentOdds :one
INSERT INTO
    current_odds (
        event_id,
        market_type_id,
        outcome,
        odds_value,
        opening_value,
        highest_value,
        lowest_value,
        winning_odds,
        total_movement,
        movement_percentage,
        market_params
    )
VALUES
    (
        $1::int,
        $2::int,
        $3::text,
        $4::decimal,
        $5::decimal,
        $6::decimal,
        $7::decimal,
        $8::decimal,
        0,
        -- First time, no movement
        0,
        -- First time, no movement percentage
        $9::jsonb
    ) ON CONFLICT (event_id, market_type_id, outcome) DO
UPDATE
SET
    odds_value = EXCLUDED.odds_value,
    winning_odds = EXCLUDED.winning_odds,
    highest_value = GREATEST(current_odds.highest_value, EXCLUDED.odds_value),
    lowest_value = LEAST(current_odds.lowest_value, EXCLUDED.odds_value),
    -- Calculate movement: new_odds - opening_odds
    total_movement = EXCLUDED.odds_value - current_odds.opening_value,
    -- Calculate movement percentage: ((new_odds - opening_odds) / opening_odds) * 100
    movement_percentage = CASE
        WHEN current_odds.opening_value > 0 THEN (
            (
                (EXCLUDED.odds_value - current_odds.opening_value) / current_odds.opening_value * 100
            )
        )::REAL
        ELSE 0
    END,
    market_params = EXCLUDED.market_params,
    last_updated = CURRENT_TIMESTAMP RETURNING id, event_id, market_type_id, outcome, odds_value, opening_value, highest_value, lowest_value, winning_odds, total_movement, movement_percentage, last_updated, market_params
`

type UpsertCurrentOddsParams struct {
	EventID      int32   `db:"event_id" json:"event_id"`
	MarketTypeID int32   `db:"market_type_id" json:"market_type_id"`
	Outcome      string  `db:"outcome" json:"outcome"`
	OddsValue    float64 `db:"odds_value" json:"odds_value"`
	OpeningValue float64 `db:"opening_value" json:"opening_value"`
	HighestValue float64 `db:"highest_value" json:"highest_value"`
	LowestValue  float64 `db:"lowest_value" json:"lowest_value"`
	WinningOdds  float64 `db:"winning_odds" json:"winning_odds"`
	MarketParams []byte  `db:"market_params" json:"market_params"`
}

func (q *Queries) UpsertCurrentOdds(ctx context.Context, arg UpsertCurrentOddsParams) (CurrentOdd, error) {
	row := q.db.QueryRow(ctx, upsertCurrentOdds,
		arg.EventID,
		arg.MarketTypeID,
		arg.Outcome,
		arg.OddsValue,
		arg.OpeningValue,
		arg.HighestValue,
		arg.LowestValue,
		arg.WinningOdds,
		arg.MarketParams,
	)
	var i CurrentOdd
	err := row.Scan(
		&i.ID,
		&i.EventID,
		&i.MarketTypeID,
		&i.Outcome,
		&i.OddsValue,
		&i.OpeningValue,
		&i.HighestValue,
		&i.LowestValue,
		&i.WinningOdds,
		&i.TotalMovement,
		&i.MovementPercentage,
		&i.LastUpdated,
		&i.MarketParams,
	)
	return i, err
}
