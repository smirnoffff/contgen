-- name: GetAvailableDatasets :many
SELECT * FROM datasets
WHERE user_id IS NULL OR user_id = $1
ORDER BY created_at DESC;

-- name: GetSystemDatasetByTitle :one
SELECT * FROM datasets
WHERE user_id IS NULL AND title = $1;

-- name: RenameSystemDataset :execrows
UPDATE datasets SET title = $2
WHERE id = $1 AND user_id IS NULL;

-- name: CreateDataset :one
INSERT INTO datasets (user_id, title) VALUES ($1, $2)
RETURNING *;

-- name: CopyDataPoints :copyfrom
INSERT INTO data_points (dataset_id, entity_name, value, recorded_at)
VALUES ($1, $2, $3, $4);

-- name: GetDataPointsByDatasetID :many
SELECT * FROM data_points WHERE dataset_id = $1
ORDER BY recorded_at ASC, id ASC;

-- name: GetDatasetForUser :one
SELECT * FROM datasets
WHERE id = $1 AND (user_id IS NULL OR user_id = sqlc.arg(viewer_id)::bigint);

-- name: DeleteOwnedDataset :execrows
DELETE FROM datasets
WHERE datasets.id = $1 AND datasets.user_id = $2
  AND NOT EXISTS (SELECT 1 FROM render_jobs WHERE dataset_id = datasets.id);
