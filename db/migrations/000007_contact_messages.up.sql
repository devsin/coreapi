CREATE TABLE IF NOT EXISTS contact_messages (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    email      text NOT NULL,
    subject    text NOT NULL DEFAULT 'general',
    message    text NOT NULL,
    status     text NOT NULL DEFAULT 'new',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_contact_messages_status ON contact_messages (status);
CREATE INDEX idx_contact_messages_created_at ON contact_messages (created_at DESC);
