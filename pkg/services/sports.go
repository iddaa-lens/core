package services

import (
	"context"
	"fmt"

	"github.com/gosimple/slug"

	"github.com/iddaa-lens/core/pkg/database/generated"
	"github.com/iddaa-lens/core/pkg/logger"
)

type SportService struct {
	db     *generated.Queries
	client *IddaaClient
	logger *logger.Logger
}

func NewSportService(db *generated.Queries, client *IddaaClient) *SportService {
	return &SportService{
		db:     db,
		client: client,
		logger: logger.New("sport-service"),
	}
}

// sportMapping defines which sports we support with their display names and codes
var sportMapping = map[int]struct {
	name string
	code string
}{
	1: {name: "Football", code: "FOOTBALL"},
	2: {name: "Basketball", code: "BASKETBALL"},
	// Future sports can be uncommented when ready:
	// 4:   {name: "Ice Hockey", code: "ICE_HOCKEY"},
	// 5:   {name: "Tennis", code: "TENNIS"},
	// 6:   {name: "Handball", code: "HANDBALL"},
	// 11:  {name: "Formula 1", code: "FORMULA1"},
	// 19:  {name: "Rugby", code: "RUGBY"},
	// 23:  {name: "Volleyball", code: "VOLLEYBALL"},
	// 117: {name: "MMA", code: "MMA"},
}

// SyncSports fetches sports from Iddaa API and bulk upserts them to database
func (s *SportService) SyncSports(ctx context.Context) error {
	// Fetch sports info from Iddaa API
	resp, err := s.client.GetSportInfo()
	if err != nil {
		return fmt.Errorf("failed to fetch sport info: %w", err)
	}

	s.logger.Info().
		Int("total_sports", len(resp.Data)).
		Msg("Fetched sports from Iddaa API")

	// Prepare bulk data arrays
	var (
		ids                []int32
		names              []string
		codes              []string
		slugs              []string
		liveCounts         []int32
		upcomingCounts     []int32
		eventsCounts       []int32
		oddsCounts         []int32
		hasResults         []bool
		hasKingOdds        []bool
		hasDigitalContents []bool
	)

	// Filter and collect sports data
	filtered := 0
	for _, sport := range resp.Data {
		// Only process sports we support
		sportInfo, exists := sportMapping[sport.SportID]
		if !exists {
			continue
		}

		// Append to bulk arrays
		ids = append(ids, int32(sport.SportID))
		names = append(names, sportInfo.name)
		codes = append(codes, sportInfo.code)
		slugs = append(slugs, slug.Make(sportInfo.name))
		liveCounts = append(liveCounts, int32(sport.LiveCount))
		upcomingCounts = append(upcomingCounts, int32(sport.UpcomingCount))
		eventsCounts = append(eventsCounts, int32(sport.EventsCount))
		oddsCounts = append(oddsCounts, int32(sport.OddsCount))
		hasResults = append(hasResults, sport.HasResults)
		hasKingOdds = append(hasKingOdds, sport.HasKingOdd)
		hasDigitalContents = append(hasDigitalContents, sport.HasDigitalContent)

		filtered++
	}

	// Skip if no sports to sync
	if filtered == 0 {
		s.logger.Warn().Msg("No supported sports found to sync")
		return nil
	}

	s.logger.Info().
		Int("filtered_sports", filtered).
		Int("supported_sports", len(sportMapping)).
		Msg("Filtered sports for bulk upsert")

	// Perform bulk upsert
	params := generated.BulkUpsertSportsParams{
		Ids:                ids,
		Names:              names,
		Codes:              codes,
		Slugs:              slugs,
		LiveCounts:         liveCounts,
		UpcomingCounts:     upcomingCounts,
		EventsCounts:       eventsCounts,
		OddsCounts:         oddsCounts,
		HasResults:         hasResults,
		HasKingOdds:        hasKingOdds,
		HasDigitalContents: hasDigitalContents,
	}

	rowsAffected, err := s.db.BulkUpsertSports(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to bulk upsert sports: %w", err)
	}

	s.logger.Info().
		Int64("rows_affected", rowsAffected).
		Int("sports_synced", filtered).
		Msg("Successfully synced sports")

	return nil
}

func (s *SportService) GetSport(ctx context.Context, sportID int32) (*generated.Sport, error) {
	sport, err := s.db.GetSport(ctx, sportID)
	if err != nil {
		return nil, err
	}
	return &sport, nil
}

func (s *SportService) ListSports(ctx context.Context) ([]generated.Sport, error) {
	return s.db.ListSports(ctx)
}
