-- name: GetReadingStatsByBook :one
SELECT book_id, COALESCE(SUM(duration_seconds), 0) AS total_duration, COALESCE(SUM(words_read), 0) AS total_words
FROM reading_sessions
WHERE user_id = ? AND book_id = ?
GROUP BY book_id;

-- name: GetReadingStatsSince :one
SELECT COALESCE(SUM(duration_seconds), 0) AS total_duration, COALESCE(SUM(words_read), 0) AS total_words
FROM reading_sessions
WHERE user_id = ? AND session_date >= ?;

-- name: CountCompletedBooksThisYear :one
SELECT COUNT(DISTINCT book_id) AS completed_count
FROM reading_progress
WHERE user_id = ? AND progress_percent >= 99.5 AND updated_at >= datetime('now', 'start of year');

-- name: GetListeningHistory :many
SELECT strftime('%Y-%m', session_date) AS month, SUM(duration_seconds) AS total_duration
FROM reading_sessions
WHERE user_id = ? AND EXISTS (
    SELECT 1 FROM book_files WHERE book_id = reading_sessions.book_id AND format IN ('mp3', 'm4a', 'm4b', 'flac')
)
GROUP BY month
ORDER BY month ASC;

-- name: GetListeningStats :one
SELECT COALESCE(SUM(duration_seconds), 0) AS total_duration, COALESCE(SUM(words_read), 0) AS total_words
FROM reading_sessions
WHERE user_id = ? AND EXISTS (
    SELECT 1 FROM book_files WHERE book_id = reading_sessions.book_id AND format IN ('mp3', 'm4a', 'm4b', 'flac')
);

-- name: StatsByFormat :many
SELECT format AS name, COUNT(*) AS book_count
FROM book_files
WHERE format IS NOT NULL AND format != ''
GROUP BY format
ORDER BY book_count DESC
LIMIT 100;

-- name: StatsByTag :many
SELECT t.name AS name, COUNT(*) AS book_count
FROM book_tags bt
JOIN tags t ON t.id = bt.tag_id
GROUP BY t.id
ORDER BY book_count DESC
LIMIT 100;

-- name: StatsByAuthor :many
SELECT a.name AS name, COUNT(*) AS book_count
FROM books b
JOIN authors a ON a.id = b.author_id
GROUP BY a.id
ORDER BY book_count DESC
LIMIT 100;

-- name: StatsByPublisher :many
SELECT p.name AS name, COUNT(*) AS book_count
FROM book_publishers bp
JOIN publishers p ON p.id = bp.publisher_id
GROUP BY p.id
ORDER BY book_count DESC
LIMIT 100;

-- name: StatsByLanguage :many
SELECT l.name AS name, COUNT(*) AS book_count
FROM book_languages bl
JOIN languages l ON l.id = bl.language_id
GROUP BY l.id
ORDER BY book_count DESC
LIMIT 100;