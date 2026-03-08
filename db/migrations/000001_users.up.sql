CREATE TABLE IF NOT EXISTS users (
    id          uuid PRIMARY KEY,
    username    text UNIQUE,
    display_name text,
    bio         text,
    avatar_url  text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    search_tsv  tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', COALESCE(username, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(display_name, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(bio, '')), 'C')
    ) STORED
);

CREATE INDEX IF NOT EXISTS idx_users_search_tsv ON users USING GIN (search_tsv);

-- Auto-update updated_at on row change
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS users_set_updated_at ON users;
CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
