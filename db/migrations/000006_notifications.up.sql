CREATE TABLE notifications (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id    uuid REFERENCES users(id) ON DELETE SET NULL,
    type        text NOT NULL,
    entity_type text,
    entity_id   uuid,
    group_key   text,
    title       text NOT NULL,
    body        text,
    url         text,
    is_read     boolean NOT NULL DEFAULT false,
    read_at     timestamptz,
    metadata    jsonb,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_read_created
    ON notifications (user_id, is_read, created_at DESC);

CREATE INDEX idx_notifications_user_created
    ON notifications (user_id, created_at DESC);

CREATE INDEX idx_notifications_dedup
    ON notifications (user_id, actor_id, type, entity_id)
    WHERE is_read = FALSE;
