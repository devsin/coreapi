-- name: FollowUser :exec
INSERT INTO follows (follower_id, following_id)
VALUES ($1, $2)
ON CONFLICT (follower_id, following_id) DO NOTHING;

-- name: UnfollowUser :exec
DELETE FROM follows
WHERE follower_id = $1 AND following_id = $2;

-- name: IsFollowing :one
SELECT EXISTS(
    SELECT 1 FROM follows 
    WHERE follower_id = $1 AND following_id = $2
) AS is_following;

-- name: GetFollowersCount :one
SELECT COUNT(*) FROM follows WHERE following_id = $1;

-- name: GetFollowingCount :one
SELECT COUNT(*) FROM follows WHERE follower_id = $1;

-- name: GetFollowers :many
SELECT u.id, u.username, u.display_name, u.bio, u.avatar_url, u.created_at, u.updated_at
FROM follows f
JOIN users u ON u.id = f.follower_id
WHERE f.following_id = $1
ORDER BY f.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetFollowing :many
SELECT u.id, u.username, u.display_name, u.bio, u.avatar_url, u.created_at, u.updated_at
FROM follows f
JOIN users u ON u.id = f.following_id
WHERE f.follower_id = $1
ORDER BY f.created_at DESC
LIMIT $2 OFFSET $3;
