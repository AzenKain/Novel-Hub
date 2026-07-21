-- name: GetDuplicateFiles :many
SELECT hash, COUNT(*) as duplicate_count, GROUP_CONCAT(id) as file_ids
FROM book_files
WHERE hash IS NOT NULL AND hash != ''
GROUP BY hash
HAVING COUNT(*) > 1
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: UpdateFileHash :exec
UPDATE book_files
SET hash = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: ListAllFiles :many
SELECT id, path, book_id
FROM book_files
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: DeleteFile :exec
DELETE FROM book_files WHERE id = ?;

-- name: CountFilesForBook :one
SELECT COUNT(*) FROM book_files WHERE book_id = ?;

-- name: GetBookFileById :one
SELECT id, book_id, path, format, size_bytes, mod_time, hash, state, created_at, updated_at FROM book_files WHERE id = ? LIMIT 1;

-- name: GetFilesByBookIDs :many
SELECT id, book_id, path, format, size_bytes, mod_time, hash, state, created_at, updated_at FROM book_files WHERE book_id IN (sqlc.slice('book_ids'));

