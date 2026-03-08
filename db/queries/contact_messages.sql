-- name: CreateContactMessage :one
INSERT INTO contact_messages (name, email, subject, message)
VALUES ($1, $2, $3, $4)
RETURNING id, name, email, subject, message, status, created_at;

-- name: ListContactMessages :many
SELECT id, name, email, subject, message, status, created_at
FROM contact_messages
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateContactMessageStatus :exec
UPDATE contact_messages
SET status = $2
WHERE id = $1;
