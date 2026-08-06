-- Komga-compatible API for the Mihon/Tachiyomi Komga extension. See internal/routes/komgaRoutes.go
-- for why this shape exists: the extension and Mihon's built-in tracker are two separate HTTP
-- clients with differing DTOs for the same endpoint, so responses emit the union of both.
--
-- Paging is offset-based here, unlike the cursor paging used everywhere else, because the
-- extension sends ?page=N (0-based) and needs totalPages/totalElements to know when to stop.

-- name: ListKomgaSeriesIDs :many
-- Returns ids only; the rows are hydrated per id so a series cached by one listing is reused
-- by the next (cache-by-ids, see internal/repositories/jobScheduleRepository.go).
-- Driven from series with EXISTS rather than joining books and de-duplicating: a JOIN forces a
-- temp b-tree over every book in the library for both GROUP BY and ORDER BY, which cost 190ms
-- per page at 100k books regardless of offset. EXISTS lets ORDER BY ride the name index.
SELECT s.id
FROM series s
WHERE (sqlc.narg('search') IS NULL OR s.name LIKE '%' || sqlc.narg('search') || '%')
  AND EXISTS (
    SELECT 1 FROM book_series bs
    JOIN books b ON b.id = bs.book_id
    WHERE bs.series_id = s.id
      AND b.library_id IN (SELECT value FROM json_each(sqlc.arg('library_ids')))
      AND b.status != 'archived'
  )
ORDER BY s.name ASC, s.id ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountKomgaSeries :one
SELECT COUNT(*)
FROM series s
WHERE (sqlc.narg('search') IS NULL OR s.name LIKE '%' || sqlc.narg('search') || '%')
  AND EXISTS (
    SELECT 1 FROM book_series bs
    JOIN books b ON b.id = bs.book_id
    WHERE bs.series_id = s.id
      AND b.library_id IN (SELECT value FROM json_each(sqlc.arg('library_ids')))
      AND b.status != 'archived'
  );

-- name: GetKomgaSeriesByIDs :many
-- Both id lists arrive as JSON arrays: sqlc.slice expands to N placeholders but leaves other
-- sqlc.arg positions numbered as if it were one, so mixing the two misbinds every later param.
-- library_id is aggregated because Komga's libraryId is a single value per series.
SELECT s.id, s.name, COUNT(bs.book_id) AS book_count,
       CAST(COALESCE(MAX(b.updated_at), CURRENT_TIMESTAMP) AS TEXT) AS last_modified,
       CAST(MIN(b.library_id) AS TEXT) AS library_id
FROM series s
JOIN book_series bs ON bs.series_id = s.id
JOIN books b ON b.id = bs.book_id
WHERE s.id IN (SELECT value FROM json_each(sqlc.arg('series_ids')))
  AND b.library_id IN (SELECT value FROM json_each(sqlc.arg('library_ids')))
  AND b.status != 'archived'
GROUP BY s.id, s.name;

-- name: ListKomgaSeriesBooks :many
-- Ordered the same way GetNextBookInSeries orders: numeric series_index first, then created_at,
-- so a series with unparseable indexes still has a stable order.
SELECT b.id, b.library_id, b.title, b.description, b.cover_url, b.updated_at, b.created_at,
       bs.series_index,
       CAST(CASE WHEN bs.series_index GLOB '[0-9]*' THEN CAST(bs.series_index AS REAL) ELSE 999999 END AS REAL) AS number_sort
FROM book_series bs
JOIN books b ON b.id = bs.book_id
WHERE bs.series_id = sqlc.arg('series_id')
  AND b.library_id IN (SELECT value FROM json_each(sqlc.arg('library_ids')))
  AND b.status != 'archived'
ORDER BY number_sort ASC, b.created_at ASC;

-- name: GetKomgaBookSeries :one
-- The reverse lookup: the extension fetches a book directly and the response must name its series.
SELECT s.id, s.name, bs.series_index,
       CAST(CASE WHEN bs.series_index GLOB '[0-9]*' THEN CAST(bs.series_index AS REAL) ELSE 999999 END AS REAL) AS number_sort
FROM book_series bs
JOIN series s ON s.id = bs.series_id
WHERE bs.book_id = ?
ORDER BY s.name ASC
LIMIT 1;

-- name: ListKomgaSeriesProgress :many
-- Batch form of GetKomgaSeriesProgress: the series listing needs read counts for every row it
-- returns, and one query per series dominated the endpoint (measured in komgaRepository_bench_test.go).
SELECT
    bs.series_id,
    COUNT(*) AS books_count,
    CAST(COALESCE(SUM(CASE WHEN rp.progress_percent >= 99.5 THEN 1 ELSE 0 END), 0) AS INTEGER) AS books_read_count,
    CAST(COALESCE(SUM(CASE WHEN rp.progress_percent > 0 AND rp.progress_percent < 99.5 THEN 1 ELSE 0 END), 0) AS INTEGER) AS books_in_progress_count,
    CAST(COALESCE(MAX(CASE WHEN rp.progress_percent >= 99.5
        THEN CASE WHEN bs.series_index GLOB '[0-9]*' THEN CAST(bs.series_index AS REAL) ELSE 0 END
        ELSE 0 END), 0) AS REAL) AS last_read_number_sort,
    CAST(COALESCE(MAX(CASE WHEN bs.series_index GLOB '[0-9]*' THEN CAST(bs.series_index AS REAL) ELSE 0 END), 0) AS REAL) AS max_number_sort
FROM book_series bs
JOIN books b ON b.id = bs.book_id
LEFT JOIN reading_progress rp ON rp.book_id = b.id AND rp.user_id = sqlc.arg('user_id')
WHERE bs.series_id IN (SELECT value FROM json_each(sqlc.arg('series_ids')))
  AND b.library_id IN (SELECT value FROM json_each(sqlc.arg('library_ids')))
  AND b.status != 'archived'
GROUP BY bs.series_id;

-- name: GetKomgaSeriesProgress :one
-- Drives both the read-progress endpoint and the booksReadCount/booksUnreadCount fields the
-- tracker requires. 99.5 is the same "read" threshold books.sql uses for filter_read.
SELECT
    COUNT(*) AS books_count,
    CAST(COALESCE(SUM(CASE WHEN rp.progress_percent >= 99.5 THEN 1 ELSE 0 END), 0) AS INTEGER) AS books_read_count,
    CAST(COALESCE(SUM(CASE WHEN rp.progress_percent > 0 AND rp.progress_percent < 99.5 THEN 1 ELSE 0 END), 0) AS INTEGER) AS books_in_progress_count,
    CAST(COALESCE(MAX(CASE WHEN rp.progress_percent >= 99.5
        THEN CASE WHEN bs.series_index GLOB '[0-9]*' THEN CAST(bs.series_index AS REAL) ELSE 0 END
        ELSE 0 END), 0) AS REAL) AS last_read_number_sort,
    CAST(COALESCE(MAX(CASE WHEN bs.series_index GLOB '[0-9]*' THEN CAST(bs.series_index AS REAL) ELSE 0 END), 0) AS REAL) AS max_number_sort
FROM book_series bs
JOIN books b ON b.id = bs.book_id
LEFT JOIN reading_progress rp ON rp.book_id = b.id AND rp.user_id = sqlc.arg('user_id')
WHERE bs.series_id = sqlc.arg('series_id')
  AND b.library_id IN (SELECT value FROM json_each(sqlc.arg('library_ids')))
  AND b.status != 'archived';
