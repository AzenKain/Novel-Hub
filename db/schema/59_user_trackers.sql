-- User OAuth tokens and mapped series IDs for external trackers (AniList, MyAnimeList)
CREATE TABLE IF NOT EXISTS user_trackers (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL, -- 'anilist' or 'myanimelist'
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expires_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, provider)
);

CREATE TABLE IF NOT EXISTS book_tracker_mappings (
    id TEXT PRIMARY KEY,
    book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    external_series_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(book_id, provider)
);

-- Composite and single column performance indexes for external trackers
CREATE INDEX IF NOT EXISTS idx_user_trackers_user_provider ON user_trackers(user_id, provider);
CREATE INDEX IF NOT EXISTS idx_btm_book_provider ON book_tracker_mappings(book_id, provider);
CREATE INDEX IF NOT EXISTS idx_btm_external_id ON book_tracker_mappings(external_series_id, provider);
