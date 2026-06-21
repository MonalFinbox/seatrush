CREATE TABLE bookings (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- RESTRICT on both: booking history is never destroyed by deleting the
    -- user or the event it belongs to.
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    event_id     UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    status       TEXT NOT NULL DEFAULT 'confirmed'
                   CHECK (status IN ('confirmed', 'cancelled')),
    total_amount NUMERIC(10,2) NOT NULL CHECK (total_amount >= 0),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bookings_user_id  ON bookings(user_id);
CREATE INDEX idx_bookings_event_id ON bookings(event_id);

CREATE TRIGGER bookings_set_updated_at
    BEFORE UPDATE ON bookings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
