
-- sql/queries/friends.sql

-- name: GetUserFriends :many
SELECT 
    uf.id,
    uf.friend_id,
    uf.user_name,
    uf.first_name,
    uf.last_name,
    uf.image_url,
    uf.created_at
FROM user_friends uf
WHERE uf.user_id = $1
ORDER BY uf.created_at DESC;

-- name: CreateFriend :one
INSERT INTO user_friends (
    user_id, friend_id, user_name, first_name, last_name, image_url
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: CheckFriendshipExists :one
SELECT COUNT(*) FROM user_friends 
WHERE user_id = $1 AND friend_id = $2;

-- name: DeleteFriend :exec
DELETE FROM user_friends 
WHERE user_id = $1 AND friend_id = $2;

-- name: GetMutualFriends :many
SELECT DISTINCT f1.friend_id as mutual_friend_id
FROM user_friends f1
JOIN user_friends f2 ON f1.friend_id = f2.friend_id
WHERE f1.user_id = $1 AND f2.user_id = $2;