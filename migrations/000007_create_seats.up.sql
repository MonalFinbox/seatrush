CREATE TABLE seats (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- CASCADE: deleting an event wipes its entire seat map automatically.
    event_id   UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    section    TEXT NOT NULL,
    -- seat_row, not "row": row is a reserved word in Postgres and would need
    -- double-quoting in every query forever.
    seat_row   TEXT NOT NULL,
    number     TEXT NOT NULL,
    price      NUMERIC(10,2) NOT NULL CHECK (price >= 0),
    -- Postgres only ever knows 'available' or 'booked'. The transient 'held'
    -- state lives exclusively in Redis — Redis is its single source of truth.
    status     TEXT NOT NULL DEFAULT 'available'
                 CHECK (status IN ('available', 'booked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- No two seats in the same event can share the same physical position.
    CONSTRAINT uniq_seat_in_event UNIQUE (event_id, section, seat_row, number)
);

-- Seat-map reads filter by event and often by status; this composite index
-- serves both.
CREATE INDEX idx_seats_event_status ON seats(event_id, status);

CREATE TRIGGER seats_set_updated_at
    BEFORE UPDATE ON seats
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
