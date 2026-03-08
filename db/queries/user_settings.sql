-- name: GetUserSettings :one
SELECT user_id, email_link_activity, email_weekly_digest, email_product_updates,
       profile_public, show_activity_status, allow_nsfw_content, created_at, updated_at
FROM user_settings
WHERE user_id = $1;

-- name: UpsertUserSettings :one
INSERT INTO user_settings (
    user_id,
    email_link_activity,
    email_weekly_digest,
    email_product_updates,
    profile_public,
    show_activity_status,
    allow_nsfw_content
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id) DO UPDATE SET
    email_link_activity = EXCLUDED.email_link_activity,
    email_weekly_digest = EXCLUDED.email_weekly_digest,
    email_product_updates = EXCLUDED.email_product_updates,
    profile_public = EXCLUDED.profile_public,
    show_activity_status = EXCLUDED.show_activity_status,
    allow_nsfw_content = EXCLUDED.allow_nsfw_content,
    updated_at = now()
RETURNING user_id, email_link_activity, email_weekly_digest, email_product_updates,
          profile_public, show_activity_status, allow_nsfw_content, created_at, updated_at;

-- name: CreateDefaultUserSettings :one
INSERT INTO user_settings (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING user_id, email_link_activity, email_weekly_digest, email_product_updates,
          profile_public, show_activity_status, allow_nsfw_content, created_at, updated_at;
