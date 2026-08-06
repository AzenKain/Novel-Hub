-- name: GetLibraryStats :one
SELECT 
    (SELECT COUNT(*) FROM books) AS total_books,
    (SELECT COUNT(*) FROM books WHERE status = 'error') AS need_review,
    (SELECT COUNT(DISTINCT author_id) FROM books WHERE author_id IS NOT NULL) AS series_tracked
;

-- name: CreateCollection :one
INSERT INTO collections (
    id, user_id, name
) VALUES (
    ?, ?, ?
)
RETURNING id, user_id, name, created_at, updated_at;

-- name: GetUserCollectionIDs :many
SELECT id FROM collections
WHERE user_id = sqlc.arg('user_id') AND
    (created_at <= COALESCE(CAST(sqlc.narg('cursor_created_at') AS TEXT), '9999-12-31 23:59:59')
     AND (sqlc.narg('cursor_created_at') IS NULL OR created_at < CAST(sqlc.narg('cursor_created_at') AS TEXT) OR id < sqlc.narg('cursor_id')))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit');

-- name: GetCollectionsByIDs :many
SELECT id, user_id, name, created_at, updated_at FROM collections WHERE id IN (sqlc.slice('ids'));

-- name: CollectionOwnedByUser :one
SELECT EXISTS(
    SELECT 1 FROM collections
    WHERE id = ? AND user_id = ?
);

-- name: DeleteCollection :exec
DELETE FROM collections
WHERE id = ? AND user_id = ?;

-- name: UpdateCollection :one
UPDATE collections
SET name = ?
WHERE id = ? AND user_id = ?
RETURNING id, user_id, name, created_at, updated_at;

-- name: AddBookToCollection :exec
INSERT INTO collection_books (
    collection_id, book_id
) VALUES (
    ?, ?
) ON CONFLICT DO NOTHING;

-- name: RemoveBookFromCollection :exec
DELETE FROM collection_books
WHERE collection_id = ? AND book_id = ?;

-- Keyset paged on the same (created_at, id) pair as SearchBookIDs, so the collection filter no
-- longer has to load every member and page in Go. created_at is second-resolution, hence the id
-- tiebreaker: without it a bulk upload's tied rows straddle the page boundary and one is skipped.
-- name: GetBookIDsInCollection :many
SELECT b.id
FROM books b
JOIN collection_books cb ON cb.book_id = b.id
WHERE cb.collection_id = sqlc.arg('collection_id') AND
    (b.created_at <= COALESCE(CAST(sqlc.narg('cursor_created_at') AS TEXT), '9999-12-31 23:59:59')
     AND (sqlc.narg('cursor_created_at') IS NULL OR b.created_at < CAST(sqlc.narg('cursor_created_at') AS TEXT) OR b.id < sqlc.narg('cursor_id')))
ORDER BY b.created_at DESC, b.id DESC
LIMIT sqlc.arg('limit');

-- name: GetReadingProgress :one
SELECT user_id, book_id, file_id, chapter_ref, chapter_title, chapter_index, progress_percent, location_cfi, location_type, opened_count, qualified_read_count, last_opened_at, last_counted_at, updated_at
FROM reading_progress
WHERE user_id = ? AND book_id = ?
LIMIT 1;

-- name: UpsertReadingProgress :one
INSERT INTO reading_progress (
    user_id,
    book_id,
    file_id,
    chapter_ref,
    chapter_title,
    chapter_index,
    progress_percent,
    location_cfi,
    location_type,
    opened_count,
    qualified_read_count,
    last_opened_at,
    last_counted_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?, CURRENT_TIMESTAMP
)
ON CONFLICT(user_id, book_id) DO UPDATE SET
    file_id = excluded.file_id,
    chapter_ref = excluded.chapter_ref,
    chapter_title = excluded.chapter_title,
    chapter_index = excluded.chapter_index,
    progress_percent = excluded.progress_percent,
    location_cfi = excluded.location_cfi,
    location_type = excluded.location_type,
    opened_count = reading_progress.opened_count + excluded.opened_count,
    qualified_read_count = reading_progress.qualified_read_count + excluded.qualified_read_count,
    last_opened_at = CURRENT_TIMESTAMP,
    last_counted_at = COALESCE(excluded.last_counted_at, reading_progress.last_counted_at),
    updated_at = CURRENT_TIMESTAMP
RETURNING user_id, book_id, file_id, chapter_ref, chapter_title, chapter_index, progress_percent, location_cfi, location_type, opened_count, qualified_read_count, last_opened_at, last_counted_at, updated_at;

-- name: GetBookReadStats :one
SELECT book_id, total_open_count, qualified_read_count, last_opened_at, last_counted_at, updated_at FROM book_read_stats
WHERE book_id = ?
LIMIT 1;

