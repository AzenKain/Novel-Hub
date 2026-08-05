-- name: GetSeriesByName :one
SELECT id, name, created_at FROM series WHERE name = ? LIMIT 1;

-- name: CreateSeries :one
INSERT INTO series (id, name) VALUES (?, ?) RETURNING *;

-- name: GetPublisherByName :one
SELECT id, name, created_at FROM publishers WHERE name = ? LIMIT 1;

-- name: CreatePublisher :one
INSERT INTO publishers (id, name) VALUES (?, ?) RETURNING *;

-- name: GetLanguageByName :one
SELECT id, name, created_at FROM languages WHERE name = ? LIMIT 1;

-- name: CreateLanguage :one
INSERT INTO languages (id, name) VALUES (?, ?) RETURNING *;

-- name: LinkBookSeries :exec
INSERT INTO book_series (book_id, series_id, series_index) VALUES (?, ?, ?)
ON CONFLICT (book_id, series_id) DO UPDATE SET series_index = excluded.series_index;

-- name: ClearBookSeries :exec
DELETE FROM book_series WHERE book_id = ?;

-- name: LinkBookPublisher :exec
INSERT OR IGNORE INTO book_publishers (book_id, publisher_id) VALUES (?, ?);

-- name: ClearBookPublishers :exec
DELETE FROM book_publishers WHERE book_id = ?;

-- name: LinkBookLanguage :exec
INSERT OR IGNORE INTO book_languages (book_id, language_id) VALUES (?, ?);

-- name: ClearBookLanguages :exec
DELETE FROM book_languages WHERE book_id = ?;

-- name: ClearBookTags :exec
DELETE FROM book_tags WHERE book_id = ?;

-- name: ListAuthorsWithCount :many
SELECT a.id, a.name, COUNT(b.id) as book_count
FROM authors a
JOIN books b ON a.id = b.author_id
WHERE b.library_id IN (SELECT value FROM json_each(sqlc.arg('library_ids')))
  AND (sqlc.narg('search') IS NULL OR a.name LIKE '%' || sqlc.narg('search') || '%')
  AND (sqlc.narg('alpha_upper') IS NULL
       OR UPPER(SUBSTR(TRIM(a.name), 1, 1)) = sqlc.narg('alpha_upper')
       OR SUBSTR(TRIM(a.name), 1, 1) = sqlc.narg('alpha_lower'))
  AND (sqlc.narg('alpha_other') IS NULL
       OR (UPPER(SUBSTR(TRIM(a.name), 1, 1)) NOT BETWEEN 'A' AND 'Z'
           AND SUBSTR(TRIM(a.name), 1, 1) <> sqlc.narg('dstroke_upper')
           AND SUBSTR(TRIM(a.name), 1, 1) <> sqlc.narg('dstroke_lower')))
  AND (sqlc.narg('cursor_name') IS NULL OR a.name > sqlc.narg('cursor_name') OR (a.name = sqlc.narg('cursor_name') AND a.id > sqlc.narg('cursor_id')))
GROUP BY a.id, a.name
ORDER BY a.name ASC, a.id ASC
LIMIT sqlc.arg('limit');

-- name: ListSeriesWithCount :many
SELECT s.id, s.name, COUNT(bs.book_id) as book_count, (
    SELECT b.cover_url
    FROM book_series bs2
    JOIN books b ON b.id = bs2.book_id
    WHERE bs2.series_id = s.id AND b.library_id = sb.library_id
      AND b.cover_url IS NOT NULL AND b.cover_url != ''
    ORDER BY
        CASE
            WHEN bs2.series_index GLOB '[0-9]*' THEN CAST(bs2.series_index AS REAL)
            ELSE 999999
        END ASC,
        b.created_at ASC
    LIMIT 1
) as cover_url
FROM series s
JOIN book_series bs ON s.id = bs.series_id
JOIN books sb ON sb.id = bs.book_id
WHERE sb.library_id IN (SELECT value FROM json_each(sqlc.arg('library_ids')))
  AND (sqlc.narg('search') IS NULL OR s.name LIKE '%' || sqlc.narg('search') || '%')
  AND (sqlc.narg('alpha_upper') IS NULL
       OR UPPER(SUBSTR(TRIM(s.name), 1, 1)) = sqlc.narg('alpha_upper')
       OR SUBSTR(TRIM(s.name), 1, 1) = sqlc.narg('alpha_lower'))
  AND (sqlc.narg('alpha_other') IS NULL
       OR (UPPER(SUBSTR(TRIM(s.name), 1, 1)) NOT BETWEEN 'A' AND 'Z'
           AND SUBSTR(TRIM(s.name), 1, 1) <> sqlc.narg('dstroke_upper')
           AND SUBSTR(TRIM(s.name), 1, 1) <> sqlc.narg('dstroke_lower')))
  AND (sqlc.narg('cursor_name') IS NULL OR s.name > sqlc.narg('cursor_name') OR (s.name = sqlc.narg('cursor_name') AND s.id > sqlc.narg('cursor_id')))
GROUP BY s.id, s.name
ORDER BY s.name ASC, s.id ASC
LIMIT sqlc.arg('limit');

