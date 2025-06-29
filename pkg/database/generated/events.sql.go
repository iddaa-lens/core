// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: events.sql

package generated

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const bulkUpsertEvents = `-- name: BulkUpsertEvents :many
WITH input_data AS (
  SELECT
    unnest($1::text[]) as external_id,
    unnest($2::int[]) as league_id,
    unnest($3::int[]) as home_team_id,
    unnest($4::int[]) as away_team_id,
    unnest($5::timestamp[]) as event_date,
    unnest($6::text[]) as status,
    unnest($7::bigint[]) as bulletin_id,
    unnest($8::bigint[]) as version,
    unnest($9::int[]) as sport_id,
    unnest($10::int[]) as bet_program,
    unnest($11::int[]) as mbc,
    unnest($12::boolean[]) as has_king_odd,
    unnest($13::int[]) as odds_count,
    unnest($14::boolean[]) as has_combine,
    unnest($15::boolean[]) as is_live,
    unnest($16::text[]) as slug
)
INSERT INTO
  events (
    external_id,
    league_id,
    home_team_id,
    away_team_id,
    event_date,
    status,
    bulletin_id,
    version,
    sport_id,
    bet_program,
    mbc,
    has_king_odd,
    odds_count,
    has_combine,
    is_live,
    slug,
    created_at,
    updated_at
  )
SELECT
  external_id, league_id, home_team_id, away_team_id, event_date, status, bulletin_id, version, sport_id, bet_program, mbc, has_king_odd, odds_count, has_combine, is_live, slug,
  NOW(),
  NOW()
FROM
  input_data ON CONFLICT (external_id) DO
UPDATE
SET
  status = EXCLUDED.status,
  bulletin_id = EXCLUDED.bulletin_id,
  version = EXCLUDED.version,
  odds_count = EXCLUDED.odds_count,
  is_live = EXCLUDED.is_live,
  updated_at = NOW() RETURNING id,
  external_id
`

type BulkUpsertEventsParams struct {
	ExternalIds []string           `db:"external_ids" json:"external_ids"`
	LeagueIds   []int32            `db:"league_ids" json:"league_ids"`
	HomeTeamIds []int32            `db:"home_team_ids" json:"home_team_ids"`
	AwayTeamIds []int32            `db:"away_team_ids" json:"away_team_ids"`
	EventDates  []pgtype.Timestamp `db:"event_dates" json:"event_dates"`
	Statuses    []string           `db:"statuses" json:"statuses"`
	BulletinIds []int64            `db:"bulletin_ids" json:"bulletin_ids"`
	Versions    []int64            `db:"versions" json:"versions"`
	SportIds    []int32            `db:"sport_ids" json:"sport_ids"`
	BetPrograms []int32            `db:"bet_programs" json:"bet_programs"`
	Mbcs        []int32            `db:"mbcs" json:"mbcs"`
	HasKingOdds []bool             `db:"has_king_odds" json:"has_king_odds"`
	OddsCounts  []int32            `db:"odds_counts" json:"odds_counts"`
	HasCombines []bool             `db:"has_combines" json:"has_combines"`
	IsLives     []bool             `db:"is_lives" json:"is_lives"`
	Slugs       []string           `db:"slugs" json:"slugs"`
}

type BulkUpsertEventsRow struct {
	ID         int32  `db:"id" json:"id"`
	ExternalID string `db:"external_id" json:"external_id"`
}

