CREATE TABLE venue_registration_requests (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    venue_id         UUID NOT NULL REFERENCES venues(id) ON DELETE RESTRICT,
    organizer_id     UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    document_mock    TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'approved', 'rejected')),
    -- The admin who acted on the request. SET NULL so deleting an admin
    -- doesn't erase the request's history — we just lose the reviewer link.
    reviewed_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at      TIMESTAMPTZ,
    rejection_reason TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vrr_venue_id     ON venue_registration_requests(venue_id);
CREATE INDEX idx_vrr_organizer_id ON venue_registration_requests(organizer_id);
CREATE INDEX idx_vrr_status       ON venue_registration_requests(status);

-- The race-condition guard: at most ONE pending request per venue at a time.
-- A partial unique index applies the uniqueness rule only to rows where
-- status = 'pending'. Approved/rejected rows are exempt, so a venue can
-- accumulate history but never two live claims competing for it.
CREATE UNIQUE INDEX uniq_vrr_one_pending_per_venue
    ON venue_registration_requests(venue_id)
    WHERE status = 'pending';

CREATE TRIGGER vrr_set_updated_at
    BEFORE UPDATE ON venue_registration_requests
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
