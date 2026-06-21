CREATE TABLE payments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- The payer, always. For a platform_fee that's the organizer; for a
    -- ticket that's the attendee.
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    -- Set only for ticket payments. NULL for the organizer platform fee.
    booking_id UUID REFERENCES bookings(id) ON DELETE RESTRICT,
    type       TEXT NOT NULL CHECK (type IN ('platform_fee', 'ticket')),
    amount     NUMERIC(10,2) NOT NULL CHECK (amount >= 0),
    status     TEXT NOT NULL DEFAULT 'completed'
                 CHECK (status IN ('pending', 'completed', 'failed')),
    reference  TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ties the payment type directly to whether booking_id is set:
    -- platform_fee => no booking; ticket => must have a booking. This is
    -- stricter than just "one of them is set" and prevents mismatched rows.
    CONSTRAINT payment_reference_type CHECK (
        (type = 'platform_fee' AND booking_id IS NULL)
     OR (type = 'ticket'        AND booking_id IS NOT NULL)
    )
);

CREATE INDEX idx_payments_user_id    ON payments(user_id);
CREATE INDEX idx_payments_booking_id ON payments(booking_id);

CREATE TRIGGER payments_set_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
