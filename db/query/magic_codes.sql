-- name: CreateMagicCode :exec
INSERT INTO magic_codes (id, code, poll_token, device_info, status, expires_at, created_at)
VALUES (?, ?, ?, ?, 'pending', ?, CURRENT_TIMESTAMP);

-- name: GetMagicCodeByCode :one
SELECT id, code, poll_token, device_info, user_id, jwt_token, status, expires_at, created_at
FROM magic_codes
WHERE code = ? AND expires_at > CURRENT_TIMESTAMP;

-- name: GetMagicCodeByPollToken :one
SELECT id, code, poll_token, device_info, user_id, jwt_token, status, expires_at, created_at
FROM magic_codes
WHERE poll_token = ? AND expires_at > CURRENT_TIMESTAMP;

-- name: ActivateMagicCode :exec
UPDATE magic_codes
SET user_id = ?, jwt_token = ?, status = 'active'
WHERE code = ? AND status = 'pending' AND expires_at > CURRENT_TIMESTAMP;

-- name: MarkMagicCodeUsed :exec
UPDATE magic_codes
SET status = 'used'
WHERE poll_token = ?;

-- name: DeleteExpiredMagicCodes :exec
DELETE FROM magic_codes 
WHERE expires_at < datetime('now', '-1 hour');
