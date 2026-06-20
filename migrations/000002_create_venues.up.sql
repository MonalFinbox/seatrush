CREATE TABLE IF NOT EXISTS venues (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    address      TEXT NOT NULL,
    city         TEXT NOT NULL,
    capacity     INT NOT NULL CHECK (capacity > 0),
    claim_status TEXT NOT NULL DEFAULT 'unclaimed' CHECK (claim_status IN ('unclaimed', 'claimed')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
