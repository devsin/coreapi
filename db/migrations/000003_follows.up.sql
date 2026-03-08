CREATE TABLE IF NOT EXISTS follows (
    follower_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    following_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (follower_id, following_id),
    CHECK (follower_id != following_id)
);

CREATE INDEX idx_follows_follower_id ON follows(follower_id);
CREATE INDEX idx_follows_following_id ON follows(following_id);
