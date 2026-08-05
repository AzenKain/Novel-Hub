-- name: GetUserTOTP :one
SELECT user_id, secret, confirmed_at, created_at
FROM user_totp
WHERE user_id = ?;

-- name: UpsertUserTOTP :one
INSERT INTO user_totp (user_id, secret, confirmed_at)
VALUES (?, ?, NULL)
ON CONFLICT(user_id) DO UPDATE SET
    secret = excluded.secret,
    confirmed_at = NULL,
    created_at = CURRENT_TIMESTAMP
RETURNING user_id, secret, confirmed_at, created_at;

-- name: ConfirmUserTOTP :execrows
UPDATE user_totp
SET confirmed_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND confirmed_at IS NULL;

-- name: DeleteUserTOTP :exec
DELETE FROM user_totp WHERE user_id = ?;

-- name: CreateRecoveryCode :exec
INSERT INTO user_totp_recovery_codes (user_id, code_hash)
VALUES (?, ?)
ON CONFLICT(user_id, code_hash) DO NOTHING;

-- name: DeleteRecoveryCodes :exec
DELETE FROM user_totp_recovery_codes WHERE user_id = ?;

-- name: CountUnusedRecoveryCodes :one
SELECT count(*) FROM user_totp_recovery_codes
WHERE user_id = ? AND used_at IS NULL;

-- name: ConsumeRecoveryCode :execrows
UPDATE user_totp_recovery_codes
SET used_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND code_hash = ? AND used_at IS NULL;