-- name: UpsertBookReadStats :exec
INSERT INTO book_read_stats (
    book_id,
    total_open_count,
    qualified_read_count,
    last_opened_at,
    last_counted_at,
    updated_at
) VALUES (
    ?, ?, ?, CURRENT_TIMESTAMP, ?, CURRENT_TIMESTAMP
)
ON CONFLICT(book_id) DO UPDATE SET
    total_open_count = book_read_stats.total_open_count + excluded.total_open_count,
    qualified_read_count = book_read_stats.qualified_read_count + excluded.qualified_read_count,
    last_opened_at = CURRENT_TIMESTAMP,
    last_counted_at = COALESCE(excluded.last_counted_at, book_read_stats.last_counted_at),
    updated_at = CURRENT_TIMESTAMP;

-- name: GetRecentReadingHistory :many
SELECT 
    rp.user_id,
    rp.book_id,
    rp.file_id,
    rp.chapter_ref as chapter_id,
    rp.progress_percent,
    rp.updated_at,
    b.title as book_title,
    b.cover_url as book_cover_url,
    rp.chapter_title,
    rp.chapter_index
FROM reading_progress rp
JOIN books b ON b.id = rp.book_id
WHERE rp.user_id = sqlc.arg('user_id') AND
    (rp.updated_at <= COALESCE(CAST(sqlc.narg('cursor_updated_at') AS TEXT), '9999-12-31 23:59:59')
     AND (sqlc.narg('cursor_updated_at') IS NULL OR rp.updated_at < CAST(sqlc.narg('cursor_updated_at') AS TEXT) OR rp.book_id < sqlc.narg('cursor_id')))
ORDER BY rp.updated_at DESC, rp.book_id DESC
LIMIT sqlc.arg('limit');

