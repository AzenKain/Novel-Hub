-- name: ListAudiobookChapters :many
SELECT id, book_id, file_id, chapter_index, title, start_sec, end_sec, created_at, updated_at
FROM audiobook_chapters
WHERE book_id = ?
ORDER BY chapter_index ASC;

-- name: GetAudiobookChapter :one
SELECT id, book_id, file_id, chapter_index, title, start_sec, end_sec, created_at, updated_at
FROM audiobook_chapters
WHERE id = ?;
-- name: UpsertAudiobookChapter :one
INSERT INTO audiobook_chapters (id, book_id, file_id, chapter_index, title, start_sec, end_sec)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(book_id, chapter_index) DO UPDATE SET
    file_id = excluded.file_id,
    title = excluded.title,
    start_sec = excluded.start_sec,
    end_sec = excluded.end_sec,
    updated_at = CURRENT_TIMESTAMP
RETURNING id, book_id, file_id, chapter_index, title, start_sec, end_sec, created_at, updated_at;

-- name: DeleteAudiobookChapter :exec
DELETE FROM audiobook_chapters WHERE id = ?;

-- name: DeleteAudiobookChaptersForBook :exec
DELETE FROM audiobook_chapters WHERE book_id = ?;

-- name: ListBooksWithAudioChapters :many
SELECT DISTINCT b.id
FROM books b
WHERE (
    EXISTS (SELECT 1 FROM audiobook_chapters ac WHERE ac.book_id = b.id)
    OR EXISTS (
      SELECT 1 FROM book_files bf
      WHERE bf.book_id = b.id
        AND LOWER(bf.format) IN ('mp3','m4a','m4b','flac','ogg','opus','wav','aac')
    )
  )
  AND (b.updated_at <= COALESCE(CAST(sqlc.narg('cursor_time') AS TEXT), '9999-12-31 23:59:59')
       AND (sqlc.narg('cursor_time') IS NULL OR b.updated_at < CAST(sqlc.narg('cursor_time') AS TEXT) OR b.id < sqlc.narg('cursor_id')))
ORDER BY b.updated_at DESC, b.id DESC
LIMIT sqlc.arg('limit');