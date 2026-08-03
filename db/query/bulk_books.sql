-- name: BulkUpdateBookLibrary :exec
UPDATE books
SET library_id = ?, updated_at = CURRENT_TIMESTAMP
WHERE id IN (sqlc.slice('book_ids'));

-- name: BulkDeleteBooks :exec
DELETE FROM books
WHERE id IN (sqlc.slice('book_ids'));

-- name: BulkDeleteBookFiles :exec
DELETE FROM book_files
WHERE book_id IN (sqlc.slice('book_ids'));

-- name: BulkDeleteBookChapters :exec
DELETE FROM chapters
WHERE book_id IN (sqlc.slice('book_ids'));

-- name: BulkDeleteBookTags :exec
DELETE FROM book_tags
WHERE book_id IN (sqlc.slice('book_ids'));
