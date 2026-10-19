-- name: TryWorkerLock :one
SELECT pg_try_advisory_lock(7400319026::bigint)::boolean;

-- name: ReleaseWorkerLock :one
SELECT pg_advisory_unlock(7400319026::bigint)::boolean;

-- name: RecoverRenderJobs :execrows
UPDATE render_jobs SET status = 'PENDING', error_message = NULL
FROM orders WHERE orders.id = render_jobs.order_id
AND orders.status = 'PAID' AND render_jobs.status = 'PROCESSING';

-- name: ClaimNextRenderJob :one
UPDATE render_jobs SET status = 'PROCESSING', error_message = NULL
WHERE id = (
    SELECT j.id FROM render_jobs j JOIN orders o ON o.id = j.order_id
    WHERE j.status = 'PENDING' AND o.status = 'PAID'
    ORDER BY j.created_at, j.id FOR UPDATE OF j SKIP LOCKED LIMIT 1
)
RETURNING *;

-- name: GetRenderJobDelivery :one
SELECT users.tg_id, datasets.title AS dataset_title
FROM render_jobs
JOIN orders ON orders.id = render_jobs.order_id
JOIN users ON users.id = orders.user_id
JOIN datasets ON datasets.id = render_jobs.dataset_id
WHERE render_jobs.id = $1;

-- name: GetDatasetByID :one
SELECT * FROM datasets WHERE id = $1;

-- name: FinishRenderJob :execrows
UPDATE render_jobs SET status = 'COMPLETED', completed_at = NOW(), error_message = NULL
WHERE id = $1 AND status = 'PROCESSING';

-- name: RequeueRenderJob :execrows
UPDATE render_jobs SET status = 'PENDING', error_message = NULL
WHERE id = $1 AND status = 'PROCESSING';

-- name: FailRenderJob :execrows
UPDATE render_jobs SET status = 'FAILED', error_message = $2
WHERE id = $1 AND status = 'PROCESSING';
