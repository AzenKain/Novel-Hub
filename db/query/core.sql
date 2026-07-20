-- name: CreateLibrary :one
INSERT INTO libraries (
    id, name
) VALUES (
    ?, ?
)
RETURNING *;

-- name: GetLibrary :one
SELECT * FROM libraries
WHERE id = ? LIMIT 1;

-- name: ListLibraryIDs :many
SELECT id FROM libraries
ORDER BY created_at DESC;

-- name: GetLibrariesByIDs :many
SELECT * FROM libraries WHERE id IN (sqlc.slice('ids'));

-- name: UpdateLibrary :one
UPDATE libraries
SET name = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteLibrary :exec
DELETE FROM libraries
WHERE id = ?;

-- name: CreateBookFile :one
INSERT INTO book_files (
    id, book_id, path, format, size_bytes, mod_time, hash, state
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

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
RETURNING *;

-- name: GetBookFileByPath :one
SELECT * FROM book_files
WHERE path = ? LIMIT 1;

-- name: GetFilesByBookId :many
SELECT * FROM book_files
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
RETURNING *;

-- name: UpdateJobProgress :one
UPDATE jobs
SET progress = ?, status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: UpdateJobStatus :one
UPDATE jobs
SET status = ?, error_msg = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: GetJob :one
SELECT * FROM jobs
WHERE id = ? LIMIT 1;

-- name: ListJobs :many
SELECT * FROM jobs
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListUnfinishedJobs :many
SELECT * FROM jobs
WHERE status IN ('pending', 'running')
ORDER BY created_at ASC;
