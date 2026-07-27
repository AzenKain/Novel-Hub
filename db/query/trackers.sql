-- name: GetUserTracker :one
SELECT id, user_id, provider, access_token, refresh_token, expires_at, created_at, updated_at
FROM user_trackers
WHERE user_id = ? AND provider = ?
LIMIT 1;

-- name: GetUserTrackersByIDs :many
SELECT id, user_id, provider, access_token, refresh_token, expires_at, created_at, updated_at
FROM user_trackers
WHERE id IN (sqlc.slice('ids'));

-- name: GetUserTrackerIDsByUser :many
SELECT id
FROM user_trackers
WHERE user_id = ?;

-- name: UpsertUserTracker :one
INSERT INTO user_trackers (id, user_id, provider, access_token, refresh_token, expires_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(user_id, provider) DO UPDATE SET
    access_token = excluded.access_token,
    refresh_token = excluded.refresh_token,
    expires_at = excluded.expires_at,
    updated_at = CURRENT_TIMESTAMP
RETURNING id, user_id, provider, access_token, refresh_token, expires_at, created_at, updated_at;

-- name: DeleteUserTracker :exec
DELETE FROM user_trackers
WHERE user_id = ? AND provider = ?;

-- name: GetBookTrackerMapping :one
SELECT id, book_id, provider, external_series_id, created_at
FROM book_tracker_mappings
WHERE book_id = ? AND provider = ?
LIMIT 1;

-- name: GetBookTrackerMappingsByIDs :many
SELECT id, book_id, provider, external_series_id, created_at
FROM book_tracker_mappings
WHERE id IN (sqlc.slice('ids'));

-- name: GetBookTrackerMappingIDsByBook :many
SELECT id
FROM book_tracker_mappings
WHERE book_id = ?;

-- name: UpsertBookTrackerMapping :one
INSERT INTO book_tracker_mappings (id, book_id, provider, external_series_id)
VALUES (?, ?, ?, ?)
ON CONFLICT(book_id, provider) DO UPDATE SET
    external_series_id = excluded.external_series_id
RETURNING id, book_id, provider, external_series_id, created_at;
