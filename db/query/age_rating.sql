-- name: ListContentWarningIDs :many
SELECT id FROM content_warnings ORDER BY name ASC;

-- name: GetContentWarningsByIDs :many
SELECT id, name, description, created_at
FROM content_warnings
WHERE id IN (sqlc.slice('ids'));

-- name: GetContentWarnings :many
SELECT id, name, description, created_at
FROM content_warnings
ORDER BY name ASC;

-- name: GetBookContentWarnings :many
SELECT cw.id, cw.name, cw.description
FROM content_warnings cw
JOIN book_content_warnings bcw ON bcw.warning_id = cw.id
WHERE bcw.book_id = ?;

-- name: AddBookContentWarning :exec
INSERT OR IGNORE INTO book_content_warnings (book_id, warning_id)
VALUES (?, ?);

-- name: ClearBookContentWarnings :exec
DELETE FROM book_content_warnings
WHERE book_id = ?;

-- name: UpdateBookAgeRating :exec
UPDATE books
SET age_rating = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateUserKidsModePin :exec
UPDATE users
SET kids_mode_pin_hash = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateUserKidsModeStatus :exec
UPDATE users
SET is_kids_mode = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: GetUserKidsModeInfo :one
SELECT id, is_kids_mode, kids_mode_pin_hash, max_allowed_age_rating
FROM users
WHERE id = ?;
