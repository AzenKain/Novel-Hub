-- Podcast RSS subscriptions. Downloaded episodes become ordinary library
-- books (one book per episode) so they flow through the normal reader,
-- audiobook merger, and scrobbling paths.
CREATE TABLE IF NOT EXISTS podcasts (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL,
    feed_url TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT,
    cover_url TEXT,
    author TEXT,
    auto_download INTEGER NOT NULL DEFAULT 0,
    last_checked_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (library_id) REFERENCES libraries(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS podcast_episodes (
    id TEXT PRIMARY KEY,
    podcast_id TEXT NOT NULL,
    guid TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    audio_url TEXT NOT NULL,
    duration_sec INTEGER,
    published_at DATETIME,
    downloaded INTEGER NOT NULL DEFAULT 0,
    book_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(podcast_id, guid),
    FOREIGN KEY (podcast_id) REFERENCES podcasts(id) ON DELETE CASCADE,
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_podcast_episodes_podcast ON podcast_episodes(podcast_id, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_podcast_episodes_downloaded ON podcast_episodes(podcast_id, downloaded);