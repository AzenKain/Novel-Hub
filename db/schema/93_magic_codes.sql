CREATE TABLE IF NOT EXISTS magic_codes (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    poll_token TEXT NOT NULL UNIQUE,
    device_info TEXT NOT NULL DEFAULT '',
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    jwt_token TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending', -- pending, active, used, expired
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_magic_codes_code ON magic_codes(code);
CREATE INDEX IF NOT EXISTS idx_magic_codes_poll ON magic_codes(poll_token);
