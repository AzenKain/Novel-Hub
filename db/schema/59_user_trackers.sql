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

-- The mapping is per user, not per book: which AniList series a book corresponds to is one
-- reader's judgement, and the sync that follows writes to that reader's own AniList account.
-- A single global row let any user with read access repoint another user's sync — the victim's
-- next sync would then push progress to the wrong series using the victim's own OAuth token.
CREATE TABLE IF NOT EXISTS book_tracker_mappings (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    external_series_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, book_id, provider)
);

-- Composite and single column performance indexes for external trackers
CREATE INDEX IF NOT EXISTS idx_user_trackers_user_provider ON user_trackers(user_id, provider);
CREATE INDEX IF NOT EXISTS idx_btm_user_book_provider ON book_tracker_mappings(user_id, book_id, provider);
CREATE INDEX IF NOT EXISTS idx_btm_external_id ON book_tracker_mappings(external_series_id, provider);
