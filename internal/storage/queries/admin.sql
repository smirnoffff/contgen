-- name: AdminSummary :one
WITH visible_users AS (
    SELECT * FROM users WHERE tg_id > 0 OR sqlc.arg(include_demo)::boolean
), visible_orders AS (
    SELECT o.* FROM orders o JOIN visible_users u ON u.id = o.user_id
), visible_jobs AS (
    SELECT j.* FROM render_jobs j JOIN visible_orders o ON o.id = j.order_id WHERE o.status = 'PAID'
)
SELECT (SELECT count(*) FROM visible_users)::bigint AS users,
(SELECT count(*) FROM visible_orders WHERE status = 'PAID')::bigint AS orders,
(SELECT count(*) FROM visible_orders WHERE status = 'CART')::bigint AS carts,
(SELECT count(*) FROM visible_jobs WHERE status = 'PENDING')::bigint AS pending_jobs,
(SELECT count(*) FROM visible_jobs WHERE status = 'PROCESSING')::bigint AS processing_jobs,
(SELECT count(*) FROM visible_jobs WHERE status = 'COMPLETED')::bigint AS completed_jobs,
(SELECT count(*) FROM visible_jobs WHERE status = 'FAILED')::bigint AS failed_jobs,
(SELECT count(*) FROM visible_jobs WHERE status = 'COMPLETED' AND telegram_file_id IS NOT NULL)::bigint AS delivered_jobs,
(SELECT count(*) FROM visible_orders WHERE status = 'PAID' AND payment_method = 'ADS')::bigint AS ad_orders,
(SELECT count(*) FROM visible_orders WHERE status = 'PAID' AND payment_method = 'PROMO')::bigint AS promo_orders;

-- name: AdminListUsers :many
SELECT * FROM users WHERE tg_id > 0 OR sqlc.arg(include_demo)::boolean
ORDER BY created_at DESC, id DESC LIMIT 100;

-- name: AdminCountOrders :one
SELECT count(*) FROM orders o JOIN users u ON u.id = o.user_id
WHERE (u.tg_id > 0 OR sqlc.arg(include_demo)::boolean)
AND (sqlc.arg(order_status)::text = '' OR o.status::text = sqlc.arg(order_status))
AND (sqlc.arg(search)::text = '' OR u.first_name ILIKE '%' || sqlc.arg(search) || '%'
    OR u.username ILIKE '%' || sqlc.arg(search) || '%'
    OR ('@' || u.username) ILIKE '%' || sqlc.arg(search) || '%' OR o.id::text = sqlc.arg(search)
    OR u.tg_id::text = sqlc.arg(search));

-- name: AdminListOrders :many
SELECT sqlc.embed(o), u.first_name, u.username, u.tg_id,
(SELECT count(*) FROM orders older WHERE older.user_id = o.user_id AND older.status <> 'CART'
    AND (older.created_at, older.id) <= (o.created_at, o.id))::bigint AS order_number,
(SELECT count(*) FROM render_jobs j WHERE j.order_id = o.id)::bigint AS item_count,
COALESCE((SELECT sum(p.amount) FROM payments p WHERE p.order_id = o.id
    AND p.method = 'STARS' AND p.status = 'SUCCEEDED' AND NULLIF(p.external_id, '') IS NOT NULL), 0)::numeric AS stars_received
FROM orders o JOIN users u ON u.id = o.user_id
WHERE (u.tg_id > 0 OR sqlc.arg(include_demo)::boolean)
AND (sqlc.arg(order_status)::text = '' OR o.status::text = sqlc.arg(order_status))
AND (sqlc.arg(search)::text = '' OR u.first_name ILIKE '%' || sqlc.arg(search) || '%'
    OR u.username ILIKE '%' || sqlc.arg(search) || '%'
    OR ('@' || u.username) ILIKE '%' || sqlc.arg(search) || '%' OR o.id::text = sqlc.arg(search)
    OR u.tg_id::text = sqlc.arg(search))
ORDER BY o.created_at DESC, o.id DESC
LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;

