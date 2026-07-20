-- name: GetSeriesByName :one
SELECT * FROM series WHERE name = ? LIMIT 1;

-- name: CreateSeries :one
INSERT INTO series (id, name) VALUES (?, ?) RETURNING *;

-- name: GetPublisherByName :one
SELECT * FROM publishers WHERE name = ? LIMIT 1;

-- name: CreatePublisher :one
INSERT INTO publishers (id, name) VALUES (?, ?) RETURNING *;

-- name: GetLanguageByName :one
SELECT * FROM languages WHERE name = ? LIMIT 1;

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

-- name: LinkBookTag :exec
INSERT OR IGNORE INTO book_tags (book_id, tag_id) VALUES (?, ?);

-- name: ClearBookTags :exec
DELETE FROM book_tags WHERE book_id = ?;

-- name: ListAuthorsWithCount :many
SELECT a.id, a.name, COUNT(b.id) as book_count
FROM authors a
JOIN books b ON a.id = b.author_id
GROUP BY a.id, a.name
ORDER BY a.name ASC;

-- name: ListSeriesWithCount :many
SELECT s.id, s.name, COUNT(bs.book_id) as book_count, (
    SELECT b.cover_url
    FROM book_series bs2
    JOIN books b ON b.id = bs2.book_id
    WHERE bs2.series_id = s.id AND b.cover_url IS NOT NULL AND b.cover_url != ''
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
GROUP BY s.id, s.name
ORDER BY s.name ASC;

-- name: ListPublishersWithCount :many
SELECT p.id, p.name, COUNT(bp.book_id) as book_count
FROM publishers p
JOIN book_publishers bp ON p.id = bp.publisher_id
GROUP BY p.id, p.name
ORDER BY p.name ASC;

-- name: ListLanguagesWithCount :many
SELECT l.id, l.name, COUNT(bl.book_id) as book_count
FROM languages l
JOIN book_languages bl ON l.id = bl.language_id
GROUP BY l.id, l.name
ORDER BY l.name ASC;

-- name: ListTagsWithCount :many
SELECT t.id, t.name, COUNT(bt.book_id) as book_count
FROM tags t
JOIN book_tags bt ON t.id = bt.tag_id
GROUP BY t.id, t.name
ORDER BY t.name ASC;

-- name: ListFormatsWithCount :many
SELECT LOWER(format) as id, UPPER(format) as name, COUNT(DISTINCT book_id) as book_count
FROM book_files
GROUP BY LOWER(format)
ORDER BY LOWER(format) ASC;
