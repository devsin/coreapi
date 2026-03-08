CREATE TABLE IF NOT EXISTS user_settings (
    user_id              uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    -- Notification settings
    email_link_activity  boolean NOT NULL DEFAULT true,
    email_weekly_digest  boolean NOT NULL DEFAULT true,
    email_product_updates boolean NOT NULL DEFAULT false,
    -- Privacy settings
    profile_public       boolean NOT NULL DEFAULT true,
    show_activity_status boolean NOT NULL DEFAULT true,
    allow_nsfw_content   boolean NOT NULL DEFAULT false,
    -- Timestamps
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

DROP TRIGGER IF EXISTS user_settings_set_updated_at ON user_settings;
CREATE TRIGGER user_settings_set_updated_at
BEFORE UPDATE ON user_settings
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
