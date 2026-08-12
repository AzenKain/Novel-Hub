CREATE TABLE IF NOT EXISTS kobo_auth_tokens (
    token TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_kobo_auth_tokens_user ON kobo_auth_tokens(user_id);

CREATE TABLE IF NOT EXISTS kobo_synced_books (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, book_id)
);

CREATE INDEX IF NOT EXISTS idx_kobo_synced_books_book ON kobo_synced_books(book_id);
