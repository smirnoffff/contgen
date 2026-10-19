-- name: FindOrCreateUser :one
INSERT INTO users (tg_id, username, first_name, language)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tg_id) DO UPDATE
SET username = EXCLUDED.username, first_name = EXCLUDED.first_name
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;
