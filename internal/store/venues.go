package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/MonalFinbox/seatrush/internal/models"
)

const venueColumns = `id, name, address, city, capacity, claim_status, organizer_id, created_at, updated_at`

func scanVenue(row pgx.Row) (*models.Venue, error) {
	var v models.Venue
	err := row.Scan(&v.ID, &v.Name, &v.Address, &v.City, &v.Capacity,
		&v.ClaimStatus, &v.OrganizerID, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListVenues optionally filters by claim status ("" = no filter).
func (s *Store) ListVenues(ctx context.Context, claimStatus string) ([]models.Venue, error) {
	query := `SELECT ` + venueColumns + ` FROM venues`
	args := []any{}
	if claimStatus != "" {
		query += ` WHERE claim_status = $1`
		args = append(args, claimStatus)
	}
	query += ` ORDER BY city, name`

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	venues := []models.Venue{}
	for rows.Next() {
		v, err := scanVenue(rows)
		if err != nil {
			return nil, err
		}
		venues = append(venues, *v)
	}
	return venues, rows.Err()
}

func (s *Store) GetVenue(ctx context.Context, id string) (*models.Venue, error) {
	row := s.db.QueryRow(ctx, `SELECT `+venueColumns+` FROM venues WHERE id = $1`, id)
	return scanVenue(row)
}
