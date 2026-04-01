-- name: CreatePush :one
INSERT INTO pushes (user_id, source_device_id, target_device_id, type, title, body, url, file_name, file_type, file_s3_key, file_size)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: ListPushes :many
SELECT * FROM pushes
WHERE user_id = $1 AND created_at < $2
ORDER BY created_at DESC
LIMIT $3;

-- name: GetPushByID :one
SELECT * FROM pushes WHERE id = $1 AND user_id = $2;

-- name: DeletePush :exec
DELETE FROM pushes WHERE id = $1 AND user_id = $2;

-- name: MarkPushDelivered :exec
UPDATE pushes SET delivered = true WHERE id = $1;

-- name: GetPendingPushes :many
SELECT * FROM pushes
WHERE delivered = false
  AND (target_device_id = $1 OR (target_device_id IS NULL AND user_id = $2));
