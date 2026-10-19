-- name: GetAvailableMusicTracks :many
SELECT * FROM music_tracks
WHERE is_active = TRUE AND (user_id IS NULL OR user_id = $1)
ORDER BY created_at DESC;

-- name: RenameSystemMusicTrack :execrows
UPDATE music_tracks SET title = $2
WHERE id = $1 AND user_id IS NULL;

-- name: GetMusicTrackByID :one
SELECT * FROM music_tracks WHERE id = $1;

-- name: CreateMusicTrack :one
INSERT INTO music_tracks (user_id, title, file_path, duration_sec)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetMusicForUser :one
SELECT * FROM music_tracks
WHERE id = $1 AND is_active = TRUE AND (user_id IS NULL OR user_id = sqlc.arg(viewer_id)::bigint);

-- name: ArchiveOwnedMusic :execrows
UPDATE music_tracks SET is_active = FALSE WHERE id = $1 AND user_id = $2;
