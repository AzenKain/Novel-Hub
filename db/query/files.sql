-- name: GetDuplicateFiles :many
SELECT hash, COUNT(*) as duplicate_count, GROUP_CONCAT(id) as file_ids
FROM book_files
WHERE hash IS NOT NULL AND hash != ''
GROUP BY hash
HAVING COUNT(*) > 1;

-- name: UpdateFileHash :exec
UPDATE book_files
SET hash = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: ListAllFiles :many
SELECT id, path, book_id
FROM book_files;

-- name: DeleteFile :exec
DELETE FROM book_files WHERE id = ?;

-- name: CountFilesForBook :one
SELECT COUNT(*) FROM book_files WHERE book_id = ?;

-- name: GetBookFileById :one
SELECT * FROM book_files WHERE id = ? LIMIT 1;

-- name: GetFilesByBookIDs :many
SELECT * FROM book_files WHERE book_id IN (sqlc.slice('book_ids'));

