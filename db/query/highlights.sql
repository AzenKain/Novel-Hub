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
