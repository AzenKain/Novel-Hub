-- name: GetBookSeries :many
-- The chip on the book page needs the series id, not just the name: the catalog filter matches
-- on facet_id and silently returns every book when only a name is supplied.
SELECT s.id, s.name, bs.series_index
FROM book_series bs
JOIN series s ON s.id = bs.series_id
WHERE bs.book_id = ?
ORDER BY s.name ASC;

-- name: GetNextBookInSeries :one
-- Ordered the same way ListSeriesWithCount picks a cover: numeric series_index first, then
-- created_at, so a series with unparseable indexes still has a stable order rather than none.
SELECT b.id, b.library_id, b.title, b.cover_url, bs.series_index
FROM book_series bs
JOIN books b ON b.id = bs.book_id
WHERE bs.series_id = sqlc.arg('series_id')
  AND b.id != sqlc.arg('current_book_id')
  AND b.status != 'archived'
  AND (
      CASE WHEN bs.series_index GLOB '[0-9]*' THEN CAST(bs.series_index AS REAL) ELSE 999999 END,
      b.created_at
  ) > (
      SELECT CASE WHEN cur.series_index GLOB '[0-9]*' THEN CAST(cur.series_index AS REAL) ELSE 999999 END,
             cb.created_at
      FROM book_series cur
      JOIN books cb ON cb.id = cur.book_id
      WHERE cur.book_id = sqlc.arg('current_book_id') AND cur.series_id = sqlc.arg('series_id')
  )
ORDER BY
    CASE WHEN bs.series_index GLOB '[0-9]*' THEN CAST(bs.series_index AS REAL) ELSE 999999 END ASC,
    b.created_at ASC
LIMIT 1;
