-- name: CreateBook :one
INSERT INTO books (
    id, library_id, title, author_id, description, cover_url, status, metadata_json,
    google_books_id, anilist_id, openlibrary_id
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING id, library_id, title, author_id, description, cover_url, status, age_rating, metadata_json, download_count, average_rating, rating_count, read_count, open_count, created_at, updated_at, google_books_id, anilist_id, openlibrary_id;

-- name: GetBook :one
SELECT id, library_id, title, author_id, description, cover_url, status, age_rating, metadata_json, download_count, average_rating, rating_count, read_count, open_count, created_at, updated_at, google_books_id, anilist_id, openlibrary_id FROM books
WHERE id = ? LIMIT 1;

-- name: ListBookIDs :many
SELECT id FROM books
WHERE
    (created_at <= COALESCE(CAST(sqlc.narg('cursor_created_at') AS TEXT), '9999-12-31 23:59:59')
     AND (sqlc.narg('cursor_created_at') IS NULL OR created_at < CAST(sqlc.narg('cursor_created_at') AS TEXT) OR id < sqlc.narg('cursor_id')))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: UpdateBook :one
UPDATE books
SET title = ?, author_id = ?, description = ?, cover_url = ?, status = ?, metadata_json = ?, google_books_id = ?, anilist_id = ?, openlibrary_id = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, library_id, title, author_id, description, cover_url, status, age_rating, metadata_json, download_count, average_rating, rating_count, read_count, open_count, created_at, updated_at, google_books_id, anilist_id, openlibrary_id;

-- name: DeleteBook :exec
DELETE FROM books
WHERE id = ?;

-- name: CreateChapter :one
INSERT INTO chapters (
    id, book_id, title, content_path, chapter_index
) VALUES (
    ?, ?, ?, ?, ?
)
RETURNING id, book_id, title, content_path, chapter_index, created_at, updated_at;

-- name: GetChapter :one
SELECT id, book_id, title, content_path, chapter_index, created_at, updated_at FROM chapters
WHERE id = ? LIMIT 1;

-- name: ListChapterIDsByBook :many
SELECT id FROM chapters
WHERE book_id = ?
ORDER BY chapter_index ASC;

-- name: GetChaptersByIDs :many
SELECT id, book_id, title, content_path, chapter_index, created_at, updated_at FROM chapters WHERE id IN (sqlc.slice('ids'));

-- name: DeleteChapter :exec
DELETE FROM chapters
WHERE id = ?;

-- name: DeleteChaptersByBook :exec
DELETE FROM chapters
WHERE book_id = ?;

-- name: CreateAuthor :one
INSERT INTO authors (
    id, name, bio
) VALUES (
    ?, ?, ?
)
RETURNING id, name, bio, created_at, updated_at;

-- name: GetAuthorByName :one
SELECT id, name, bio, created_at, updated_at FROM authors
WHERE name = ? LIMIT 1;

-- name: GetAuthorById :one
SELECT id, name, bio, created_at, updated_at FROM authors
WHERE id = ? LIMIT 1;

-- name: GetAuthorsByIDs :many
SELECT id, name, bio, created_at, updated_at FROM authors WHERE id IN (sqlc.slice('ids'));

-- name: CreateTag :one
INSERT INTO tags (
    id, name
) VALUES (
    ?, ?
)
RETURNING id, name, created_at;

-- name: GetTagByName :one
SELECT id, name, created_at FROM tags
WHERE name = ? LIMIT 1;

-- name: AddBookTag :exec
INSERT INTO book_tags (
    book_id, tag_id
) VALUES (
    ?, ?
) ON CONFLICT DO NOTHING;

-- name: SearchBookIDs :many
SELECT b.id FROM books b
WHERE
    (b.created_at <= COALESCE(CAST(sqlc.narg('cursor_created_at') AS TEXT), '9999-12-31 23:59:59')
     AND (sqlc.narg('cursor_created_at') IS NULL OR b.created_at < CAST(sqlc.narg('cursor_created_at') AS TEXT) OR b.id < sqlc.narg('cursor_id'))) AND
    (sqlc.narg('library_id') IS NULL OR b.library_id = sqlc.narg('library_id')) AND
    (
        sqlc.narg('search') IS NULL OR
        b.id IN (SELECT book_id FROM fts_metadata WHERE fts_metadata MATCH sqlc.narg('search') ORDER BY rank)
    ) AND
    (sqlc.narg('filter_missing_metadata') IS NULL OR b.metadata_json IS NULL OR b.metadata_json = '' OR b.metadata_json = '{}') AND
    (sqlc.narg('filter_no_cover') IS NULL OR b.cover_url IS NULL OR b.cover_url = '') AND
    (sqlc.narg('filter_has_files') IS NULL OR EXISTS (SELECT 1 FROM book_files bf WHERE bf.book_id = b.id)) AND
    (sqlc.narg('filter_has_author') IS NULL OR b.author_id IS NOT NULL) AND
    (sqlc.narg('filter_has_series') IS NULL OR EXISTS (SELECT 1 FROM book_series bs WHERE bs.book_id = b.id)) AND
    (sqlc.narg('filter_has_tags') IS NULL OR EXISTS (SELECT 1 FROM book_tags bt WHERE bt.book_id = b.id)) AND
    (sqlc.narg('filter_has_publishers') IS NULL OR EXISTS (SELECT 1 FROM book_publishers bp WHERE bp.book_id = b.id)) AND
    (sqlc.narg('filter_has_languages') IS NULL OR EXISTS (SELECT 1 FROM book_languages bl WHERE bl.book_id = b.id)) AND
    (sqlc.narg('filter_has_formats') IS NULL OR EXISTS (SELECT 1 FROM book_files bf WHERE bf.book_id = b.id)) AND
    (sqlc.narg('filter_reading') IS NULL OR EXISTS (SELECT 1 FROM reading_progress rp WHERE rp.book_id = b.id AND rp.progress_percent > 0 AND rp.progress_percent < 99.5)) AND
    (sqlc.narg('filter_read') IS NULL OR EXISTS (SELECT 1 FROM reading_progress rp WHERE rp.book_id = b.id AND rp.progress_percent >= 99.5)) AND
    (sqlc.narg('filter_unread') IS NULL OR NOT EXISTS (SELECT 1 FROM reading_progress rp WHERE rp.book_id = b.id AND rp.progress_percent > 0)) AND
    (sqlc.narg('filter_hot') IS NULL OR b.read_count > 0 OR b.open_count > 0 OR EXISTS (SELECT 1 FROM book_read_stats brs WHERE brs.book_id = b.id AND (brs.total_open_count > 0 OR brs.qualified_read_count > 0))) AND
    (sqlc.narg('filter_top_downloaded') IS NULL OR b.download_count > 0) AND
    (sqlc.narg('filter_top_rated') IS NULL OR b.rating_count > 0) AND
    (sqlc.narg('filter_archived') IS NULL OR b.status = 'archived') AND
    (sqlc.narg('filter_bookmarked') IS NULL OR EXISTS (SELECT 1 FROM bookmarks bm WHERE bm.book_id = b.id AND bm.user_id = sqlc.narg('user_id'))) AND
    (sqlc.narg('author_id') IS NULL OR b.author_id = sqlc.narg('author_id')) AND
    (sqlc.narg('series_id') IS NULL OR EXISTS (SELECT 1 FROM book_series bs WHERE bs.book_id = b.id AND bs.series_id = sqlc.narg('series_id'))) AND
    (sqlc.narg('tag_id') IS NULL OR EXISTS (SELECT 1 FROM book_tags bt WHERE bt.book_id = b.id AND bt.tag_id = sqlc.narg('tag_id'))) AND
    (sqlc.narg('publisher_id') IS NULL OR EXISTS (SELECT 1 FROM book_publishers bp WHERE bp.book_id = b.id AND bp.publisher_id = sqlc.narg('publisher_id'))) AND
    (sqlc.narg('language_id') IS NULL OR EXISTS (SELECT 1 FROM book_languages bl WHERE bl.book_id = b.id AND bl.language_id = sqlc.narg('language_id'))) AND
    (sqlc.narg('file_format') IS NULL OR EXISTS (SELECT 1 FROM book_files bf WHERE bf.book_id = b.id AND LOWER(bf.format) = LOWER(sqlc.narg('file_format')))) AND
    (sqlc.narg('exclude_audiobooks') IS NULL OR NOT EXISTS (SELECT 1 FROM book_files bf WHERE bf.book_id = b.id) OR EXISTS (SELECT 1 FROM book_files bf WHERE bf.book_id = b.id AND LOWER(bf.format) NOT IN ('mp3','m4a','m4b','flac','ogg','opus','wav','aac')))
ORDER BY
    b.created_at DESC, b.id DESC
LIMIT sqlc.arg('limit');

-- name: GetBooksByIDs :many
SELECT id, library_id, title, author_id, description, cover_url, status, age_rating, metadata_json, download_count, average_rating, rating_count, read_count, open_count, created_at, updated_at, google_books_id, anilist_id, openlibrary_id FROM books WHERE id IN (sqlc.slice('ids'));

-- name: GetRandomBookIDs :many
SELECT b.id FROM books b
WHERE (sqlc.narg('library_id') IS NULL OR b.library_id = sqlc.narg('library_id'))
ORDER BY RANDOM()
LIMIT sqlc.arg('limit');

-- name: SearchSmartFilterBookIDs :many
SELECT b.id FROM books b
WHERE
    (b.created_at <= COALESCE(CAST(sqlc.narg('cursor_created_at') AS TEXT), '9999-12-31 23:59:59')
     AND (sqlc.narg('cursor_created_at') IS NULL OR b.created_at < CAST(sqlc.narg('cursor_created_at') AS TEXT) OR b.id < sqlc.narg('cursor_id'))) AND
    (sqlc.narg('library_id') IS NULL OR b.library_id = sqlc.narg('library_id')) AND
    (sqlc.narg('file_format') IS NULL OR EXISTS (SELECT 1 FROM book_files bf WHERE bf.book_id = b.id AND LOWER(bf.format) = LOWER(sqlc.narg('file_format')))) AND
    (sqlc.narg('status_unread') IS NULL OR NOT EXISTS (SELECT 1 FROM reading_progress rp WHERE rp.book_id = b.id AND rp.progress_percent > 0)) AND
    (sqlc.narg('status_read') IS NULL OR EXISTS (SELECT 1 FROM reading_progress rp WHERE rp.book_id = b.id AND rp.progress_percent >= 99.5)) AND
    (sqlc.narg('status_reading') IS NULL OR EXISTS (SELECT 1 FROM reading_progress rp WHERE rp.book_id = b.id AND rp.progress_percent > 0 AND rp.progress_percent < 99.5)) AND
    (sqlc.narg('rating_gte') IS NULL OR b.average_rating >= sqlc.narg('rating_gte')) AND
    (sqlc.narg('author_id') IS NULL OR b.author_id = sqlc.narg('author_id')) AND
    (sqlc.narg('series_id') IS NULL OR EXISTS (SELECT 1 FROM book_series bs WHERE bs.book_id = b.id AND bs.series_id = sqlc.narg('series_id'))) AND
    (sqlc.narg('tag_id') IS NULL OR EXISTS (SELECT 1 FROM book_tags bt WHERE bt.book_id = b.id AND bt.tag_id = sqlc.narg('tag_id')))
ORDER BY
    b.created_at DESC, b.id DESC
LIMIT sqlc.arg('limit');

