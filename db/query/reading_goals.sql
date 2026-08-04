-- name: GetUserReadingGoal :one
SELECT user_id, target_words_per_day, target_books_per_year, created_at, updated_at
FROM reading_goals
WHERE user_id = ?;

-- name: UpsertUserReadingGoal :one
INSERT INTO reading_goals (
    user_id, target_words_per_day, target_books_per_year, updated_at
) VALUES (
    ?, ?, ?, CURRENT_TIMESTAMP
)
ON CONFLICT (user_id) DO UPDATE SET
    target_words_per_day = excluded.target_words_per_day,
    target_books_per_year = excluded.target_books_per_year,
    updated_at = CURRENT_TIMESTAMP
RETURNING user_id, target_words_per_day, target_books_per_year, created_at, updated_at;
