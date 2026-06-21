CREATE TABLE events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    venue_id     UUID NOT NULL REFERENCES venues(id) ON DELETE RESTRICT,
    organizer_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title        TEXT NOT NULL,
    description  TEXT,
    event_date   TIMESTAMPTZ NOT NULL,
    status       TEXT NOT NULL DEFAULT 'draft'
                   CHECK (status IN ('draft', 'published', 'cancelled', 'completed')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_venue_id     ON events(venue_id);
CREATE INDEX idx_events_organizer_id ON events(organizer_id);

-- Attendees browse published events sorted by date. A composite index on
-- (status, event_date) serves the WHERE status = 'published' ORDER BY
-- event_date query directly, without a separate sort step.
CREATE INDEX idx_events_status_date ON events(status, event_date);

-- The headline business rule, enforced at the database level as a backstop
-- to app logic: one active event per venue. "Active" = anything not yet
-- cancelled or completed. Cancelling/completing an event frees the venue
-- for the next one, because those rows fall out of the index's WHERE clause.
CREATE UNIQUE INDEX uniq_events_one_active_per_venue
    ON events(venue_id)
    WHERE status NOT IN ('cancelled', 'completed');

CREATE TRIGGER events_set_updated_at
    BEFORE UPDATE ON events
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
