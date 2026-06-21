package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/MonalFinbox/seatrush/internal/models"
)

// ErrSeatUnavailable means one or more requested seats were not available at
// booking time (already booked, or not part of the event).
var ErrSeatUnavailable = errors.New("one or more seats are unavailable")

// bookingSelect aggregates the seat ids for a booking into a Postgres array so
// we can return them in one round trip.
const bookingSelect = `
	SELECT b.id, b.user_id, b.event_id, b.status, b.total_amount::float8,
	       COALESCE(array_agg(bs.seat_id) FILTER (WHERE bs.seat_id IS NOT NULL), '{}') AS seat_ids,
	       b.created_at, b.updated_at
	FROM bookings b
	LEFT JOIN booking_seats bs ON bs.booking_id = b.id`

func scanBooking(row pgx.Row) (*models.Booking, error) {
	var b models.Booking
	err := row.Scan(&b.ID, &b.UserID, &b.EventID, &b.Status, &b.TotalAmount,
		&b.SeatIDs, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// CreateBooking converts a set of held seats into a permanent booking in one
// transaction. It locks the seat rows FOR UPDATE, re-verifies they're all
// still available (defence in depth beyond the Redis hold), flips them to
// booked, writes the booking + join rows + ticket payment, and commits. Any
// failure rolls the whole thing back.
func (s *Store) CreateBooking(ctx context.Context, userID, eventID string, seatIDs []string, paymentRef string) (*models.Booking, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Lock the seats so no concurrent booking can touch them mid-transaction.
	rows, err := tx.Query(ctx, `
		SELECT id, price::float8, status
		FROM seats
		WHERE id = ANY($1) AND event_id = $2
		FOR UPDATE`,
		seatIDs, eventID,
	)
	if err != nil {
		return nil, err
	}

	var total float64
	found := 0
	for rows.Next() {
		var id, status string
		var price float64
		if err := rows.Scan(&id, &price, &status); err != nil {
			rows.Close()
			return nil, err
		}
		if status != "available" {
			rows.Close()
			return nil, ErrSeatUnavailable
		}
		total += price
		found++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if found != len(seatIDs) {
		return nil, ErrSeatUnavailable // some seat ids didn't belong to this event
	}

	if _, err := tx.Exec(ctx, `UPDATE seats SET status = 'booked' WHERE id = ANY($1)`, seatIDs); err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO bookings (user_id, event_id, total_amount)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, event_id, status, total_amount::float8, created_at, updated_at`,
		userID, eventID, total,
	)
	var b models.Booking
	if err := row.Scan(&b.ID, &b.UserID, &b.EventID, &b.Status, &b.TotalAmount, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}

	for _, seatID := range seatIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO booking_seats (booking_id, seat_id) VALUES ($1, $2)`, b.ID, seatID); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO payments (user_id, booking_id, type, amount, reference)
		VALUES ($1, $2, 'ticket', $3, $4)`,
		userID, b.ID, total, paymentRef,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	b.SeatIDs = seatIDs
	return &b, nil
}

// CancelBooking frees the seats and marks the booking cancelled in one
// transaction. It returns the freed seat ids so the caller can broadcast
// seat.released. The booking_seats rows are deleted, which both frees the
// unique(seat_id) slot and is the accepted trade-off of losing seat history.
func (s *Store) CancelBooking(ctx context.Context, bookingID string) ([]string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM bookings WHERE id = $1`, bookingID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status == "cancelled" {
		return nil, fmt.Errorf("booking already cancelled")
	}

	rows, err := tx.Query(ctx, `SELECT seat_id FROM booking_seats WHERE booking_id = $1`, bookingID)
	if err != nil {
		return nil, err
	}
	seatIDs := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		seatIDs = append(seatIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `UPDATE bookings SET status = 'cancelled' WHERE id = $1`, bookingID); err != nil {
		return nil, err
	}
	if len(seatIDs) > 0 {
		if _, err := tx.Exec(ctx, `UPDATE seats SET status = 'available' WHERE id = ANY($1)`, seatIDs); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM booking_seats WHERE booking_id = $1`, bookingID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return seatIDs, nil
}

func (s *Store) GetBooking(ctx context.Context, id string) (*models.Booking, error) {
	row := s.db.QueryRow(ctx, bookingSelect+` WHERE b.id = $1 GROUP BY b.id`, id)
	return scanBooking(row)
}

func (s *Store) ListBookingsByUser(ctx context.Context, userID string) ([]models.Booking, error) {
	rows, err := s.db.Query(ctx, bookingSelect+` WHERE b.user_id = $1 GROUP BY b.id ORDER BY b.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectBookings(rows)
}

func (s *Store) ListAllBookings(ctx context.Context) ([]models.Booking, error) {
	rows, err := s.db.Query(ctx, bookingSelect+` GROUP BY b.id ORDER BY b.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectBookings(rows)
}

func collectBookings(rows pgx.Rows) ([]models.Booking, error) {
	out := []models.Booking{}
	for rows.Next() {
		b, err := scanBooking(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}
