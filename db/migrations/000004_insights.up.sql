-- Profile view events with full client + geo details
CREATE TABLE IF NOT EXISTS profile_views (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    viewer_ip       text,
    referrer        text,
    user_agent      text,
    browser         text,
    os              text,
    device_type     text,
    country         text,
    country_code    text,
    city            text,
    region          text,
    latitude        double precision,
    longitude       double precision,
    viewed_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_profile_views_user_id ON profile_views(profile_user_id);
CREATE INDEX idx_profile_views_viewed_at ON profile_views(profile_user_id, viewed_at DESC);
CREATE INDEX idx_profile_views_country ON profile_views(country) WHERE country IS NOT NULL;
CREATE INDEX idx_profile_views_dedup ON profile_views(profile_user_id, viewer_ip, viewed_at DESC);

-- Pre-aggregated daily counters per user
CREATE TABLE IF NOT EXISTS daily_insights (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date           date NOT NULL,
    profile_views  integer NOT NULL DEFAULT 0,
    new_followers  integer NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE(user_id, date)
);

CREATE INDEX idx_daily_insights_user_date ON daily_insights(user_id, date DESC);
