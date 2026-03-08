-- name: SearchUsers :many
SELECT u.id, u.username, u.display_name, u.bio, u.avatar_url, u.created_at, u.updated_at,
       COALESCE(fc.cnt, 0)::bigint AS follower_count,
       ts_rank(u.search_tsv, to_tsquery('english', @query::text)) AS rank
FROM users u
LEFT JOIN (SELECT following_id, COUNT(*) AS cnt FROM follows GROUP BY following_id) fc ON fc.following_id = u.id
WHERE u.username IS NOT NULL
  AND (u.search_tsv @@ to_tsquery('english', @query::text)
       OR u.username ILIKE @ilike_query::text
       OR u.display_name ILIKE @ilike_query::text)
ORDER BY rank DESC, COALESCE(fc.cnt, 0) DESC
LIMIT $1 OFFSET $2;

-- name: CountSearchUsers :one
SELECT COUNT(*)
FROM users
WHERE username IS NOT NULL
  AND (search_tsv @@ to_tsquery('english', @query::text)
       OR username ILIKE @ilike_query::text
       OR display_name ILIKE @ilike_query::text);
