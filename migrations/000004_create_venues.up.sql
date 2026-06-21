CREATE TABLE venues (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    address      TEXT NOT NULL,
    city         TEXT NOT NULL,
    capacity     INT NOT NULL CHECK (capacity > 0),
    claim_status TEXT NOT NULL DEFAULT 'unclaimed'
                   CHECK (claim_status IN ('unclaimed', 'claimed')),
    -- NULL while unclaimed; set to the owning organizer once claimed.
    -- RESTRICT: a user who owns a venue can't be deleted out from under it.
    organizer_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- claim_status and organizer_id must always move together. A claimed
    -- venue has an owner; an unclaimed one has none. This stops the two
    -- columns from silently drifting out of sync after a buggy approval.
    CONSTRAINT venue_claim_consistency CHECK (
        (claim_status = 'claimed'   AND organizer_id IS NOT NULL)
     OR (claim_status = 'unclaimed' AND organizer_id IS NULL)
    )
);

CREATE INDEX idx_venues_claim_status ON venues(claim_status);
CREATE INDEX idx_venues_organizer_id ON venues(organizer_id);

CREATE TRIGGER venues_set_updated_at
    BEFORE UPDATE ON venues
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
