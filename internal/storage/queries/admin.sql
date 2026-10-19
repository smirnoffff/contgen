-- name: AdminSummary :one
SELECT (SELECT count(*) FROM users)::bigint AS users,
(SELECT count(*) FROM orders WHERE status = 'PAID')::bigint AS orders,
(SELECT count(*) FROM render_jobs WHERE status = 'PENDING' AND order_id IN (SELECT id FROM orders WHERE status = 'PAID'))::bigint AS pending_jobs,
(SELECT count(*) FROM render_jobs WHERE status = 'PROCESSING')::bigint AS processing_jobs,
(SELECT count(*) FROM render_jobs WHERE status = 'COMPLETED')::bigint AS completed_jobs,
(SELECT count(*) FROM render_jobs WHERE status = 'FAILED')::bigint AS failed_jobs,
(SELECT count(*) FROM datasets)::bigint AS datasets,
(SELECT count(*) FROM music_tracks)::bigint AS music_tracks,
(SELECT count(*) FROM promocodes)::bigint AS promocodes;

-- name: AdminListUsers :many
SELECT * FROM users ORDER BY created_at DESC, id DESC LIMIT 100;

-- name: AdminListOrders :many
SELECT sqlc.embed(orders), users.first_name, users.tg_id, count(render_jobs.id)::bigint AS item_count
FROM orders JOIN users ON users.id = orders.user_id
LEFT JOIN render_jobs ON render_jobs.order_id = orders.id
GROUP BY orders.id, users.id ORDER BY orders.created_at DESC, orders.id DESC LIMIT 100;

-- name: AdminListJobs :many
SELECT sqlc.embed(render_jobs), users.first_name, datasets.title AS dataset_title
FROM render_jobs JOIN orders ON orders.id = render_jobs.order_id
JOIN users ON users.id = orders.user_id JOIN datasets ON datasets.id = render_jobs.dataset_id
WHERE orders.status = 'PAID'
ORDER BY render_jobs.created_at DESC, render_jobs.id DESC LIMIT 100;

-- name: GetAdminJob :one
SELECT sqlc.embed(render_jobs), datasets.title AS dataset_title, music_tracks.title AS music_title
FROM render_jobs JOIN orders ON orders.id = render_jobs.order_id
JOIN datasets ON datasets.id = render_jobs.dataset_id
LEFT JOIN music_tracks ON music_tracks.id = render_jobs.music_track_id
WHERE render_jobs.id = $1 AND orders.status = 'PAID';

-- name: AdminRevenue :one
SELECT COALESCE(sum(total_amount), 0)::numeric AS amount FROM orders WHERE status = 'PAID';

-- name: AdminListDatasets :many
SELECT sqlc.embed(datasets), users.first_name AS owner_name, count(data_points.id)::bigint AS point_count
FROM datasets LEFT JOIN users ON users.id = datasets.user_id
LEFT JOIN data_points ON data_points.dataset_id = datasets.id
GROUP BY datasets.id, users.id ORDER BY datasets.created_at DESC, datasets.id DESC LIMIT 100;

-- name: AdminListMusicTracks :many
SELECT sqlc.embed(music_tracks), users.first_name AS owner_name
FROM music_tracks LEFT JOIN users ON users.id = music_tracks.user_id
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
