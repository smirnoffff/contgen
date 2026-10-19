-- name: CreateRenderJob :one
INSERT INTO render_jobs (order_id, dataset_id, music_track_id, theme, status)
VALUES ($1, $2, $3, $4, 'PENDING')
RETURNING *;

-- name: GetJobByID :one
SELECT * FROM render_jobs WHERE id = $1;

-- name: SetJobCompleted :execrows
UPDATE render_jobs
SET status = 'COMPLETED', telegram_file_id = $2, completed_at = NOW()
WHERE id = $1;

-- name: MarkJobFailed :exec
UPDATE render_jobs SET status = 'FAILED', error_message = $2
WHERE id = $1;

-- name: SetJobTelegramFile :execrows
UPDATE render_jobs SET telegram_file_id = $2 WHERE id = $1 AND status = 'COMPLETED';