-- name: ListPublishersWithCount :many
SELECT p.id, p.name, COUNT(bp.book_id) as book_count
FROM publishers p
JOIN book_publishers bp ON p.id = bp.publisher_id
JOIN books b ON b.id = bp.book_id
WHERE b.library_id IN (SELECT value FROM json_each(sqlc.arg('library_ids')))
  AND (sqlc.narg('search') IS NULL OR p.name LIKE '%' || sqlc.narg('search') || '%')
  AND (sqlc.narg('alpha_upper') IS NULL
       OR UPPER(SUBSTR(TRIM(p.name), 1, 1)) = sqlc.narg('alpha_upper')
       OR SUBSTR(TRIM(p.name), 1, 1) = sqlc.narg('alpha_lower'))
  AND (sqlc.narg('alpha_other') IS NULL
       OR (UPPER(SUBSTR(TRIM(p.name), 1, 1)) NOT BETWEEN 'A' AND 'Z'
           AND SUBSTR(TRIM(p.name), 1, 1) <> sqlc.narg('dstroke_upper')
           AND SUBSTR(TRIM(p.name), 1, 1) <> sqlc.narg('dstroke_lower')))
  AND (sqlc.narg('cursor_name') IS NULL OR p.name > sqlc.narg('cursor_name') OR (p.name = sqlc.narg('cursor_name') AND p.id > sqlc.narg('cursor_id')))
GROUP BY p.id, p.name
ORDER BY p.name ASC, p.id ASC
LIMIT sqlc.arg('limit');

-- name: ListLanguagesWithCount :many
SELECT l.id, l.name, COUNT(bl.book_id) as book_count
FROM languages l
JOIN book_languages bl ON l.id = bl.language_id
JOIN books b ON b.id = bl.book_id
WHERE b.library_id IN (SELECT value FROM json_each(sqlc.arg('library_ids')))
  AND (sqlc.narg('search') IS NULL OR l.name LIKE '%' || sqlc.narg('search') || '%')
  AND (sqlc.narg('alpha_upper') IS NULL
       OR UPPER(SUBSTR(TRIM(l.name), 1, 1)) = sqlc.narg('alpha_upper')
       OR SUBSTR(TRIM(l.name), 1, 1) = sqlc.narg('alpha_lower'))
  AND (sqlc.narg('alpha_other') IS NULL
       OR (UPPER(SUBSTR(TRIM(l.name), 1, 1)) NOT BETWEEN 'A' AND 'Z'
           AND SUBSTR(TRIM(l.name), 1, 1) <> sqlc.narg('dstroke_upper')
           AND SUBSTR(TRIM(l.name), 1, 1) <> sqlc.narg('dstroke_lower')))
  AND (sqlc.narg('cursor_name') IS NULL OR l.name > sqlc.narg('cursor_name') OR (l.name = sqlc.narg('cursor_name') AND l.id > sqlc.narg('cursor_id')))
GROUP BY l.id, l.name
ORDER BY l.name ASC, l.id ASC
LIMIT sqlc.arg('limit');

-- name: ListTagsWithCount :many
SELECT t.id, t.name, COUNT(bt.book_id) as book_count
FROM tags t
JOIN book_tags bt ON t.id = bt.tag_id
JOIN books b ON b.id = bt.book_id
WHERE b.library_id IN (SELECT value FROM json_each(sqlc.arg('library_ids')))
  AND (sqlc.narg('search') IS NULL OR t.name LIKE '%' || sqlc.narg('search') || '%')
  AND (sqlc.narg('alpha_upper') IS NULL
       OR UPPER(SUBSTR(TRIM(t.name), 1, 1)) = sqlc.narg('alpha_upper')
       OR SUBSTR(TRIM(t.name), 1, 1) = sqlc.narg('alpha_lower'))
  AND (sqlc.narg('alpha_other') IS NULL
       OR (UPPER(SUBSTR(TRIM(t.name), 1, 1)) NOT BETWEEN 'A' AND 'Z'
           AND SUBSTR(TRIM(t.name), 1, 1) <> sqlc.narg('dstroke_upper')
           AND SUBSTR(TRIM(t.name), 1, 1) <> sqlc.narg('dstroke_lower')))
  AND (sqlc.narg('cursor_name') IS NULL OR t.name > sqlc.narg('cursor_name') OR (t.name = sqlc.narg('cursor_name') AND t.id > sqlc.narg('cursor_id')))
GROUP BY t.id, t.name
ORDER BY t.name ASC, t.id ASC
LIMIT sqlc.arg('limit');

-- name: ListFormatsWithCount :many
SELECT LOWER(bf.format) as id, UPPER(bf.format) as name, COUNT(DISTINCT bf.book_id) as book_count
FROM book_files bf
JOIN books b ON b.id = bf.book_id
WHERE b.library_id IN (SELECT value FROM json_each(sqlc.arg('library_ids')))
  AND (sqlc.narg('search') IS NULL OR bf.format LIKE '%' || sqlc.narg('search') || '%')
  AND (sqlc.narg('alpha_upper') IS NULL
       OR UPPER(SUBSTR(TRIM(bf.format), 1, 1)) = sqlc.narg('alpha_upper')
       OR SUBSTR(TRIM(bf.format), 1, 1) = sqlc.narg('alpha_lower'))
  AND (sqlc.narg('alpha_other') IS NULL
       OR (UPPER(SUBSTR(TRIM(bf.format), 1, 1)) NOT BETWEEN 'A' AND 'Z'
           AND SUBSTR(TRIM(bf.format), 1, 1) <> sqlc.narg('dstroke_upper')
           AND SUBSTR(TRIM(bf.format), 1, 1) <> sqlc.narg('dstroke_lower')))
  AND (sqlc.narg('cursor_name') IS NULL OR UPPER(bf.format) > sqlc.narg('cursor_name') OR (UPPER(bf.format) = sqlc.narg('cursor_name') AND LOWER(bf.format) > sqlc.narg('cursor_id')))
GROUP BY LOWER(bf.format)
ORDER BY LOWER(bf.format) ASC
LIMIT sqlc.arg('limit');
