CREATE TABLE IF NOT EXISTS user_sessions (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id     text NOT NULL,
    ip_address     text,
    user_agent     text,
    browser        text,
    os             text,
    device_type    text,
    last_active_at timestamptz NOT NULL DEFAULT now(),
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE(user_id, session_id)
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_last_active ON user_sessions(user_id, last_active_at DESC);
