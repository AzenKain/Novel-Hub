-- name: CreateUserDevice :one
INSERT INTO user_devices (
    id, user_id, name, device_type, target_address
) VALUES (
    ?, ?, ?, ?, ?
) RETURNING id, user_id, name, device_type, target_address, created_at, updated_at;

-- name: GetUserDeviceByID :one
SELECT id, user_id, name, device_type, target_address, created_at, updated_at
FROM user_devices
WHERE id = ?;

-- name: GetUserDevicesByIDs :many
SELECT id, user_id, name, device_type, target_address, created_at, updated_at
FROM user_devices
WHERE id IN (sqlc.slice('ids'));

-- name: ListUserDeviceIDs :many
SELECT id FROM user_devices
WHERE user_id = sqlc.arg('user_id')
  AND (
      updated_at <= COALESCE(CAST(sqlc.narg('cursor_updated_at') AS TEXT), '9999-12-31 23:59:59')
      AND (sqlc.narg('cursor_updated_at') IS NULL OR updated_at < CAST(sqlc.narg('cursor_updated_at') AS TEXT) OR id < sqlc.narg('cursor_id'))
  )
ORDER BY updated_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: DeleteUserDevice :exec
DELETE FROM user_devices
WHERE id = ? AND user_id = ?;
