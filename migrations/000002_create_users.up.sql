CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         CITEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('admin', 'organizer', 'attendee')),
    status        TEXT NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Only these two status values are ever valid.
    CONSTRAINT users_status_values CHECK (status IN ('active', 'pending_payment')),

    -- pending_payment is an organizer-only state. An admin or attendee must
    -- always be 'active' — this stops a bug from parking a non-organizer in
    -- the onboarding limbo state.
    CONSTRAINT users_status_role CHECK (role = 'organizer' OR status = 'active')
);

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