-- name: AdminListJobs :many
SELECT sqlc.embed(render_jobs), datasets.title AS dataset_title, music_tracks.title AS music_title
FROM render_jobs JOIN datasets ON datasets.id = render_jobs.dataset_id
LEFT JOIN music_tracks ON music_tracks.id = render_jobs.music_track_id
WHERE render_jobs.order_id = ANY(sqlc.arg(order_ids)::bigint[])
ORDER BY render_jobs.created_at, render_jobs.id;

-- name: GetAdminJob :one
SELECT sqlc.embed(render_jobs), datasets.title AS dataset_title, music_tracks.title AS music_title
FROM render_jobs JOIN orders ON orders.id = render_jobs.order_id
JOIN datasets ON datasets.id = render_jobs.dataset_id
LEFT JOIN music_tracks ON music_tracks.id = render_jobs.music_track_id
WHERE render_jobs.id = $1 AND orders.status = 'PAID';

-- name: AdminRevenue :one
SELECT COALESCE(sum(p.amount), 0)::numeric AS stars_received, count(*)::bigint AS stars_payments
FROM payments p JOIN orders o ON o.id = p.order_id JOIN users u ON u.id = o.user_id
WHERE p.status = 'SUCCEEDED' AND p.method = 'STARS' AND NULLIF(p.external_id, '') IS NOT NULL
AND o.status = 'PAID' AND (u.tg_id > 0 OR sqlc.arg(include_demo)::boolean);

-- name: AdminListDatasets :many
SELECT sqlc.embed(datasets), users.first_name AS owner_name, count(data_points.id)::bigint AS point_count
FROM datasets LEFT JOIN users ON users.id = datasets.user_id
LEFT JOIN data_points ON data_points.dataset_id = datasets.id
WHERE datasets.user_id IS NULL OR users.tg_id > 0 OR sqlc.arg(include_demo)::boolean
GROUP BY datasets.id, users.id ORDER BY datasets.created_at DESC, datasets.id DESC LIMIT 100;

-- name: AdminListMusicTracks :many
SELECT sqlc.embed(music_tracks), users.first_name AS owner_name
FROM music_tracks LEFT JOIN users ON users.id = music_tracks.user_id
WHERE music_tracks.user_id IS NULL OR users.tg_id > 0 OR sqlc.arg(include_demo)::boolean
ORDER BY music_tracks.created_at DESC, music_tracks.id DESC LIMIT 100;

-- name: AdminListPromocodes :many
SELECT * FROM promocodes ORDER BY created_at DESC, id DESC LIMIT 100;

-- name: CreatePromocode :one
INSERT INTO promocodes (code, discount_percent, valid_until, max_uses)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: AdminSetPromocodeActive :execrows
UPDATE promocodes SET is_active = $2 WHERE id = $1;

-- name: AdminSetMusicActive :execrows
UPDATE music_tracks SET is_active = $2 WHERE id = $1;

-- name: AdminPublishDataset :execrows
UPDATE datasets SET user_id = NULL WHERE id = $1;

-- name: AdminPublishMusic :execrows
UPDATE music_tracks SET user_id = NULL, is_active = TRUE WHERE id = $1;

-- name: SetUserRoleByTelegramID :execrows
UPDATE users SET role = $2 WHERE tg_id = $1;

-- name: PromoteTelegramAdmins :execrows
UPDATE users SET role = 'ADMIN' WHERE tg_id = ANY(sqlc.arg(telegram_ids)::bigint[]) AND role <> 'ADMIN';

-- name: CountSystemDatasets :one
SELECT count(*) FROM datasets WHERE user_id IS NULL;

-- name: CountSystemMusicTracks :one
SELECT count(*) FROM music_tracks WHERE user_id IS NULL;

-- name: GrantAdminByTelegramID :execrows
UPDATE users SET role = 'ADMIN' WHERE tg_id = $1;

-- name: AdminSetUserRole :execrows
UPDATE users SET role = $2 WHERE id = $1;

-- name: GetPromocodeByID :one
SELECT * FROM promocodes WHERE id = $1;

-- name: AdminRetryJob :execrows
UPDATE render_jobs SET status = 'PENDING', error_message = NULL,
completed_at = NULL, telegram_file_id = NULL
FROM orders WHERE orders.id = render_jobs.order_id
AND render_jobs.id = $1 AND orders.status = 'PAID' AND render_jobs.status = 'FAILED';
