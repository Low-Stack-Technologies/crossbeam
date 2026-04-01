-- name: CreateDevice :one
INSERT INTO devices (user_id, name, type)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListDevicesByUser :many
SELECT * FROM devices WHERE user_id = $1 ORDER BY created_at;

-- name: GetDeviceByID :one
SELECT * FROM devices WHERE id = $1;

-- name: DeleteDevice :exec
DELETE FROM devices WHERE id = $1 AND user_id = $2;

-- name: UpdateDeviceLastSeen :exec
UPDATE devices SET last_seen = now() WHERE id = $1;
