CREATE TABLE IF NOT EXISTS soundscapes (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'ambient',
    file_path TEXT NOT NULL,
    icon TEXT NOT NULL DEFAULT 'Music',
    volume REAL NOT NULL DEFAULT 0.5,
    is_system INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS custom_fonts (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    font_family TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'file',
    file_path TEXT NOT NULL DEFAULT '',
    font_url TEXT NOT NULL DEFAULT '',
    is_system INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS custom_themes (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    bg_color TEXT NOT NULL,
    text_color TEXT NOT NULL,
    accent_color TEXT NOT NULL,
    custom_css TEXT NOT NULL DEFAULT '',
    is_system INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_soundscapes_user_id ON soundscapes(user_id);
CREATE INDEX IF NOT EXISTS idx_soundscapes_is_system ON soundscapes(is_system);
CREATE INDEX IF NOT EXISTS idx_custom_fonts_user_id ON custom_fonts(user_id);
CREATE INDEX IF NOT EXISTS idx_custom_fonts_is_system ON custom_fonts(is_system);
CREATE INDEX IF NOT EXISTS idx_custom_themes_user_id ON custom_themes(user_id);
CREATE INDEX IF NOT EXISTS idx_custom_themes_is_system ON custom_themes(is_system);

INSERT INTO permissions (key, description) VALUES
    ('user.font.manage', 'Upload and manage personal custom fonts'),
    ('user.soundscape.manage', 'Upload and manage personal ambient soundscapes'),
    ('user.theme.manage', 'Create and customize personal reader themes and custom CSS'),
    ('admin.soundscape.manage', 'Manage system default ambient soundscapes'),
    ('admin.font.manage', 'Manage system default reader fonts'),
    ('admin.theme.manage', 'Manage system default themes and global styles')
ON CONFLICT(key) DO UPDATE SET
    description = excluded.description;

INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'ADMIN'
ON CONFLICT(role_id, permission_key) DO UPDATE SET
    effect = 'allow',
    conditions_json = '{}';
