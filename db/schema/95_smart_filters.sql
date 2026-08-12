CREATE TABLE IF NOT EXISTS smart_filters (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    rules_json TEXT NOT NULL,
    is_pinned_sidebar BOOLEAN NOT NULL DEFAULT 0,
    is_pinned_home BOOLEAN NOT NULL DEFAULT 0,
    home_position INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_smart_filters_user ON smart_filters(user_id);
