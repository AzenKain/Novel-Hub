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
LIMIT ? OFFSET ? /* capped at 50 by repository */;
