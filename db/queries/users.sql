-- name: GetUser :one
SELECT id, username, display_name, bio, avatar_url, created_at, updated_at
FROM users
WHERE id = $1;

-- name: InsertUser :one
INSERT INTO users (id)
VALUES ($1)
ON CONFLICT (id) DO UPDATE SET id = EXCLUDED.id
RETURNING id, username, display_name, bio, avatar_url, created_at, updated_at;

-- name: UpdateUser :one
UPDATE users SET
  username = COALESCE(sqlc.narg('username'), username),
  display_name = COALESCE(sqlc.narg('display_name'), display_name),
  bio = COALESCE(sqlc.narg('bio'), bio),
  avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
  updated_at = now()
WHERE id = $1
RETURNING id, username, display_name, bio, avatar_url, created_at, updated_at;

-- name: GetUserByUsername :one
SELECT id, username, display_name, bio, avatar_url, created_at, updated_at
FROM users
WHERE username = $1;

-- name: IsUsernameTaken :one
SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND id != $2) AS taken;

-- name: DiscoverUsersPopular :many
SELECT u.id, u.username, u.display_name, u.bio, u.avatar_url, u.created_at, u.updated_at,
       COALESCE(fc.cnt, 0)::bigint AS follower_count
FROM users u
LEFT JOIN (SELECT following_id, COUNT(*) AS cnt FROM follows GROUP BY following_id) fc ON fc.following_id = u.id
WHERE u.username IS NOT NULL
  AND (sqlc.narg('exclude_user_id')::uuid IS NULL OR u.id != sqlc.narg('exclude_user_id')::uuid)
ORDER BY COALESCE(fc.cnt, 0) DESC, u.created_at DESC
LIMIT $1 OFFSET $2;

-- name: DiscoverUsersNew :many
SELECT u.id, u.username, u.display_name, u.bio, u.avatar_url, u.created_at, u.updated_at,
       COALESCE(fc.cnt, 0)::bigint AS follower_count
FROM users u
LEFT JOIN (SELECT following_id, COUNT(*) AS cnt FROM follows GROUP BY following_id) fc ON fc.following_id = u.id
WHERE u.username IS NOT NULL
  AND (sqlc.narg('exclude_user_id')::uuid IS NULL OR u.id != sqlc.narg('exclude_user_id')::uuid)
ORDER BY u.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountDiscoverUsers :one
SELECT COUNT(*) FROM users
WHERE username IS NOT NULL
  AND (sqlc.narg('exclude_user_id')::uuid IS NULL OR id != sqlc.narg('exclude_user_id')::uuid);
