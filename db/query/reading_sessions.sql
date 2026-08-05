-- name: UpsertReadingSession :one
INSERT INTO reading_sessions (
    id, user_id, book_id, session_date, duration_seconds, words_read
) VALUES (
    sqlc.arg('id'),
    sqlc.arg('user_id'),
    sqlc.arg('book_id'),
    date(sqlc.arg('session_date')),
    sqlc.arg('duration_seconds'),
    sqlc.arg('words_read')
)
ON CONFLICT (user_id, book_id, session_date) DO UPDATE
SET 
    duration_seconds = reading_sessions.duration_seconds + excluded.duration_seconds,
    words_read = reading_sessions.words_read + excluded.words_read,
    updated_at = CURRENT_TIMESTAMP
RETURNING id, user_id, book_id, duration_seconds, words_read, session_date, created_at, updated_at;

-- name: GetReadingHeatmap :many
SELECT session_date, SUM(duration_seconds) as total_duration, SUM(words_read) as total_words
FROM reading_sessions
WHERE user_id = ? AND session_date >= date('now', '-365 days')
GROUP BY session_date
ORDER BY session_date ASC;
