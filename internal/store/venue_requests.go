package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/MonalFinbox/seatrush/internal/models"
)

const vrrColumns = `id, venue_id, organizer_id, document_mock, status, reviewed_by, reviewed_at, rejection_reason, created_at, updated_at`

func scanVRR(row pgx.Row) (*models.VenueRegistrationRequest, error) {
	var r models.VenueRegistrationRequest
	err := row.Scan(&r.ID, &r.VenueID, &r.OrganizerID, &r.DocumentMock, &r.Status,
		&r.ReviewedBy, &r.ReviewedAt, &r.RejectionReason, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) CreateVenueRequest(ctx context.Context, venueID, organizerID, documentMock string) (*models.VenueRegistrationRequest, error) {
	row := s.db.QueryRow(ctx, `
		INSERT INTO venue_registration_requests (venue_id, organizer_id, document_mock)
		VALUES ($1, $2, $3)
		RETURNING `+vrrColumns,
		venueID, organizerID, documentMock,
	)
	return scanVRR(row)
}

func (s *Store) GetVenueRequest(ctx context.Context, id string) (*models.VenueRegistrationRequest, error) {
	row := s.db.QueryRow(ctx, `SELECT `+vrrColumns+` FROM venue_registration_requests WHERE id = $1`, id)
	return scanVRR(row)
}

func (s *Store) ListRequestsByOrganizer(ctx context.Context, organizerID string) ([]models.VenueRegistrationRequest, error) {
	rows, err := s.db.Query(ctx, `SELECT `+vrrColumns+`
		FROM venue_registration_requests WHERE organizer_id = $1 ORDER BY created_at DESC`, organizerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectVRR(rows)
}

// ListRequests returns all requests, optionally filtered by status ("" = all).
func (s *Store) ListRequests(ctx context.Context, status string) ([]models.VenueRegistrationRequest, error) {
	query := `SELECT ` + vrrColumns + ` FROM venue_registration_requests`
	args := []any{}
	if status != "" {
		query += ` WHERE status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectVRR(rows)
}

func collectVRR(rows pgx.Rows) ([]models.VenueRegistrationRequest, error) {
	out := []models.VenueRegistrationRequest{}
	for rows.Next() {
		r, err := scanVRR(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// ApproveRequest atomically marks the request approved and claims the venue
// for the organizer. Both happen in one transaction: if either fails, neither
// is committed, so a venue can never be left half-claimed.
func (s *Store) ApproveRequest(ctx context.Context, requestID, adminID string) (*models.VenueRegistrationRequest, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) // no-op after a successful Commit

	row := tx.QueryRow(ctx, `
		UPDATE venue_registration_requests
		SET status = 'approved', reviewed_by = $1, reviewed_at = NOW()
		WHERE id = $2 AND status = 'pending'
		RETURNING `+vrrColumns,
		adminID, requestID,
	)
	req, err := scanVRR(row)
	if err != nil {
		return nil, err // ErrNotFound if not pending / missing
	}

	// Claim the venue. The venue_claim_consistency CHECK requires both columns
	// move together, which this does.
	_, err = tx.Exec(ctx, `
		UPDATE venues SET claim_status = 'claimed', organizer_id = $1
		WHERE id = $2 AND claim_status = 'unclaimed'`,
		req.OrganizerID, req.VenueID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Store) RejectRequest(ctx context.Context, requestID, adminID, reason string) (*models.VenueRegistrationRequest, error) {
	row := s.db.QueryRow(ctx, `
		UPDATE venue_registration_requests
		SET status = 'rejected', reviewed_by = $1, reviewed_at = NOW(), rejection_reason = $2
		WHERE id = $3 AND status = 'pending'
		RETURNING `+vrrColumns,
		adminID, reason, requestID,
	)
	return scanVRR(row)
}
