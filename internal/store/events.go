package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MonalFinbox/seatrush/internal/models"
)

const eventColumns = `id, venue_id, organizer_id, title, description, event_date, status, created_at, updated_at`

func scanEvent(row pgx.Row) (*models.Event, error) {
	var e models.Event
	err := row.Scan(&e.ID, &e.VenueID, &e.OrganizerID, &e.Title, &e.Description,
		&e.EventDate, &e.Status, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) CreateEvent(ctx context.Context, venueID, organizerID, title string, description *string, eventDate time.Time) (*models.Event, error) {
	row := s.db.QueryRow(ctx, `
		INSERT INTO events (venue_id, organizer_id, title, description, event_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+eventColumns,
		venueID, organizerID, title, description, eventDate,
	)
	return scanEvent(row)
}

func (s *Store) GetEvent(ctx context.Context, id string) (*models.Event, error) {
	row := s.db.QueryRow(ctx, `SELECT `+eventColumns+` FROM events WHERE id = $1`, id)
	return scanEvent(row)
}

func (s *Store) ListEvents(ctx context.Context, status string) ([]models.Event, error) {
	query := `SELECT ` + eventColumns + ` FROM events`
	args := []any{}
	if status != "" {
		query += ` WHERE status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY event_date ASC`

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectEvents(rows)
}

func (s *Store) ListAllEvents(ctx context.Context) ([]models.Event, error) {
	rows, err := s.db.Query(ctx, `SELECT `+eventColumns+` FROM events ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectEvents(rows)
}

func collectEvents(rows pgx.Rows) ([]models.Event, error) {
	out := []models.Event{}
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

// UpdateEvent applies a partial update. Nil fields are left unchanged via
// COALESCE, so callers only set what they want to change.
func (s *Store) UpdateEvent(ctx context.Context, id string, title, description *string, eventDate *time.Time) (*models.Event, error) {
	row := s.db.QueryRow(ctx, `
		UPDATE events SET
			title = COALESCE($1, title),
			description = COALESCE($2, description),
			event_date = COALESCE($3, event_date)
		WHERE id = $4
		RETURNING `+eventColumns,
		title, description, eventDate, id,
	)
	return scanEvent(row)
}

func (s *Store) SetEventStatus(ctx context.Context, id, status string) (*models.Event, error) {
	row := s.db.QueryRow(ctx, `
		UPDATE events SET status = $1 WHERE id = $2 RETURNING `+eventColumns,
		status, id,
	)
	return scanEvent(row)
}
