-- Kobo device sync. See db/schema/98_kobo_auth.sql for why auth is a path token.

-- name: UpsertKoboAuthToken :one
-- Regenerating replaces the previous token, which is how a user revokes a lost device.
INSERT INTO kobo_auth_tokens (token, user_id)
VALUES (?, ?)
ON CONFLICT (user_id) DO UPDATE SET
    token = excluded.token,
    created_at = CURRENT_TIMESTAMP,
    last_used_at = NULL
RETURNING token, user_id, created_at, last_used_at;

-- name: GetKoboAuthToken :one
SELECT token, user_id, created_at, last_used_at FROM kobo_auth_tokens
WHERE user_id = ? LIMIT 1;

-- name: GetKoboUserByToken :one
-- The hot path: every device request resolves its path token to a user.
SELECT token, user_id, created_at, last_used_at FROM kobo_auth_tokens
WHERE token = ? LIMIT 1;

-- name: TouchKoboAuthToken :exec
-- Records that a device actually used the token, so a user can tell a paired reader from a
-- token they generated and never applied.
UPDATE kobo_auth_tokens SET last_used_at = CURRENT_TIMESTAMP WHERE token = ?;

-- name: DeleteKoboAuthToken :exec
DELETE FROM kobo_auth_tokens WHERE user_id = ?;

-- name: MarkKoboBookSynced :exec
INSERT INTO kobo_synced_books (user_id, book_id) VALUES (?, ?)
ON CONFLICT (user_id, book_id) DO NOTHING;

-- name: ListKoboSyncedBookIDs :many
SELECT book_id FROM kobo_synced_books WHERE user_id = ?;

-- name: CountKoboSyncedBooks :one
SELECT COUNT(*) FROM kobo_synced_books WHERE user_id = ?;

-- name: DeleteKoboSyncedBooks :exec
DELETE FROM kobo_synced_books WHERE user_id = ?;
