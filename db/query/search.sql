-- name: InsertFTSChapter :exec
INSERT INTO fts_chapters (book_id, chapter_id, title, content)
VALUES (?, ?, ?, ?);

-- name: DeleteFTSBook :exec
DELETE FROM fts_chapters WHERE book_id = ?;

-- name: SearchFTS :many
SELECT book_id, chapter_id, title
FROM fts_chapters
WHERE content MATCH ?
ORDER BY rank
LIMIT ? OFFSET ?;
