-- name: ListAudiobookChapters :many
SELECT id, book_id, file_id, chapter_index, title, start_sec, end_sec, created_at, updated_at
FROM audiobook_chapters
WHERE book_id = ?
ORDER BY chapter_index ASC;

-- name: GetAudiobookChapter :one
SELECT id, book_id, file_id, chapter_index, title, start_sec, end_sec, created_at, updated_at
FROM audiobook_chapters
WHERE id = ?;

-- Upsert keyed on (book_id, chapter_index): editing a chapter keeps its id,
-- re-inserting with the same index overwrites in place.
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