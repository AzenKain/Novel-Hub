-- name: GetDuplicateFiles :many
SELECT bf.hash, COUNT(*) as duplicate_count, GROUP_CONCAT(bf.id) as file_ids
FROM book_files bf
JOIN books b ON bf.book_id = b.id
WHERE bf.hash IS NOT NULL AND bf.hash != ''
GROUP BY bf.hash
HAVING COUNT(*) > 1
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetDuplicateFileDetails :many
SELECT 
    bf.id as file_id,
    bf.book_id,
    bf.format,
    bf.size_bytes,
    bf.path,
    bf.hash,
    bf.created_at as file_created_at,
    b.title as book_title,
    b.cover_url as book_cover_url,
    b.library_id
FROM book_files bf
JOIN books b ON bf.book_id = b.id
WHERE bf.hash IS NOT NULL AND bf.hash != '' AND bf.hash IN (
    SELECT bf2.hash 
    FROM book_files bf2 
    JOIN books b2 ON bf2.book_id = b2.id 
    WHERE bf2.hash IS NOT NULL AND bf2.hash != '' 
    GROUP BY bf2.hash 
    HAVING COUNT(*) > 1
)
ORDER BY bf.hash, bf.created_at ASC;

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
SELECT id, book_id, path, format, size_bytes, mod_time, hash, state, created_at, updated_at FROM book_files
WHERE book_id IN (sqlc.slice('book_ids'))
ORDER BY
    CASE WHEN LOWER(format) = 'epub' THEN 0 ELSE 1 END,
    created_at ASC;
