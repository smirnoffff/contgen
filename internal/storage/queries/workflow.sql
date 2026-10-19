-- name: EnsureCart :one
INSERT INTO orders (user_id, status) VALUES ($1, 'CART')
ON CONFLICT (user_id) WHERE status = 'CART'
DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING *;

-- name: LockOwnedOrder :one
SELECT * FROM orders WHERE id = $1 AND user_id = $2 FOR UPDATE;

-- name: LockUser :one
SELECT * FROM users WHERE id = $1 FOR UPDATE;

-- name: CountOrderItems :one
SELECT count(*) FROM render_jobs WHERE order_id = $1;

-- name: DeleteCartItem :execrows
DELETE FROM render_jobs WHERE id = $1 AND order_id = $2 AND status = 'PENDING';

-- name: GetCartItemOrder :one
SELECT o.* FROM orders o
JOIN render_jobs j ON j.order_id = o.id
WHERE j.id = sqlc.arg(job_id) AND o.user_id = sqlc.arg(user_id) AND o.status = 'CART'
FOR UPDATE OF o;

-- name: ListCartItems :many
SELECT sqlc.embed(render_jobs), datasets.title AS dataset_title, music_tracks.title AS music_title
FROM render_jobs
JOIN orders ON orders.id = render_jobs.order_id
JOIN datasets ON datasets.id = render_jobs.dataset_id
LEFT JOIN music_tracks ON music_tracks.id = render_jobs.music_track_id
WHERE orders.user_id = $1 AND orders.status = 'CART'
ORDER BY render_jobs.created_at, render_jobs.id;

-- name: ListUserOrders :many
SELECT sqlc.embed(orders), count(render_jobs.id)::bigint AS item_count,
       row_number() OVER (ORDER BY orders.created_at, orders.id)::bigint AS order_number
FROM orders LEFT JOIN render_jobs ON render_jobs.order_id = orders.id
WHERE orders.user_id = $1 AND orders.status <> 'CART'
GROUP BY orders.id ORDER BY orders.created_at DESC, orders.id DESC LIMIT 100;

-- name: ListUserJobs :many
SELECT sqlc.embed(render_jobs), datasets.title AS dataset_title, music_tracks.title AS music_title,
orders.status AS order_status
FROM render_jobs
JOIN orders ON orders.id = render_jobs.order_id
JOIN datasets ON datasets.id = render_jobs.dataset_id
LEFT JOIN music_tracks ON music_tracks.id = render_jobs.music_track_id
WHERE orders.user_id = $1 AND orders.status = 'PAID'
ORDER BY render_jobs.created_at DESC, render_jobs.id DESC LIMIT 100;

-- name: GetOwnedJob :one
SELECT sqlc.embed(render_jobs), datasets.title AS dataset_title, music_tracks.title AS music_title,
orders.status AS order_status
FROM render_jobs
JOIN orders ON orders.id = render_jobs.order_id
JOIN datasets ON datasets.id = render_jobs.dataset_id
LEFT JOIN music_tracks ON music_tracks.id = render_jobs.music_track_id
WHERE render_jobs.id = sqlc.arg(id) AND orders.user_id = sqlc.arg(user_id);

-- name: RetryOwnedJob :execrows
UPDATE render_jobs SET status = 'PENDING', error_message = NULL,
completed_at = NULL, telegram_file_id = NULL
FROM orders WHERE orders.id = render_jobs.order_id
AND render_jobs.id = sqlc.arg(id) AND orders.user_id = sqlc.arg(user_id)
AND orders.status = 'PAID' AND render_jobs.status = 'FAILED';

-- name: CountUnusableCartItems :one
SELECT count(*) FROM render_jobs j
JOIN datasets d ON d.id = j.dataset_id
LEFT JOIN music_tracks m ON m.id = j.music_track_id
WHERE j.order_id = sqlc.arg(order_id) AND (
    (d.user_id IS NOT NULL AND d.user_id <> sqlc.arg(user_id))
    OR NOT EXISTS (SELECT 1 FROM data_points p WHERE p.dataset_id = d.id)
    OR (j.music_track_id IS NOT NULL AND (NOT m.is_active OR (m.user_id IS NOT NULL AND m.user_id <> sqlc.arg(user_id))))
);

-- name: DatasetHasPoints :one
SELECT EXISTS (SELECT 1 FROM data_points WHERE dataset_id = $1);
