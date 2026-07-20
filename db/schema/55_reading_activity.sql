CREATE TABLE IF NOT EXISTS reading_progress (
    user_id INTEGER NOT NULL,
    book_id TEXT NOT NULL,
    file_id TEXT,
    chapter_ref TEXT NOT NULL,
    chapter_title TEXT NOT NULL DEFAULT '',
    chapter_index INTEGER NOT NULL DEFAULT 0,
    progress_percent REAL DEFAULT 0,
    opened_count INTEGER NOT NULL DEFAULT 0,
    qualified_read_count INTEGER NOT NULL DEFAULT 0,
    last_opened_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_counted_at DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, book_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS book_read_stats (
    book_id TEXT PRIMARY KEY,
    total_open_count INTEGER NOT NULL DEFAULT 0,
    qualified_read_count INTEGER NOT NULL DEFAULT 0,
    last_opened_at DATETIME,
    last_counted_at DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_reading_progress_user_time ON reading_progress(user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_book_read_stats_count ON book_read_stats(qualified_read_count DESC, updated_at DESC);

CREATE TABLE IF NOT EXISTS book_download_stats (
    book_id TEXT PRIMARY KEY,
    total_download_count INTEGER NOT NULL DEFAULT 0,
    last_downloaded_at DATETIME,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_book_download_stats_count ON book_download_stats(total_download_count DESC, updated_at DESC);
