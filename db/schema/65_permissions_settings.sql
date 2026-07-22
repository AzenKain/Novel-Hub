CREATE TABLE IF NOT EXISTS permissions (
    key TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS role_permissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_key TEXT NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
    effect TEXT NOT NULL DEFAULT 'allow',
    conditions_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(role_id, permission_key),
    CHECK (effect IN ('allow', 'deny'))
);

CREATE TABLE IF NOT EXISTS app_settings (
    key TEXT PRIMARY KEY,
    value_json TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS setup_state (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_key ON role_permissions(permission_key);
CREATE INDEX IF NOT EXISTS idx_roles_auto_assign ON roles(auto_assign) WHERE is_deleted = 0;

CREATE TRIGGER IF NOT EXISTS trigger_permissions_updated_at
AFTER UPDATE ON permissions
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE permissions SET updated_at = datetime('now') WHERE key = OLD.key;
END;

CREATE TRIGGER IF NOT EXISTS trigger_role_permissions_updated_at
AFTER UPDATE ON role_permissions
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE role_permissions SET updated_at = datetime('now') WHERE id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS trigger_app_settings_updated_at
AFTER UPDATE ON app_settings
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE app_settings SET updated_at = datetime('now') WHERE key = OLD.key;
END;

CREATE TRIGGER IF NOT EXISTS trigger_setup_state_updated_at
AFTER UPDATE ON setup_state
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE setup_state SET updated_at = datetime('now') WHERE key = OLD.key;
END;

INSERT INTO permissions (key, description) VALUES
    ('book.read', 'Read books and book metadata'),
    ('book.download', 'Download book files'),
    ('book.share', 'Share books and generate share links'),
    ('book.bookmark', 'Bookmark books'),
    ('book.collection', 'Manage personal collections'),
    ('book.review.create', 'Create and update book reviews'),
    ('book.review.delete', 'Delete book reviews'),
    ('book.manage', 'Manage books and metadata'),
    ('library.read', 'Read libraries'),
    ('library.manage', 'Manage libraries and uploads'),
    ('role.manage', 'Manage roles and permissions'),
    ('user.manage', 'Manage users'),
    ('setting.manage', 'Manage application settings'),
    ('admin.access', 'Access admin area'),
    ('job.read', 'Read background job status'),
    ('webhook.manage', 'Manage webhooks and notification integrations')
ON CONFLICT(key) DO UPDATE SET
    description = excluded.description;

INSERT INTO app_settings (key, value_json) VALUES
    ('site.title', '"NovelHub"'),
    ('site.description', '"Local novel library manager"'),
    ('site.favicon', '""'),
    ('site.logo', '""'),
    ('site.meta_description', '"Self-hosted local-first digital book library manager."'),
    ('sidebar.visible_items', '["books","hot_books","downloaded_books","top_rated_books","bookmarked_books","read_books","unread_books","subjects","series","authors","publishers","languages","file_formats","ratings","archived_books","collections"]'),
    ('home.sections', '{"random_books":true,"top_books":true}'),
    ('auth.registration_enabled', 'true'),
    ('guest_access.mode', '"all"'),
    ('guest_access.library_ids', '[]'),
    ('download.mode', '"all"'),
    ('download.library_ids', '[]'),
    ('bookmark.mode', '"all"'),
    ('bookmark.library_ids', '[]'),
    ('collection.mode', '"all"'),
    ('collection.library_ids', '[]'),
    ('review.mode', '"all"'),
    ('review.library_ids', '[]')
ON CONFLICT(key) DO NOTHING;
