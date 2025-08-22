
-- sql/queries/visited_streets.sql

-- name: GetVisitedStreets :many
SELECT * FROM visited_streets 
WHERE user_id = $1
ORDER BY entry_timestamp DESC;

-- name: CreateVisitedStreet :one
INSERT INTO visited_streets (
    user_id, session_id, street_id, street_name, entry_timestamp,
    exit_timestamp, duration_seconds, entry_latitude, entry_longitude
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: CheckVisitedStreetExists :one
SELECT COUNT(*) FROM visited_streets 
WHERE user_id = $1 AND session_id = $2 AND street_id = $3 AND entry_timestamp = $4;

-- name: GetVisitedStreetsBySession :many
SELECT * FROM visited_streets 
WHERE user_id = $1 AND session_id = $2
ORDER BY entry_timestamp ASC;