-- ============================================================
-- Profile view recording & dedup
-- ============================================================

-- name: RecordProfileView :exec
INSERT INTO profile_views (profile_user_id, viewer_ip, referrer, user_agent, browser, os, device_type, country, country_code, city, region, latitude, longitude)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: CheckDuplicateProfileView :one
SELECT EXISTS(
    SELECT 1 FROM profile_views
    WHERE profile_user_id = $1
      AND viewer_ip = $2
      AND viewed_at > NOW() - INTERVAL '30 minutes'
) AS is_duplicate;

-- name: CountProfileViewsSince :one
SELECT COUNT(*) FROM profile_views
WHERE profile_user_id = $1 AND viewed_at >= $2;

-- name: CountProfileViewsBetween :one
SELECT COUNT(*) FROM profile_views
WHERE profile_user_id = $1 AND viewed_at >= $2 AND viewed_at < $3;

-- ============================================================
-- Daily insights
-- ============================================================

-- name: GetDailyInsights :many
SELECT id, user_id, date, profile_views, new_followers, created_at
FROM daily_insights
WHERE user_id = $1 AND date >= $2 AND date <= $3
ORDER BY date ASC;

-- name: UpsertDailyInsights :exec
INSERT INTO daily_insights (user_id, date, profile_views, new_followers)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, date) DO UPDATE SET
    profile_views = EXCLUDED.profile_views,
    new_followers = EXCLUDED.new_followers;

-- name: IncrementDailyProfileViews :exec
INSERT INTO daily_insights (user_id, date, profile_views)
VALUES ($1, CURRENT_DATE, 1)
ON CONFLICT (user_id, date) DO UPDATE SET
    profile_views = daily_insights.profile_views + 1;

-- ============================================================
-- Backfill queries — aggregate raw events into daily_insights
-- ============================================================

-- name: BackfillDailyProfileViews :exec
INSERT INTO daily_insights (user_id, date, profile_views)
SELECT pv.profile_user_id, date_trunc('day', pv.viewed_at)::date, COUNT(*)::integer
FROM profile_views pv
WHERE pv.viewed_at >= $1 AND pv.viewed_at < $2
GROUP BY pv.profile_user_id, date_trunc('day', pv.viewed_at)::date
ON CONFLICT (user_id, date) DO UPDATE SET
    profile_views = EXCLUDED.profile_views;

-- name: BackfillDailyNewFollowers :exec
INSERT INTO daily_insights (user_id, date, new_followers)
SELECT f.following_id, date_trunc('day', f.created_at)::date, COUNT(*)::integer
FROM follows f
WHERE f.created_at >= $1 AND f.created_at < $2
GROUP BY f.following_id, date_trunc('day', f.created_at)::date
ON CONFLICT (user_id, date) DO UPDATE SET
    new_followers = EXCLUDED.new_followers;

-- ============================================================
-- Follower counting
-- ============================================================

-- name: GetNewFollowersSince :one
SELECT COUNT(*) FROM follows
WHERE following_id = $1 AND created_at >= $2;

-- name: GetNewFollowersBetween :one
SELECT COUNT(*) FROM follows
WHERE following_id = $1 AND created_at >= $2 AND created_at < $3;

-- ============================================================
-- Daily profile view breakdown (for time series charts)
-- ============================================================

-- name: GetDailyProfileViews :many
SELECT date_trunc('day', viewed_at)::date AS day, COUNT(*) AS views
FROM profile_views
WHERE profile_user_id = $1 AND viewed_at >= $2 AND viewed_at <= $3
GROUP BY day
ORDER BY day ASC;

-- ============================================================
-- Event listing queries (for the detailed events page)
-- ============================================================

-- name: GetRecentProfileViewEvents :many
SELECT id, profile_user_id, viewer_ip, referrer, user_agent, browser, os, device_type,
    country, country_code, city, region, latitude, longitude, viewed_at
FROM profile_views
WHERE profile_user_id = $1
ORDER BY viewed_at DESC
LIMIT $2 OFFSET $3;

-- name: CountProfileViewEvents :one
SELECT COUNT(*) FROM profile_views WHERE profile_user_id = $1;

-- ============================================================
-- Geo aggregation queries (for the map visualization)
-- ============================================================

-- name: GetProfileViewGeoAggregation :many
SELECT country, country_code, city,
    COALESCE(AVG(latitude), 0)::double precision as latitude,
    COALESCE(AVG(longitude), 0)::double precision as longitude,
    COUNT(*) as event_count
FROM profile_views
WHERE profile_user_id = $1 AND latitude IS NOT NULL AND country IS NOT NULL
GROUP BY country, country_code, city
ORDER BY event_count DESC;

-- ============================================================
-- Browser / OS / Device / Referrer breakdown queries
-- ============================================================

-- name: GetProfileViewBrowserCounts :many
SELECT browser, COUNT(*) AS event_count
FROM profile_views
WHERE profile_user_id = $1 AND viewed_at >= $2 AND browser IS NOT NULL AND browser != ''
GROUP BY browser;

-- name: GetProfileViewOSCounts :many
SELECT os, COUNT(*) AS event_count
FROM profile_views
WHERE profile_user_id = $1 AND viewed_at >= $2 AND os IS NOT NULL AND os != ''
GROUP BY os;

-- name: GetProfileViewDeviceCounts :many
SELECT device_type, COUNT(*) AS event_count
FROM profile_views
WHERE profile_user_id = $1 AND viewed_at >= $2 AND device_type IS NOT NULL AND device_type != ''
GROUP BY device_type;

-- name: GetProfileViewReferrerCounts :many
SELECT referrer, COUNT(*) AS event_count
FROM profile_views
WHERE profile_user_id = $1 AND viewed_at >= $2 AND referrer IS NOT NULL AND referrer != ''
GROUP BY referrer;