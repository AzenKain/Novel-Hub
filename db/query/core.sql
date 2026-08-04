-- name: CreateLibrary :one
INSERT INTO libraries (
    id, name
) VALUES (
    ?, ?
)
RETURNING id, name, created_at, updated_at;

-- name: GetLibrary :one
SELECT id, name, created_at, updated_at FROM libraries
WHERE id = ? LIMIT 1;

-- name: ListLibraryIDs :many
SELECT id FROM libraries
ORDER BY created_at DESC;

-- name: GetLibrariesByIDs :many
SELECT id, name, created_at, updated_at FROM libraries WHERE id IN (sqlc.slice('ids'));

-- name: UpdateLibrary :one
UPDATE libraries
SET name = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, name, created_at, updated_at;

-- name: DeleteLibrary :exec
DELETE FROM libraries
WHERE id = ?;

-- name: CreateBookFile :one
INSERT INTO book_files (
    id, book_id, path, format, size_bytes, mod_time, hash, state
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING id, book_id, path, format, size_bytes, mod_time, hash, state, created_at, updated_at;

-- name: UpsertBookFile :one
INSERT INTO book_files (
    id, book_id, path, format, size_bytes, mod_time, hash, state
) VALUES (
    ?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8
)
ON CONFLICT(path) DO UPDATE SET
    size_bytes = excluded.size_bytes,
    mod_time = excluded.mod_time,
    hash = excluded.hash,
    state = excluded.state,
    updated_at = CURRENT_TIMESTAMP
RETURNING id, book_id, path, format, size_bytes, mod_time, hash, state, created_at, updated_at;

-- name: GetBookFileByPath :one
SELECT id, book_id, path, format, size_bytes, mod_time, hash, state, created_at, updated_at FROM book_files
WHERE path = ? LIMIT 1;

-- name: GetFilesByBookId :many
SELECT id, book_id, path, format, size_bytes, mod_time, hash, state, created_at, updated_at FROM book_files
WHERE book_id = ?
ORDER BY
    CASE WHEN LOWER(format) = 'epub' THEN 0 ELSE 1 END,
    created_at ASC;

-- name: CreateJob :one
INSERT INTO jobs (
    id, type, status, total, payload_json
) VALUES (
    ?, ?, ?, ?, ?
)
RETURNING id, type, status, progress, total, error_msg, payload_json, created_at, updated_at;

-- name: UpdateJobStatus :one
UPDATE jobs
SET status = ?, error_msg = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, type, status, progress, total, error_msg, payload_json, created_at, updated_at;

-- name: GetJob :one
SELECT id, type, status, progress, total, error_msg, payload_json, created_at, updated_at FROM jobs
WHERE id = ? LIMIT 1;

-- name: ListJobs :many
SELECT id, type, status, progress, total, error_msg, payload_json, created_at, updated_at FROM jobs
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListUnfinishedJobs :many
SELECT id, type, status, progress, total, error_msg, payload_json, created_at, updated_at FROM jobs
WHERE status IN ('pending', 'running')
ORDER BY created_at ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListUnfinishedJobIDs :many
SELECT id FROM jobs
WHERE status IN ('pending', 'running')
ORDER BY created_at ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetJobsByIDs :many
SELECT id, type, status, progress, total, error_msg, payload_json, created_at, updated_at FROM jobs WHERE id IN (sqlc.slice('ids'));

-- name: ListFileIDsByBookId :many
SELECT id FROM book_files
WHERE book_id = ?
ORDER BY
    CASE WHEN LOWER(format) = 'epub' THEN 0 ELSE 1 END,
    created_at ASC;

-- name: GetBookFilesByIDs :many
SELECT id, book_id, path, format, size_bytes, mod_time, hash, state, created_at, updated_at FROM book_files WHERE id IN (sqlc.slice('ids'));