func (q *Queries) BulkUpsertEvents(ctx context.Context, arg BulkUpsertEventsParams) ([]BulkUpsertEventsRow, error) {
	rows, err := q.db.Query(ctx, bulkUpsertEvents,
		arg.ExternalIds,
		arg.LeagueIds,
		arg.HomeTeamIds,
		arg.AwayTeamIds,
		arg.EventDates,
		arg.Statuses,
		arg.BulletinIds,
		arg.Versions,
		arg.SportIds,
		arg.BetPrograms,
		arg.Mbcs,
		arg.HasKingOdds,
		arg.OddsCounts,
		arg.HasCombines,
		arg.IsLives,
		arg.Slugs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []BulkUpsertEventsRow{}
	for rows.Next() {
		var i BulkUpsertEventsRow
		if err := rows.Scan(&i.ID, &i.ExternalID); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const countEventsFiltered = `-- name: CountEventsFiltered :one
SELECT
  COUNT(*)::int
FROM
  events e
  JOIN teams ht ON e.home_team_id = ht.id
  JOIN teams at ON e.away_team_id = at.id
  JOIN leagues l ON e.league_id = l.id
  JOIN sports s ON e.sport_id = s.id
WHERE
  e.event_date >= $1::timestamp
  AND e.event_date <= $2::timestamp
  AND (
    $3::text = ''
    OR s.code = $3::text
  )
  AND (
    $4::text = ''
    OR l.name ILIKE '%' || $4::text || '%'
  )
  AND (
    $5::text = ''
    OR e.status = $5::text
  )
`

type CountEventsFilteredParams struct {
	TimeAfter  pgtype.Timestamp `db:"time_after" json:"time_after"`
	TimeBefore pgtype.Timestamp `db:"time_before" json:"time_before"`
	SportCode  string           `db:"sport_code" json:"sport_code"`
	LeagueName string           `db:"league_name" json:"league_name"`
	Status     string           `db:"status" json:"status"`
}

func (q *Queries) CountEventsFiltered(ctx context.Context, arg CountEventsFilteredParams) (int32, error) {
	row := q.db.QueryRow(ctx, countEventsFiltered,
		arg.TimeAfter,
		arg.TimeBefore,
		arg.SportCode,
		arg.LeagueName,
		arg.Status,
	)
	var column_1 int32
	err := row.Scan(&column_1)
	return column_1, err
}

const createEvent = `-- name: CreateEvent :one
INSERT INTO
  events (
    external_id,
    league_id,
    home_team_id,
    away_team_id,
    slug,
    event_date,
    status
  )
VALUES
  (
    $1::text,
    $2::int,
    $3::int,
    $4::int,
    $5::text,
    $6::timestamp,
    $7::text
  ) RETURNING id, external_id, league_id, home_team_id, away_team_id, slug, event_date, status, home_score, away_score, is_live, minute_of_match, half, betting_volume_percentage, volume_rank, volume_updated_at, bulletin_id, version, sport_id, bet_program, mbc, has_king_odd, odds_count, has_combine, created_at, updated_at
`

type CreateEventParams struct {
	ExternalID string           `db:"external_id" json:"external_id"`
	LeagueID   int32            `db:"league_id" json:"league_id"`
	HomeTeamID int32            `db:"home_team_id" json:"home_team_id"`
	AwayTeamID int32            `db:"away_team_id" json:"away_team_id"`
	Slug       string           `db:"slug" json:"slug"`
	EventDate  pgtype.Timestamp `db:"event_date" json:"event_date"`
	Status     string           `db:"status" json:"status"`
}

func (q *Queries) CreateEvent(ctx context.Context, arg CreateEventParams) (Event, error) {
	row := q.db.QueryRow(ctx, createEvent,
		arg.ExternalID,
		arg.LeagueID,
		arg.HomeTeamID,
		arg.AwayTeamID,
		arg.Slug,
		arg.EventDate,
		arg.Status,
	)
	var i Event
	err := row.Scan(
		&i.ID,
		&i.ExternalID,
		&i.LeagueID,
		&i.HomeTeamID,
		&i.AwayTeamID,
		&i.Slug,
		&i.EventDate,
		&i.Status,
		&i.HomeScore,
		&i.AwayScore,
		&i.IsLive,
		&i.MinuteOfMatch,
		&i.Half,
		&i.BettingVolumePercentage,
		&i.VolumeRank,
		&i.VolumeUpdatedAt,
		&i.BulletinID,
		&i.Version,
		&i.SportID,
		&i.BetProgram,
		&i.Mbc,
		&i.HasKingOdd,
		&i.OddsCount,
		&i.HasCombine,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getActiveEventsForDetailedSync = `-- name: GetActiveEventsForDetailedSync :many
SELECT
  id, external_id, league_id, home_team_id, away_team_id, slug, event_date, status, home_score, away_score, is_live, minute_of_match, half, betting_volume_percentage, volume_rank, volume_updated_at, bulletin_id, version, sport_id, bet_program, mbc, has_king_odd, odds_count, has_combine, created_at, updated_at
FROM
  events
WHERE
  (
    status = 'live'
    OR status = 'scheduled'
  )
  AND event_date >= NOW() - INTERVAL '2 hours'
  AND event_date <= NOW() + INTERVAL '24 hours'
ORDER BY
  CASE
    WHEN status = 'live' THEN 1
    ELSE 2
  END,
  event_date ASC
LIMIT
  $1::int
`

func (q *Queries) GetActiveEventsForDetailedSync(ctx context.Context, limitCount int32) ([]Event, error) {
	rows, err := q.db.Query(ctx, getActiveEventsForDetailedSync, limitCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Event{}
	for rows.Next() {
		var i Event
		if err := rows.Scan(
			&i.ID,
			&i.ExternalID,
			&i.LeagueID,
			&i.HomeTeamID,
			&i.AwayTeamID,
			&i.Slug,
			&i.EventDate,
			&i.Status,
			&i.HomeScore,
			&i.AwayScore,
			&i.IsLive,
			&i.MinuteOfMatch,
			&i.Half,
			&i.BettingVolumePercentage,
			&i.VolumeRank,
			&i.VolumeUpdatedAt,
			&i.BulletinID,
			&i.Version,
			&i.SportID,
			&i.BetProgram,
			&i.Mbc,
			&i.HasKingOdd,
			&i.OddsCount,
			&i.HasCombine,
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

const getAllActiveEventsForDetailedSync = `-- name: GetAllActiveEventsForDetailedSync :many
SELECT
  id, external_id, league_id, home_team_id, away_team_id, slug, event_date, status, home_score, away_score, is_live, minute_of_match, half, betting_volume_percentage, volume_rank, volume_updated_at, bulletin_id, version, sport_id, bet_program, mbc, has_king_odd, odds_count, has_combine, created_at, updated_at
FROM
  events
WHERE
  (
    status = 'live'
    OR status = 'scheduled'
  )
  AND event_date >= NOW() - INTERVAL '2 hours'
  AND event_date <= NOW() + INTERVAL '48 hours'
ORDER BY
  CASE
    WHEN status = 'live' THEN 1
    WHEN event_date <= NOW() + INTERVAL '6 hours' THEN 2
    ELSE 3
  END,
  event_date ASC
`

func (q *Queries) GetAllActiveEventsForDetailedSync(ctx context.Context) ([]Event, error) {
	rows, err := q.db.Query(ctx, getAllActiveEventsForDetailedSync)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Event{}
	for rows.Next() {
		var i Event
		if err := rows.Scan(
			&i.ID,
			&i.ExternalID,
			&i.LeagueID,
			&i.HomeTeamID,
			&i.AwayTeamID,
			&i.Slug,
			&i.EventDate,
			&i.Status,
			&i.HomeScore,
			&i.AwayScore,
			&i.IsLive,
			&i.MinuteOfMatch,
			&i.Half,
			&i.BettingVolumePercentage,
			&i.VolumeRank,
			&i.VolumeUpdatedAt,
			&i.BulletinID,
			&i.Version,
			&i.SportID,
			&i.BetProgram,
			&i.Mbc,
			&i.HasKingOdd,
			&i.OddsCount,
			&i.HasCombine,
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

const getEvent = `-- name: GetEvent :one
SELECT
  e.id, e.external_id, e.league_id, e.home_team_id, e.away_team_id, e.slug, e.event_date, e.status, e.home_score, e.away_score, e.is_live, e.minute_of_match, e.half, e.betting_volume_percentage, e.volume_rank, e.volume_updated_at, e.bulletin_id, e.version, e.sport_id, e.bet_program, e.mbc, e.has_king_odd, e.odds_count, e.has_combine, e.created_at, e.updated_at,
  ht.name as home_team_name,
  at.name as away_team_name,
  l.name as league_name
FROM
  events e
  JOIN teams ht ON e.home_team_id = ht.id
  JOIN teams at ON e.away_team_id = at.id
  JOIN leagues l ON e.league_id = l.id
WHERE
  e.id = $1::int
`

type GetEventRow struct {
	ID                      int32            `db:"id" json:"id"`
	ExternalID              string           `db:"external_id" json:"external_id"`
	LeagueID                *int32           `db:"league_id" json:"league_id"`
	HomeTeamID              *int32           `db:"home_team_id" json:"home_team_id"`
	AwayTeamID              *int32           `db:"away_team_id" json:"away_team_id"`
	Slug                    string           `db:"slug" json:"slug"`
	EventDate               pgtype.Timestamp `db:"event_date" json:"event_date"`
	Status                  string           `db:"status" json:"status"`
	HomeScore               *int32           `db:"home_score" json:"home_score"`
	AwayScore               *int32           `db:"away_score" json:"away_score"`
	IsLive                  *bool            `db:"is_live" json:"is_live"`
	MinuteOfMatch           *int32           `db:"minute_of_match" json:"minute_of_match"`
	Half                    *int32           `db:"half" json:"half"`
	BettingVolumePercentage *float32         `db:"betting_volume_percentage" json:"betting_volume_percentage"`
	VolumeRank              *int32           `db:"volume_rank" json:"volume_rank"`
	VolumeUpdatedAt         pgtype.Timestamp `db:"volume_updated_at" json:"volume_updated_at"`
	BulletinID              *int64           `db:"bulletin_id" json:"bulletin_id"`
	Version                 *int64           `db:"version" json:"version"`
	SportID                 *int32           `db:"sport_id" json:"sport_id"`
	BetProgram              *int32           `db:"bet_program" json:"bet_program"`
	Mbc                     *int32           `db:"mbc" json:"mbc"`
	HasKingOdd              *bool            `db:"has_king_odd" json:"has_king_odd"`
	OddsCount               *int32           `db:"odds_count" json:"odds_count"`
	HasCombine              *bool            `db:"has_combine" json:"has_combine"`
	CreatedAt               pgtype.Timestamp `db:"created_at" json:"created_at"`
	UpdatedAt               pgtype.Timestamp `db:"updated_at" json:"updated_at"`
	HomeTeamName            string           `db:"home_team_name" json:"home_team_name"`
	AwayTeamName            string           `db:"away_team_name" json:"away_team_name"`
	LeagueName              string           `db:"league_name" json:"league_name"`
}

func (q *Queries) GetEvent(ctx context.Context, id int32) (GetEventRow, error) {
	row := q.db.QueryRow(ctx, getEvent, id)
	var i GetEventRow
	err := row.Scan(
		&i.ID,
		&i.ExternalID,
		&i.LeagueID,
		&i.HomeTeamID,
		&i.AwayTeamID,
		&i.Slug,
		&i.EventDate,
		&i.Status,
		&i.HomeScore,
		&i.AwayScore,
		&i.IsLive,
		&i.MinuteOfMatch,
		&i.Half,
		&i.BettingVolumePercentage,
		&i.VolumeRank,
		&i.VolumeUpdatedAt,
		&i.BulletinID,
		&i.Version,
		&i.SportID,
		&i.BetProgram,
		&i.Mbc,
		&i.HasKingOdd,
		&i.OddsCount,
		&i.HasCombine,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.HomeTeamName,
		&i.AwayTeamName,
		&i.LeagueName,
	)
	return i, err
}

const getEventByExternalID = `-- name: GetEventByExternalID :one
SELECT
  e.id, e.external_id, e.league_id, e.home_team_id, e.away_team_id, e.slug, e.event_date, e.status, e.home_score, e.away_score, e.is_live, e.minute_of_match, e.half, e.betting_volume_percentage, e.volume_rank, e.volume_updated_at, e.bulletin_id, e.version, e.sport_id, e.bet_program, e.mbc, e.has_king_odd, e.odds_count, e.has_combine, e.created_at, e.updated_at,
  ht.name as home_team_name,
  at.name as away_team_name,
  l.name as league_name
FROM
  events e
  JOIN teams ht ON e.home_team_id = ht.id
  JOIN teams at ON e.away_team_id = at.id
  LEFT JOIN leagues l ON e.league_id = l.id
WHERE
  e.external_id = $1::text
`

type GetEventByExternalIDRow struct {
	ID                      int32            `db:"id" json:"id"`
	ExternalID              string           `db:"external_id" json:"external_id"`
	LeagueID                *int32           `db:"league_id" json:"league_id"`
	HomeTeamID              *int32           `db:"home_team_id" json:"home_team_id"`
	AwayTeamID              *int32           `db:"away_team_id" json:"away_team_id"`
	Slug                    string           `db:"slug" json:"slug"`
	EventDate               pgtype.Timestamp `db:"event_date" json:"event_date"`
	Status                  string           `db:"status" json:"status"`
	HomeScore               *int32           `db:"home_score" json:"home_score"`
	AwayScore               *int32           `db:"away_score" json:"away_score"`
	IsLive                  *bool            `db:"is_live" json:"is_live"`
	MinuteOfMatch           *int32           `db:"minute_of_match" json:"minute_of_match"`
	Half                    *int32           `db:"half" json:"half"`
	BettingVolumePercentage *float32         `db:"betting_volume_percentage" json:"betting_volume_percentage"`
	VolumeRank              *int32           `db:"volume_rank" json:"volume_rank"`
	VolumeUpdatedAt         pgtype.Timestamp `db:"volume_updated_at" json:"volume_updated_at"`
	BulletinID              *int64           `db:"bulletin_id" json:"bulletin_id"`
	Version                 *int64           `db:"version" json:"version"`
	SportID                 *int32           `db:"sport_id" json:"sport_id"`
	BetProgram              *int32           `db:"bet_program" json:"bet_program"`
	Mbc                     *int32           `db:"mbc" json:"mbc"`
	HasKingOdd              *bool            `db:"has_king_odd" json:"has_king_odd"`
	OddsCount               *int32           `db:"odds_count" json:"odds_count"`
	HasCombine              *bool            `db:"has_combine" json:"has_combine"`
	CreatedAt               pgtype.Timestamp `db:"created_at" json:"created_at"`
	UpdatedAt               pgtype.Timestamp `db:"updated_at" json:"updated_at"`
	HomeTeamName            string           `db:"home_team_name" json:"home_team_name"`
	AwayTeamName            string           `db:"away_team_name" json:"away_team_name"`
	LeagueName              *string          `db:"league_name" json:"league_name"`
}

func (q *Queries) GetEventByExternalID(ctx context.Context, externalID string) (GetEventByExternalIDRow, error) {
	row := q.db.QueryRow(ctx, getEventByExternalID, externalID)
	var i GetEventByExternalIDRow
	err := row.Scan(
		&i.ID,
		&i.ExternalID,
		&i.LeagueID,
		&i.HomeTeamID,
		&i.AwayTeamID,
		&i.Slug,
		&i.EventDate,
		&i.Status,
		&i.HomeScore,
		&i.AwayScore,
		&i.IsLive,
		&i.MinuteOfMatch,
		&i.Half,
		&i.BettingVolumePercentage,
		&i.VolumeRank,
		&i.VolumeUpdatedAt,
		&i.BulletinID,
		&i.Version,
		&i.SportID,
		&i.BetProgram,
		&i.Mbc,
		&i.HasKingOdd,
		&i.OddsCount,
		&i.HasCombine,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.HomeTeamName,
		&i.AwayTeamName,
		&i.LeagueName,
	)
	return i, err
}

const getEventByExternalIDSimple = `-- name: GetEventByExternalIDSimple :one
SELECT
  id, external_id, league_id, home_team_id, away_team_id, slug, event_date, status, home_score, away_score, is_live, minute_of_match, half, betting_volume_percentage, volume_rank, volume_updated_at, bulletin_id, version, sport_id, bet_program, mbc, has_king_odd, odds_count, has_combine, created_at, updated_at
FROM
  events
WHERE
  external_id = $1::text
`

func (q *Queries) GetEventByExternalIDSimple(ctx context.Context, externalID string) (Event, error) {
	row := q.db.QueryRow(ctx, getEventByExternalIDSimple, externalID)
	var i Event
	err := row.Scan(
		&i.ID,
		&i.ExternalID,
		&i.LeagueID,
		&i.HomeTeamID,
		&i.AwayTeamID,
		&i.Slug,
		&i.EventDate,
		&i.Status,
		&i.HomeScore,
		&i.AwayScore,
		&i.IsLive,
		&i.MinuteOfMatch,
		&i.Half,
		&i.BettingVolumePercentage,
		&i.VolumeRank,
		&i.VolumeUpdatedAt,
		&i.BulletinID,
		&i.Version,
		&i.SportID,
		&i.BetProgram,
		&i.Mbc,
		&i.HasKingOdd,
		&i.OddsCount,
		&i.HasCombine,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getEventByID = `-- name: GetEventByID :one
SELECT
  id, external_id, league_id, home_team_id, away_team_id, slug, event_date, status, home_score, away_score, is_live, minute_of_match, half, betting_volume_percentage, volume_rank, volume_updated_at, bulletin_id, version, sport_id, bet_program, mbc, has_king_odd, odds_count, has_combine, created_at, updated_at
FROM
  events
WHERE
  id = $1::int
`

func (q *Queries) GetEventByID(ctx context.Context, id int32) (Event, error) {
	row := q.db.QueryRow(ctx, getEventByID, id)
	var i Event
	err := row.Scan(
		&i.ID,
		&i.ExternalID,
		&i.LeagueID,
		&i.HomeTeamID,
		&i.AwayTeamID,
		&i.Slug,
		&i.EventDate,
		&i.Status,
		&i.HomeScore,
		&i.AwayScore,
		&i.IsLive,
		&i.MinuteOfMatch,
		&i.Half,
		&i.BettingVolumePercentage,
		&i.VolumeRank,
		&i.VolumeUpdatedAt,
		&i.BulletinID,
		&i.Version,
		&i.SportID,
		&i.BetProgram,
		&i.Mbc,
		&i.HasKingOdd,
		&i.OddsCount,
		&i.HasCombine,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getEventsByTeam = `-- name: GetEventsByTeam :many
SELECT
  e.id, e.external_id, e.league_id, e.home_team_id, e.away_team_id, e.slug, e.event_date, e.status, e.home_score, e.away_score, e.is_live, e.minute_of_match, e.half, e.betting_volume_percentage, e.volume_rank, e.volume_updated_at, e.bulletin_id, e.version, e.sport_id, e.bet_program, e.mbc, e.has_king_odd, e.odds_count, e.has_combine, e.created_at, e.updated_at,
  l.name as league_name
FROM
  events e
  LEFT JOIN leagues l ON e.league_id = l.id
WHERE
  (
    e.home_team_id = $1::int
    OR e.away_team_id = $1::int
  )
  AND e.event_date >= $2::timestamp
ORDER BY
  e.event_date DESC
LIMIT
  $3::int
`

type GetEventsByTeamParams struct {
	TeamID     int32            `db:"team_id" json:"team_id"`
	SinceDate  pgtype.Timestamp `db:"since_date" json:"since_date"`
	LimitCount int32            `db:"limit_count" json:"limit_count"`
}

type GetEventsByTeamRow struct {
	ID                      int32            `db:"id" json:"id"`
	ExternalID              string           `db:"external_id" json:"external_id"`
	LeagueID                *int32           `db:"league_id" json:"league_id"`
	HomeTeamID              *int32           `db:"home_team_id" json:"home_team_id"`
	AwayTeamID              *int32           `db:"away_team_id" json:"away_team_id"`
	Slug                    string           `db:"slug" json:"slug"`
	EventDate               pgtype.Timestamp `db:"event_date" json:"event_date"`
	Status                  string           `db:"status" json:"status"`
	HomeScore               *int32           `db:"home_score" json:"home_score"`
	AwayScore               *int32           `db:"away_score" json:"away_score"`
	IsLive                  *bool            `db:"is_live" json:"is_live"`
	MinuteOfMatch           *int32           `db:"minute_of_match" json:"minute_of_match"`
	Half                    *int32           `db:"half" json:"half"`
	BettingVolumePercentage *float32         `db:"betting_volume_percentage" json:"betting_volume_percentage"`
	VolumeRank              *int32           `db:"volume_rank" json:"volume_rank"`
	VolumeUpdatedAt         pgtype.Timestamp `db:"volume_updated_at" json:"volume_updated_at"`
	BulletinID              *int64           `db:"bulletin_id" json:"bulletin_id"`
	Version                 *int64           `db:"version" json:"version"`
	SportID                 *int32           `db:"sport_id" json:"sport_id"`
	BetProgram              *int32           `db:"bet_program" json:"bet_program"`
	Mbc                     *int32           `db:"mbc" json:"mbc"`
	HasKingOdd              *bool            `db:"has_king_odd" json:"has_king_odd"`
	OddsCount               *int32           `db:"odds_count" json:"odds_count"`
	HasCombine              *bool            `db:"has_combine" json:"has_combine"`
	CreatedAt               pgtype.Timestamp `db:"created_at" json:"created_at"`
	UpdatedAt               pgtype.Timestamp `db:"updated_at" json:"updated_at"`
	LeagueName              *string          `db:"league_name" json:"league_name"`
}

func (q *Queries) GetEventsByTeam(ctx context.Context, arg GetEventsByTeamParams) ([]GetEventsByTeamRow, error) {
	rows, err := q.db.Query(ctx, getEventsByTeam, arg.TeamID, arg.SinceDate, arg.LimitCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetEventsByTeamRow{}
	for rows.Next() {
		var i GetEventsByTeamRow
		if err := rows.Scan(
			&i.ID,
			&i.ExternalID,
			&i.LeagueID,
			&i.HomeTeamID,
			&i.AwayTeamID,
			&i.Slug,
			&i.EventDate,
			&i.Status,
			&i.HomeScore,
			&i.AwayScore,
			&i.IsLive,
			&i.MinuteOfMatch,
			&i.Half,
			&i.BettingVolumePercentage,
			&i.VolumeRank,
			&i.VolumeUpdatedAt,
			&i.BulletinID,
			&i.Version,
			&i.SportID,
			&i.BetProgram,
			&i.Mbc,
			&i.HasKingOdd,
			&i.OddsCount,
			&i.HasCombine,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.LeagueName,
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

const listEventsByDate = `-- name: ListEventsByDate :many
SELECT
  e.id, e.external_id, e.league_id, e.home_team_id, e.away_team_id, e.slug, e.event_date, e.status, e.home_score, e.away_score, e.is_live, e.minute_of_match, e.half, e.betting_volume_percentage, e.volume_rank, e.volume_updated_at, e.bulletin_id, e.version, e.sport_id, e.bet_program, e.mbc, e.has_king_odd, e.odds_count, e.has_combine, e.created_at, e.updated_at,
  ht.name as home_team_name,
  at.name as away_team_name,
  l.name as league_name
FROM
  events e
  JOIN teams ht ON e.home_team_id = ht.id
  JOIN teams at ON e.away_team_id = at.id
  JOIN leagues l ON e.league_id = l.id
WHERE
  DATE(e.event_date) = DATE($1::timestamp)
ORDER BY
  e.event_date
`

type ListEventsByDateRow struct {
	ID                      int32            `db:"id" json:"id"`
	ExternalID              string           `db:"external_id" json:"external_id"`
	LeagueID                *int32           `db:"league_id" json:"league_id"`
	HomeTeamID              *int32           `db:"home_team_id" json:"home_team_id"`
	AwayTeamID              *int32           `db:"away_team_id" json:"away_team_id"`
	Slug                    string           `db:"slug" json:"slug"`
	EventDate               pgtype.Timestamp `db:"event_date" json:"event_date"`
	Status                  string           `db:"status" json:"status"`
	HomeScore               *int32           `db:"home_score" json:"home_score"`
	AwayScore               *int32           `db:"away_score" json:"away_score"`
	IsLive                  *bool            `db:"is_live" json:"is_live"`
	MinuteOfMatch           *int32           `db:"minute_of_match" json:"minute_of_match"`
	Half                    *int32           `db:"half" json:"half"`
	BettingVolumePercentage *float32         `db:"betting_volume_percentage" json:"betting_volume_percentage"`
	VolumeRank              *int32           `db:"volume_rank" json:"volume_rank"`
	VolumeUpdatedAt         pgtype.Timestamp `db:"volume_updated_at" json:"volume_updated_at"`
	BulletinID              *int64           `db:"bulletin_id" json:"bulletin_id"`
	Version                 *int64           `db:"version" json:"version"`
	SportID                 *int32           `db:"sport_id" json:"sport_id"`
	BetProgram              *int32           `db:"bet_program" json:"bet_program"`
	Mbc                     *int32           `db:"mbc" json:"mbc"`
	HasKingOdd              *bool            `db:"has_king_odd" json:"has_king_odd"`
	OddsCount               *int32           `db:"odds_count" json:"odds_count"`
	HasCombine              *bool            `db:"has_combine" json:"has_combine"`
	CreatedAt               pgtype.Timestamp `db:"created_at" json:"created_at"`
	UpdatedAt               pgtype.Timestamp `db:"updated_at" json:"updated_at"`
	HomeTeamName            string           `db:"home_team_name" json:"home_team_name"`
	AwayTeamName            string           `db:"away_team_name" json:"away_team_name"`
	LeagueName              string           `db:"league_name" json:"league_name"`
}

func (q *Queries) ListEventsByDate(ctx context.Context, eventDate pgtype.Timestamp) ([]ListEventsByDateRow, error) {
	rows, err := q.db.Query(ctx, listEventsByDate, eventDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListEventsByDateRow{}
	for rows.Next() {
		var i ListEventsByDateRow
		if err := rows.Scan(
			&i.ID,
			&i.ExternalID,
			&i.LeagueID,
			&i.HomeTeamID,
			&i.AwayTeamID,
			&i.Slug,
			&i.EventDate,
			&i.Status,
			&i.HomeScore,
			&i.AwayScore,
			&i.IsLive,
			&i.MinuteOfMatch,
			&i.Half,
			&i.BettingVolumePercentage,
			&i.VolumeRank,
			&i.VolumeUpdatedAt,
			&i.BulletinID,
			&i.Version,
			&i.SportID,
			&i.BetProgram,
			&i.Mbc,
			&i.HasKingOdd,
			&i.OddsCount,
			&i.HasCombine,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.HomeTeamName,
			&i.AwayTeamName,
			&i.LeagueName,
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

const listEventsFiltered = `-- name: ListEventsFiltered :many
SELECT
  e.id,
  e.external_id,
  e.league_id,
  e.home_team_id,
  e.away_team_id,
  e.slug,
  e.event_date,
  e.status,
  e.home_score,
  e.away_score,
  e.is_live,
  e.minute_of_match,
  e.half,
  e.betting_volume_percentage,
  e.volume_rank,
  e.volume_updated_at,
  e.bulletin_id,
  e.version,
  e.sport_id,
  e.bet_program,
  e.mbc,
  e.has_king_odd,
  e.odds_count,
  e.has_combine,
  e.created_at,
  e.updated_at,
  ht.name as home_team_name,
  ht.country as home_team_country,
  at.name as away_team_name,
  at.country as away_team_country,
  l.name as league_name,
  l.country as league_country,
  s.name as sport_name,
  s.code as sport_code
FROM
  events e
  JOIN teams ht ON e.home_team_id = ht.id
  JOIN teams at ON e.away_team_id = at.id
  JOIN leagues l ON e.league_id = l.id
  JOIN sports s ON e.sport_id = s.id
WHERE
  e.event_date >= $1::timestamp
  AND e.event_date <= $2::timestamp
  AND (
    $3::text = ''
    OR s.code = $3::text
  )
  AND (
    $4::text = ''
    OR l.name ILIKE '%' || $4::text || '%'
  )
  AND (
    $5::text = ''
    OR e.status = $5::text
  )
ORDER BY
  e.event_date ASC
LIMIT
  $7::int OFFSET $6::int
`

type ListEventsFilteredParams struct {
	TimeAfter   pgtype.Timestamp `db:"time_after" json:"time_after"`
	TimeBefore  pgtype.Timestamp `db:"time_before" json:"time_before"`
	SportCode   string           `db:"sport_code" json:"sport_code"`
	LeagueName  string           `db:"league_name" json:"league_name"`
	Status      string           `db:"status" json:"status"`
	OffsetCount int32            `db:"offset_count" json:"offset_count"`
	LimitCount  int32            `db:"limit_count" json:"limit_count"`
}

type ListEventsFilteredRow struct {
	ID                      int32            `db:"id" json:"id"`
	ExternalID              string           `db:"external_id" json:"external_id"`
	LeagueID                *int32           `db:"league_id" json:"league_id"`
	HomeTeamID              *int32           `db:"home_team_id" json:"home_team_id"`
	AwayTeamID              *int32           `db:"away_team_id" json:"away_team_id"`
	Slug                    string           `db:"slug" json:"slug"`
	EventDate               pgtype.Timestamp `db:"event_date" json:"event_date"`
	Status                  string           `db:"status" json:"status"`
	HomeScore               *int32           `db:"home_score" json:"home_score"`
	AwayScore               *int32           `db:"away_score" json:"away_score"`
	IsLive                  *bool            `db:"is_live" json:"is_live"`
	MinuteOfMatch           *int32           `db:"minute_of_match" json:"minute_of_match"`
	Half                    *int32           `db:"half" json:"half"`
	BettingVolumePercentage *float32         `db:"betting_volume_percentage" json:"betting_volume_percentage"`
	VolumeRank              *int32           `db:"volume_rank" json:"volume_rank"`
	VolumeUpdatedAt         pgtype.Timestamp `db:"volume_updated_at" json:"volume_updated_at"`
	BulletinID              *int64           `db:"bulletin_id" json:"bulletin_id"`
	Version                 *int64           `db:"version" json:"version"`
	SportID                 *int32           `db:"sport_id" json:"sport_id"`
	BetProgram              *int32           `db:"bet_program" json:"bet_program"`
	Mbc                     *int32           `db:"mbc" json:"mbc"`
	HasKingOdd              *bool            `db:"has_king_odd" json:"has_king_odd"`
	OddsCount               *int32           `db:"odds_count" json:"odds_count"`
	HasCombine              *bool            `db:"has_combine" json:"has_combine"`
	CreatedAt               pgtype.Timestamp `db:"created_at" json:"created_at"`
	UpdatedAt               pgtype.Timestamp `db:"updated_at" json:"updated_at"`
	HomeTeamName            string           `db:"home_team_name" json:"home_team_name"`
	HomeTeamCountry         *string          `db:"home_team_country" json:"home_team_country"`
	AwayTeamName            string           `db:"away_team_name" json:"away_team_name"`
	AwayTeamCountry         *string          `db:"away_team_country" json:"away_team_country"`
	LeagueName              string           `db:"league_name" json:"league_name"`
	LeagueCountry           *string          `db:"league_country" json:"league_country"`
	SportName               string           `db:"sport_name" json:"sport_name"`
	SportCode               string           `db:"sport_code" json:"sport_code"`
}

func (q *Queries) ListEventsFiltered(ctx context.Context, arg ListEventsFilteredParams) ([]ListEventsFilteredRow, error) {
	rows, err := q.db.Query(ctx, listEventsFiltered,
		arg.TimeAfter,
		arg.TimeBefore,
		arg.SportCode,
		arg.LeagueName,
		arg.Status,
		arg.OffsetCount,
		arg.LimitCount,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListEventsFilteredRow{}
	for rows.Next() {
		var i ListEventsFilteredRow
		if err := rows.Scan(
			&i.ID,
			&i.ExternalID,
			&i.LeagueID,
			&i.HomeTeamID,
			&i.AwayTeamID,
			&i.Slug,
			&i.EventDate,
			&i.Status,
			&i.HomeScore,
			&i.AwayScore,
			&i.IsLive,
			&i.MinuteOfMatch,
			&i.Half,
			&i.BettingVolumePercentage,
			&i.VolumeRank,
			&i.VolumeUpdatedAt,
			&i.BulletinID,
			&i.Version,
			&i.SportID,
			&i.BetProgram,
			&i.Mbc,
			&i.HasKingOdd,
			&i.OddsCount,
			&i.HasCombine,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.HomeTeamName,
			&i.HomeTeamCountry,
			&i.AwayTeamName,
			&i.AwayTeamCountry,
			&i.LeagueName,
			&i.LeagueCountry,
			&i.SportName,
			&i.SportCode,
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

const updateEventStatus = `-- name: UpdateEventStatus :one
UPDATE
  events
SET
  status = $1::text,
  home_score = $2::int,
  away_score = $3::int,
  updated_at = CURRENT_TIMESTAMP
WHERE
  id = $4::int RETURNING id, external_id, league_id, home_team_id, away_team_id, slug, event_date, status, home_score, away_score, is_live, minute_of_match, half, betting_volume_percentage, volume_rank, volume_updated_at, bulletin_id, version, sport_id, bet_program, mbc, has_king_odd, odds_count, has_combine, created_at, updated_at
`

type UpdateEventStatusParams struct {
	Status    string `db:"status" json:"status"`
	HomeScore int32  `db:"home_score" json:"home_score"`
	AwayScore int32  `db:"away_score" json:"away_score"`
	ID        int32  `db:"id" json:"id"`
}

func (q *Queries) UpdateEventStatus(ctx context.Context, arg UpdateEventStatusParams) (Event, error) {
	row := q.db.QueryRow(ctx, updateEventStatus,
		arg.Status,
		arg.HomeScore,
		arg.AwayScore,
		arg.ID,
	)
	var i Event
	err := row.Scan(
		&i.ID,
		&i.ExternalID,
		&i.LeagueID,
		&i.HomeTeamID,
		&i.AwayTeamID,
		&i.Slug,
		&i.EventDate,
		&i.Status,
		&i.HomeScore,
		&i.AwayScore,
		&i.IsLive,
		&i.MinuteOfMatch,
		&i.Half,
		&i.BettingVolumePercentage,
		&i.VolumeRank,
		&i.VolumeUpdatedAt,
		&i.BulletinID,
		&i.Version,
		&i.SportID,
		&i.BetProgram,
		&i.Mbc,
		&i.HasKingOdd,
		&i.OddsCount,
		&i.HasCombine,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const upsertEvent = `-- name: UpsertEvent :one
INSERT INTO
  events (
    external_id,
    league_id,
    home_team_id,
    away_team_id,
    slug,
    event_date,
    status,
    home_score,
    away_score,
    bulletin_id,
    version,
    sport_id,
    bet_program,
    mbc,
    has_king_odd,
    odds_count,
    has_combine,
    is_live
  )
VALUES
  (
    $1::text,
    $2::int,
    $3::int,
    $4::int,
    $5::text,
    $6::timestamp,
    $7::text,
    NULLIF($8::int, 0),
    NULLIF($9::int, 0),
    $10::bigint,
    $11::bigint,
    $12::int,
    $13::int,
    $14::int,
    $15::boolean,
    $16::int,
    $17::boolean,
    $18::boolean
  ) ON CONFLICT (external_id) DO
UPDATE
SET
  league_id = EXCLUDED.league_id,
  home_team_id = EXCLUDED.home_team_id,
  away_team_id = EXCLUDED.away_team_id,
  slug = EXCLUDED.slug,
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
  updated_at = CURRENT_TIMESTAMP RETURNING id, external_id, league_id, home_team_id, away_team_id, slug, event_date, status, home_score, away_score, is_live, minute_of_match, half, betting_volume_percentage, volume_rank, volume_updated_at, bulletin_id, version, sport_id, bet_program, mbc, has_king_odd, odds_count, has_combine, created_at, updated_at
`

type UpsertEventParams struct {
	ExternalID string           `db:"external_id" json:"external_id"`
	LeagueID   int32            `db:"league_id" json:"league_id"`
	HomeTeamID int32            `db:"home_team_id" json:"home_team_id"`
	AwayTeamID int32            `db:"away_team_id" json:"away_team_id"`
	Slug       string           `db:"slug" json:"slug"`
	EventDate  pgtype.Timestamp `db:"event_date" json:"event_date"`
	Status     string           `db:"status" json:"status"`
	HomeScore  int32            `db:"home_score" json:"home_score"`
	AwayScore  int32            `db:"away_score" json:"away_score"`
	BulletinID int64            `db:"bulletin_id" json:"bulletin_id"`
	Version    int64            `db:"version" json:"version"`
	SportID    int32            `db:"sport_id" json:"sport_id"`
	BetProgram int32            `db:"bet_program" json:"bet_program"`
	Mbc        int32            `db:"mbc" json:"mbc"`
	HasKingOdd bool             `db:"has_king_odd" json:"has_king_odd"`
	OddsCount  int32            `db:"odds_count" json:"odds_count"`
	HasCombine bool             `db:"has_combine" json:"has_combine"`
	IsLive     bool             `db:"is_live" json:"is_live"`
}

func (q *Queries) UpsertEvent(ctx context.Context, arg UpsertEventParams) (Event, error) {
	row := q.db.QueryRow(ctx, upsertEvent,
		arg.ExternalID,
		arg.LeagueID,
		arg.HomeTeamID,
		arg.AwayTeamID,
		arg.Slug,
		arg.EventDate,
		arg.Status,
		arg.HomeScore,
		arg.AwayScore,
		arg.BulletinID,
		arg.Version,
		arg.SportID,
		arg.BetProgram,
		arg.Mbc,
		arg.HasKingOdd,
		arg.OddsCount,
		arg.HasCombine,
		arg.IsLive,
	)
	var i Event
	err := row.Scan(
		&i.ID,
		&i.ExternalID,
		&i.LeagueID,
		&i.HomeTeamID,
		&i.AwayTeamID,
		&i.Slug,
		&i.EventDate,
		&i.Status,
		&i.HomeScore,
		&i.AwayScore,
		&i.IsLive,
		&i.MinuteOfMatch,
		&i.Half,
		&i.BettingVolumePercentage,
		&i.VolumeRank,
		&i.VolumeUpdatedAt,
		&i.BulletinID,
		&i.Version,
		&i.SportID,
		&i.BetProgram,
		&i.Mbc,
		&i.HasKingOdd,
		&i.OddsCount,
		&i.HasCombine,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
