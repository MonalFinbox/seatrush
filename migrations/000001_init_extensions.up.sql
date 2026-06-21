-- citext gives us case-insensitive text. We use it for users.email so that
-- Monal@x.com and monal@x.com are treated as the same address automatically,
-- without every query needing LOWER(email).
CREATE EXTENSION IF NOT EXISTS citext;

-- A single trigger function reused by every table that has updated_at.
-- BEFORE UPDATE, it overwrites NEW.updated_at with the current time, so the
-- application never has to remember to set it. Postgres owns the timestamp.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
