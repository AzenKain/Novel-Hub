-- name: BulkUpdateBookAuthor :exec
UPDATE books
SET author_id = ?, updated_at = CURRENT_TIMESTAMP
WHERE id IN (sqlc.slice('book_ids'));

-- name: BulkUpdateBookLibrary :exec
UPDATE books
SET library_id = ?, updated_at = CURRENT_TIMESTAMP
WHERE id IN (sqlc.slice('book_ids'));

-- name: BulkDeleteBooks :exec
DELETE FROM books
WHERE id IN (sqlc.slice('book_ids'));
