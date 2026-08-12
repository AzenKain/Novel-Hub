-- name: CreateHighlight :one
INSERT INTO highlights (
    id, user_id, book_id, chapter_id, text_content, start_index, end_index, color, note, cfi_range
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
) RETURNING id, user_id, book_id, chapter_id, text_content, start_index, end_index, color, note, created_at, updated_at, cfi_range;

-- name: GetHighlightsByIDs :many
SELECT id, user_id, book_id, chapter_id, text_content, start_index, end_index, color, note, created_at, updated_at, cfi_range FROM highlights
WHERE id IN (sqlc.slice('ids'));

-- name: DeleteHighlight :exec
DELETE FROM highlights
WHERE id = ? AND user_id = ?;

-- name: UpdateHighlightNote :one
UPDATE highlights
SET note = ?, color = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND user_id = ?
RETURNING id, user_id, book_id, chapter_id, text_content, start_index, end_index, color, note, created_at, updated_at, cfi_range;

-- name: GetHighlightsByChapter :many
SELECT id, user_id, book_id, chapter_id, text_content, start_index, end_index, color, note, created_at, updated_at, cfi_range FROM highlights
WHERE user_id = ? AND chapter_id = ?
ORDER BY start_index ASC;

-- Export surfaces (Readwise / markdown) join book title + author in one pass so
-- the exporter never issues a second query per book. Served by idx_highlights_user_book.
-- name: GetHighlightsByBook :many
SELECT h.id, h.user_id, h.book_id, h.chapter_id, h.text_content, h.start_index, h.end_index, h.color, h.note, h.created_at, h.updated_at, h.cfi_range,
       b.title AS book_title, COALESCE(a.name, '') AS author_name
FROM highlights h
JOIN books b ON b.id = h.book_id
LEFT JOIN authors a ON a.id = b.author_id
WHERE h.user_id = ? AND h.book_id = ?
ORDER BY h.created_at ASC;

-- Same projection by explicit IDs, for the cache-by-IDs fetch path.
-- name: GetHighlightBooksByIDs :many
SELECT h.id, h.user_id, h.book_id, h.chapter_id, h.text_content, h.start_index, h.end_index, h.color, h.note, h.created_at, h.updated_at, h.cfi_range,
       b.title AS book_title, COALESCE(a.name, '') AS author_name
FROM highlights h
JOIN books b ON b.id = h.book_id
LEFT JOIN authors a ON a.id = b.author_id
WHERE h.id IN (sqlc.slice('ids'));
