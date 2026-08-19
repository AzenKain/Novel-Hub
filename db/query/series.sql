-- name: GetBookSeries :many
SELECT s.id, s.name, bs.series_index
FROM book_series bs
JOIN series s ON s.id = bs.series_id
WHERE bs.book_id = ?
ORDER BY s.name ASC;

-- name: GetNextBookInSeries :one
SELECT b.id, b.library_id, b.title, b.cover_url, bs.series_index
FROM book_series bs
JOIN books b ON b.id = bs.book_id
WHERE bs.series_id = sqlc.arg('series_id')
  AND b.id != sqlc.arg('current_book_id')
  AND b.status != 'archived'
  AND (
      CASE WHEN LTRIM(COALESCE(bs.series_index, '')) GLOB '[0-9]*' THEN CAST(LTRIM(bs.series_index) AS REAL) ELSE 999999 END,
      b.created_at,
      b.title,
      b.id
  ) > (
      SELECT CASE WHEN LTRIM(COALESCE(cur.series_index, '')) GLOB '[0-9]*' THEN CAST(LTRIM(cur.series_index) AS REAL) ELSE 999999 END,
             cb.created_at,
             cb.title,
             cb.id
      FROM book_series cur
      JOIN books cb ON cb.id = cur.book_id
      WHERE cur.book_id = sqlc.arg('current_book_id') AND cur.series_id = sqlc.arg('series_id')
  )
ORDER BY
    CASE WHEN LTRIM(COALESCE(bs.series_index, '')) GLOB '[0-9]*' THEN CAST(LTRIM(bs.series_index) AS REAL) ELSE 999999 END ASC,
    b.created_at ASC,
    b.title COLLATE NOCASE ASC,
    b.id ASC
LIMIT 1;
