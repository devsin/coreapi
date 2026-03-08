-- name: CreateNotification :one
INSERT INTO notifications (user_id, actor_id, type, entity_type, entity_id, group_key, title, body, url, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListNotificationsByUser :many
SELECT * FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUnreadNotifications :one
SELECT count(*) FROM notifications
WHERE user_id = $1 AND is_read = FALSE;

-- name: MarkNotificationRead :exec
UPDATE notifications
SET is_read = TRUE, read_at = now()
WHERE id = $1 AND user_id = $2;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications
SET is_read = TRUE, read_at = now()
WHERE user_id = $1 AND is_read = FALSE;

-- name: CheckDuplicateNotification :one
SELECT EXISTS(
    SELECT 1 FROM notifications
    WHERE user_id = $1
      AND actor_id = $2
      AND type = $3
      AND entity_id = $4
      AND is_read = FALSE
      AND created_at > now() - interval '24 hours'
) AS exists;

-- name: DeleteNotification :exec
DELETE FROM notifications
WHERE id = $1 AND user_id = $2;
