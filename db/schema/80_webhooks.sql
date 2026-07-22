CREATE TABLE IF NOT EXISTS webhooks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    template_type TEXT NOT NULL DEFAULT 'generic', -- 'generic', 'discord', 'telegram', 'slack'
    secret TEXT,
    custom_headers TEXT, -- JSON object string, e.g. {"Authorization": "Bearer token"}
    events TEXT NOT NULL, -- comma separated event list, e.g. "book.created,reading.completed"
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_webhooks_active ON webhooks(is_active);
