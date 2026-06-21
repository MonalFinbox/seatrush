package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/MonalFinbox/seatrush/internal/models"
)

// price is cast to float8 so it scans cleanly into a Go float64 (NUMERIC
// otherwise comes back as an arbitrary-precision type).
const seatColumns = `id, event_id, section, seat_row, number, price::float8, status, created_at, updated_at`

// SeatInput is one seat in a bulk seat-map definition.
type SeatInput struct {
	Section string
	Row     string
	Number  string
	Price   float64
}

func scanSeat(row pgx.Row) (*models.Seat, error) {
	var st models.Seat
	err := row.Scan(&st.ID, &st.EventID, &st.Section, &st.Row, &st.Number,
		&st.Price, &st.Status, &st.CreatedAt, &st.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// BulkCreateSeats inserts an entire seat map in one transaction. If any seat
// collides with the (event, section, row, number) unique constraint, the whole
// batch rolls back — you never get a half-built seat map.
func (s *Store) BulkCreateSeats(ctx context.Context, eventID string, seats []SeatInput) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, st := range seats {
		_, err := tx.Exec(ctx, `
			INSERT INTO seats (event_id, section, seat_row, number, price)
			VALUES ($1, $2, $3, $4, $5)`,
			eventID, st.Section, st.Row, st.Number, st.Price,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) ListSeats(ctx context.Context, eventID string) ([]models.Seat, error) {
	rows, err := s.db.Query(ctx, `SELECT `+seatColumns+`
		FROM seats WHERE event_id = $1
		ORDER BY section, seat_row, number`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Seat{}
	for rows.Next() {
		st, err := scanSeat(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *st)
	}
	return out, rows.Err()
}

// CountSeats reports how many seats an event already has — used to reject a
// second bulk seat-map definition.
func (s *Store) CountSeats(ctx context.Context, eventID string) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM seats WHERE event_id = $1`, eventID).Scan(&n)
	return n, err
}
