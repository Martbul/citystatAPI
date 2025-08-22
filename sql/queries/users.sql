-- sql/queries/users.sql

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (
    id, email, first_name, last_name, user_name, image_url, phone_number, role, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: UpdateUser :one
UPDATE users 
SET 
    first_name = COALESCE($2, first_name),
    last_name = COALESCE($3, last_name),
    user_name = COALESCE($4, user_name),
    image_url = COALESCE($5, image_url),
    phone_number = COALESCE($6, phone_number),
    completed_tutorial = COALESCE($7, completed_tutorial),
    about_me = COALESCE($8, about_me),
    note = COALESCE($9, note),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SearchUsers :many
SELECT id, user_name, first_name, last_name, image_url
FROM users 
WHERE user_name ILIKE '%' || $2 || '%' 
AND id != $1
LIMIT 10;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;



