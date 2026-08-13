-- name: CreateReadList :one
INSERT INTO read_lists (
    id, user_id, name, description
) VALUES (
    ?, ?, ?, ?
)
RETURNING id, user_id, name, description, created_at, updated_at;

-- name: GetUserReadListIDs :many
SELECT id FROM read_lists
WHERE user_id = sqlc.arg('user_id') AND
    (created_at <= COALESCE(CAST(sqlc.narg('cursor_created_at') AS TEXT), '9999-12-31 23:59:59')
     AND (sqlc.narg('cursor_created_at') IS NULL OR created_at < CAST(sqlc.narg('cursor_created_at') AS TEXT) OR id < sqlc.narg('cursor_id')))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: GetReadListsByIDs :many
SELECT id, user_id, name, description, created_at, updated_at
FROM read_lists WHERE id IN (sqlc.slice('ids'));

-- name: ReadListOwnedByUser :one
SELECT EXISTS(
    SELECT 1 FROM read_lists
    WHERE id = ? AND user_id = ?
);

-- name: UpdateReadList :one
UPDATE read_lists
SET name = ?, description = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND user_id = ?
RETURNING id, user_id, name, description, created_at, updated_at;

-- name: DeleteReadList :exec
DELETE FROM read_lists
WHERE id = ? AND user_id = ?;

-- name: GetReadListBookIDs :many
SELECT book_id FROM read_list_books
WHERE read_list_id = ?
ORDER BY position ASC;

-- name: CountBooksInReadLists :many
SELECT read_list_id, COUNT(*) AS book_count
FROM read_list_books
WHERE read_list_id IN (SELECT value FROM json_each(sqlc.arg('read_list_ids')))
GROUP BY read_list_id;

-- name: AppendBookToReadList :exec
INSERT INTO read_list_books (
    read_list_id, book_id, position
) VALUES (
    sqlc.arg('read_list_id'),
    sqlc.arg('book_id'),
    COALESCE((SELECT MAX(position) + 1 FROM read_list_books WHERE read_list_id = sqlc.arg('read_list_id')), 0)
) ON CONFLICT DO NOTHING;

-- name: RemoveBookFromReadList :exec
DELETE FROM read_list_books
WHERE read_list_id = ? AND book_id = ?;

-- name: SetReadListBookPosition :exec
UPDATE read_list_books
SET position = sqlc.arg('position')
WHERE read_list_id = sqlc.arg('read_list_id') AND book_id = sqlc.arg('book_id');

-- name: GetNextInReadList :one
SELECT rlb.book_id, rlb.position
FROM read_list_books rlb
JOIN books b ON b.id = rlb.book_id
WHERE rlb.read_list_id = sqlc.arg('read_list_id')
  AND b.status != 'archived'
  AND rlb.position > (
      SELECT prev.position FROM read_list_books prev
      WHERE prev.read_list_id = sqlc.arg('read_list_id') AND prev.book_id = sqlc.arg('after_book_id')
  )
ORDER BY rlb.position ASC
LIMIT 1;

-- name: GetFirstInReadList :one
SELECT rlb.book_id, rlb.position
FROM read_list_books rlb
JOIN books b ON b.id = rlb.book_id
WHERE rlb.read_list_id = sqlc.arg('read_list_id')
  AND b.status != 'archived'
ORDER BY rlb.position ASC
LIMIT 1;

-- name: MatchBooksBySeriesNames :many
SELECT LOWER(s.name) AS series_key, bs.series_index, bs.book_id
FROM book_series bs
JOIN series s ON s.id = bs.series_id
JOIN books b ON b.id = bs.book_id
WHERE LOWER(s.name) IN (SELECT value FROM json_each(sqlc.arg('series_keys')))
  AND b.status != 'archived';
