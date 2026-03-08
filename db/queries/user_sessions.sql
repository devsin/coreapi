-- name: UpsertSession :exec
INSERT INTO user_sessions (user_id, session_id, ip_address, user_agent, browser, os, device_type, last_active_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, now())
ON CONFLICT (user_id, session_id) DO UPDATE
SET ip_address    = EXCLUDED.ip_address,
    user_agent    = EXCLUDED.user_agent,
    browser       = EXCLUDED.browser,
    os            = EXCLUDED.os,
    device_type   = EXCLUDED.device_type,
    last_active_at = now();

-- name: ListSessionsByUserID :many
SELECT id, user_id, session_id, ip_address, user_agent, browser, os, device_type, last_active_at, created_at
FROM user_sessions
WHERE user_id = $1
ORDER BY last_active_at DESC;

-- name: DeleteSession :exec
DELETE FROM user_sessions
WHERE id = $1 AND user_id = $2;

-- name: DeleteOtherSessions :exec
DELETE FROM user_sessions
WHERE user_id = $1 AND session_id != $2;

-- name: DeleteAllUserSessions :exec
DELETE FROM user_sessions
WHERE user_id = $1;

-- name: CleanupStaleSessions :exec
DELETE FROM user_sessions
WHERE last_active_at < now() - interval '30 days';