-- name: UpsertBookDownloadStats :exec
INSERT INTO book_download_stats (
    book_id,
    total_download_count,
    last_downloaded_at,
    updated_at
) VALUES (
    ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
ON CONFLICT(book_id) DO UPDATE SET
    total_download_count = book_download_stats.total_download_count + excluded.total_download_count,
    last_downloaded_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP;

-- name: GetBookDownloadStats :one
SELECT book_id, total_download_count, last_downloaded_at, updated_at FROM book_download_stats
WHERE book_id = ?
LIMIT 1;

-- name: GetBookmark :one
SELECT user_id, book_id, created_at FROM bookmarks
WHERE user_id = ? AND book_id = ?
LIMIT 1;

-- name: UpsertBookmark :one
INSERT INTO bookmarks (
    user_id, book_id, created_at
) VALUES (
    ?, ?, CURRENT_TIMESTAMP
)
ON CONFLICT(user_id, book_id) DO UPDATE SET
    created_at = bookmarks.created_at
RETURNING user_id, book_id, created_at;

-- name: DeleteBookmark :exec
DELETE FROM bookmarks
WHERE user_id = ? AND book_id = ?;

-- name: RefreshBookBookmarkStats :exec
INSERT INTO book_social_stats (
    book_id,
    bookmark_count,
    rating_count,
    average_rating,
    share_count,
    updated_at
)
VALUES (
    ?,
    (SELECT COUNT(*) FROM bookmarks WHERE bookmarks.book_id = ?),
    COALESCE((SELECT stats.rating_count FROM book_social_stats AS stats WHERE stats.book_id = ?), 0),
    COALESCE((SELECT stats.average_rating FROM book_social_stats AS stats WHERE stats.book_id = ?), 0),
    COALESCE((SELECT stats.share_count FROM book_social_stats AS stats WHERE stats.book_id = ?), 0),
    CURRENT_TIMESTAMP
)
ON CONFLICT(book_id) DO UPDATE SET
    bookmark_count = excluded.bookmark_count,
    updated_at = CURRENT_TIMESTAMP;

-- name: GetBookmarkedBooks :many
SELECT book_id, created_at FROM bookmarks
WHERE user_id = sqlc.arg('user_id') AND
    (created_at <= COALESCE(CAST(sqlc.narg('cursor_created_at') AS TEXT), '9999-12-31 23:59:59')
     AND (sqlc.narg('cursor_created_at') IS NULL OR created_at < CAST(sqlc.narg('cursor_created_at') AS TEXT) OR book_id < sqlc.narg('cursor_id')))
ORDER BY created_at DESC, book_id DESC
LIMIT sqlc.arg('limit');

-- name: UpsertBookReview :one
INSERT INTO book_reviews (
    user_id, book_id, rating, review, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
ON CONFLICT(user_id, book_id) DO UPDATE SET
    rating = excluded.rating,
    review = excluded.review,
    updated_at = CURRENT_TIMESTAMP
RETURNING user_id, book_id, rating, review, created_at, updated_at;

-- name: DeleteBookReview :exec
DELETE FROM book_reviews
WHERE user_id = ? AND book_id = ?;

-- name: RefreshBookRatingStats :exec
INSERT INTO book_social_stats (
    book_id,
    bookmark_count,
    rating_count,
    average_rating,
    share_count,
    updated_at
)
VALUES (
    ?,
    COALESCE((SELECT stats.bookmark_count FROM book_social_stats AS stats WHERE stats.book_id = ?), 0),
    (SELECT COUNT(*) FROM book_reviews WHERE book_reviews.book_id = ?),
    COALESCE((SELECT AVG(book_reviews.rating) FROM book_reviews WHERE book_reviews.book_id = ?), 0),
    COALESCE((SELECT stats.share_count FROM book_social_stats AS stats WHERE stats.book_id = ?), 0),
    CURRENT_TIMESTAMP
)
ON CONFLICT(book_id) DO UPDATE SET
    rating_count = excluded.rating_count,
    average_rating = excluded.average_rating,
    updated_at = CURRENT_TIMESTAMP;

-- name: GetBookReview :one
SELECT user_id, book_id, rating, review, created_at, updated_at FROM book_reviews
WHERE user_id = ? AND book_id = ?
LIMIT 1;

-- name: ListBookReviews :many
SELECT user_id, book_id, rating, review, created_at, updated_at FROM book_reviews
WHERE book_id = sqlc.arg('book_id') AND
    (updated_at <= COALESCE(CAST(sqlc.narg('cursor_updated_at') AS TEXT), '9999-12-31 23:59:59')
     AND (sqlc.narg('cursor_updated_at') IS NULL OR updated_at < CAST(sqlc.narg('cursor_updated_at') AS TEXT) OR user_id < sqlc.narg('cursor_id')))
ORDER BY updated_at DESC, user_id DESC
LIMIT sqlc.arg('limit');

-- name: GetBookRatingSummary :one
SELECT
    book_id,
    rating_count,
    average_rating
FROM book_social_stats
WHERE book_id = ?
LIMIT 1;

-- name: GetBookSocialStats :one
SELECT book_id, bookmark_count, rating_count, average_rating, share_count, updated_at FROM book_social_stats
WHERE book_id = ?
LIMIT 1;

-- name: CreateBookShareEvent :one
INSERT INTO book_share_events (
    book_id,
    actor_key,
    window_bucket,
    created_at
) VALUES (
    ?, ?, ?, CURRENT_TIMESTAMP
)
ON CONFLICT(book_id, actor_key, window_bucket) DO NOTHING
RETURNING book_id, actor_key, window_bucket, created_at;

-- name: UpsertBookShareStats :exec
INSERT INTO book_social_stats (
    book_id,
    bookmark_count,
    rating_count,
    average_rating,
    share_count,
    updated_at
)
VALUES (
    ?,
    COALESCE((SELECT stats.bookmark_count FROM book_social_stats AS stats WHERE stats.book_id = ?), 0),
    COALESCE((SELECT stats.rating_count FROM book_social_stats AS stats WHERE stats.book_id = ?), 0),
    COALESCE((SELECT stats.average_rating FROM book_social_stats AS stats WHERE stats.book_id = ?), 0),
    ?,
    CURRENT_TIMESTAMP
)
ON CONFLICT(book_id) DO UPDATE SET
    share_count = book_social_stats.share_count + excluded.share_count,
    updated_at = CURRENT_TIMESTAMP;

-- name: GetBookCollectionIDs :many
SELECT cb.collection_id 
FROM collection_books cb
JOIN collections c ON c.id = cb.collection_id
WHERE c.user_id = ? AND cb.book_id = ?;

-- name: ListAllReviews :many
SELECT br.user_id, br.book_id, br.rating, br.review, br.created_at, br.updated_at,
       u.full_name as user_name, u.email as user_email,
       b.title as book_title
FROM (SELECT user_id, book_id, rating, review, created_at, updated_at
      FROM book_reviews ORDER BY updated_at DESC LIMIT ? OFFSET ?) br
JOIN users u ON u.id = br.user_id
JOIN books b ON b.id = br.book_id
ORDER BY br.updated_at DESC;

-- name: GetRecentReadingHistoryBookIDs :many
SELECT rp.book_id FROM reading_progress rp
WHERE rp.user_id = ? AND
    (rp.updated_at <= COALESCE(CAST(sqlc.narg('cursor_updated_at') AS TEXT), '9999-12-31 23:59:59')
     AND (sqlc.narg('cursor_updated_at') IS NULL OR rp.updated_at < CAST(sqlc.narg('cursor_updated_at') AS TEXT) OR rp.book_id < sqlc.narg('cursor_id')))
ORDER BY rp.updated_at DESC, rp.book_id DESC
LIMIT sqlc.arg('limit');

-- name: ListBookReviewCompositeKeys :many
SELECT CAST(user_id AS TEXT) || ':' || book_id as composite_key FROM book_reviews
WHERE book_id = ? AND
    (updated_at <= COALESCE(CAST(sqlc.narg('cursor_updated_at') AS TEXT), '9999-12-31 23:59:59')
     AND (sqlc.narg('cursor_updated_at') IS NULL OR updated_at < CAST(sqlc.narg('cursor_updated_at') AS TEXT) OR user_id < sqlc.narg('cursor_id')))
ORDER BY updated_at DESC, user_id DESC
LIMIT sqlc.arg('limit');
