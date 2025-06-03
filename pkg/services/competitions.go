package services

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/betslib/iddaa-core/pkg/database"
	"github.com/betslib/iddaa-core/pkg/models"
)

type CompetitionService struct {
	db     *database.Queries
	client *IddaaClient
}

func NewCompetitionService(db *database.Queries, client *IddaaClient) *CompetitionService {
	return &CompetitionService{
		db:     db,
		client: client,
	}
}

func (s *CompetitionService) SyncCompetitions(ctx context.Context) error {
	log.Println("Starting competitions sync...")

	resp, err := s.client.GetCompetitions()
	if err != nil {
		return fmt.Errorf("failed to fetch competitions: %w", err)
	}

	log.Printf("Fetched %d competitions from API", len(resp.Data))

	for _, comp := range resp.Data {
		if err := s.saveCompetition(ctx, comp); err != nil {
			log.Printf("Failed to save competition %d: %v", comp.ID, err)
			continue
		}
	}

	log.Println("Competitions sync completed")
	return nil
}

func (s *CompetitionService) saveCompetition(ctx context.Context, comp models.IddaaCompetition) error {
	params := database.UpsertCompetitionParams{
		IddaaID:  int32(comp.ID),
		FullName: comp.FullName,
	}

	// Handle sport ID - check if it's a valid numeric ID and exists in sports table
	if comp.SportID != "" {
		sportID, err := strconv.Atoi(comp.SportID)
		if err != nil {
			log.Printf("Warning: invalid sport ID %s for competition %d: %v", comp.SportID, comp.ID, err)
		} else if sportID != 0 {
			// Check if sport exists
			sport, err := s.db.GetSport(ctx, int32(sportID))
			if err != nil {
				log.Printf("Warning: sport ID %d not found for competition %d, will be saved without sport", sportID, comp.ID)
			} else {
				// Sport exists, we can safely reference it
				params.SportID = pgtype.Int4{Int32: sport.ID, Valid: true}
			}
		}
	}

	// Handle optional external ref
	if comp.ExternalRef != 0 {
		params.ExternalRef = pgtype.Int4{Int32: int32(comp.ExternalRef), Valid: true}
	}

	// Handle optional country code
	if comp.CountryCode != "" {
		params.CountryCode = pgtype.Text{String: comp.CountryCode, Valid: true}
	}

	// Handle optional parent ID
	if comp.ParentID != 0 {
		params.ParentID = pgtype.Int4{Int32: int32(comp.ParentID), Valid: true}
	}

	// Handle optional short name
	if comp.ShortName != "" {
		params.ShortName = pgtype.Text{String: comp.ShortName, Valid: true}
	}

	// Handle optional icon URL
	if comp.IconURL != "" {
		params.IconUrl = pgtype.Text{String: comp.IconURL, Valid: true}
	}

	_, err := s.db.UpsertCompetition(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to upsert competition: %w", err)
	}

	return nil
}

func (s *CompetitionService) GetCompetitionByIddaaID(ctx context.Context, iddaaID int32) (*database.GetCompetitionByIddaaIDRow, error) {
	comp, err := s.db.GetCompetitionByIddaaID(ctx, iddaaID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("competition with iddaa ID %d not found", iddaaID)
		}
		return nil, fmt.Errorf("failed to get competition: %w", err)
	}
	return &comp, nil
}

func (s *CompetitionService) ListCompetitionsBySport(ctx context.Context, sportID int32) ([]database.ListCompetitionsBySportRow, error) {
	sportIDParam := pgtype.Int4{Int32: sportID, Valid: true}
	comps, err := s.db.ListCompetitionsBySport(ctx, sportIDParam)
	if err != nil {
		return nil, fmt.Errorf("failed to list competitions by sport: %w", err)
	}
	return comps, nil
}
