package store

import "context"

// CreatePlatformFeePayment records the organizer's mock registration fee.
// Ticket payments are written inside CreateBooking; this one stands alone
// because a platform fee has no booking attached (enforced by the
// payment_reference_type CHECK constraint).
func (s *Store) CreatePlatformFeePayment(ctx context.Context, userID string, amount float64, reference string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO payments (user_id, type, amount, reference)
		VALUES ($1, 'platform_fee', $2, $3)`,
		userID, amount, reference,
	)
	return err
}
