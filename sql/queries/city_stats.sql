
-- sql/queries/city_stats.sql

-- name: GetUserCityStats :one
SELECT * FROM city_stats WHERE user_id = $1;

-- name: CreateCityStats :one
INSERT INTO city_stats (
    user_id, name, state, country, population, area
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: UpdateCityStats :one
UPDATE city_stats 
SET 
    total_streets_walked = COALESCE($2, total_streets_walked),
    total_kilometers = COALESCE($3, total_kilometers),
    city_coverage_pct = COALESCE($4, city_coverage_pct),
    days_active = COALESCE($5, days_active),
    longest_streak_days = COALESCE($6, longest_streak_days),
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;