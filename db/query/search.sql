-- name: InsertFTSChapter :exec
INSERT INTO fts_chapters (rowid, book_id, chapter_id, title, content)
VALUES (
    (SELECT rowid FROM chapters WHERE id = sqlc.arg('chapter_id')),
    sqlc.arg('book_id'),
    sqlc.arg('chapter_id'),
    sqlc.arg('title'),
    sqlc.arg('content')
);

-- name: DeleteFTSBook :exec
DELETE FROM fts_chapters WHERE rowid IN (SELECT c.rowid FROM chapters c WHERE c.book_id = ?);

-- name: SearchFTS :many
SELECT book_id, chapter_id, title
FROM fts_chapters
WHERE content MATCH ?
ORDER BY rank
LIMIT ? OFFSET ?;

-- name: SearchFTSInBook :many
SELECT
    fts_chapters.chapter_id,
    fts_chapters.title AS chapter_title,
    chapters.chapter_index,
    snippet(fts_chapters, 3, '[NH_MARK_START]', '[NH_MARK_END]', ' ... ', 12) AS snippet
FROM fts_chapters
JOIN chapters
    ON chapters.id = fts_chapters.chapter_id
    AND chapters.book_id = fts_chapters.book_id
WHERE content MATCH ?
    AND fts_chapters.book_id = ?
ORDER BY rank, chapters.chapter_index
LIMIT ? OFFSET ?;
