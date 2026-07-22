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
    ('book.read', 'Read books and online reader'),
    ('book.tts', 'Use text-to-speech audio reader'),
    ('book.search.deep', 'Deep search inside book content'),
    ('book.download', 'Download raw book files'),
    ('book.send_email', 'Send book files to email / Send-to-Kindle'),
    ('book.share', 'Share books and create public share links'),
    ('book.bookmark', 'Bookmark books and manage reading lists'),
    ('book.collection', 'Manage personal book collections'),
    ('book.highlight', 'Create and manage reading highlights and notes'),
    ('book.review.create', 'Create and update book reviews'),
    ('book.review.delete', 'Delete book reviews'),
    ('user.stats.read', 'View reading statistics and progress'),
    ('tracker.sync', 'Sync reading progress with external trackers'),
    ('book.upload', 'Upload new books to library'),
    ('book.edit', 'Edit book details and metadata'),
    ('book.metadata.fetch', 'Fetch metadata automatically from online providers'),
    ('book.delete', 'Delete books or book files'),
    ('book.duplicate.manage', 'Manage and clean up duplicate book files'),
    ('book.archive', 'Archive or unarchive books'),
    ('book.bulk.manage', 'Perform bulk operations on books'),
    ('library.read', 'Read libraries and details'),
    ('library.manage', 'Manage libraries'),
    ('opds.read', 'Access OPDS catalog feed'),
    ('opds.download', 'Download books via OPDS'),
    ('kobo.sync', 'Sync reading progress with Kobo devices'),
    ('calibre.sync', 'Sync and import from Calibre server'),
    ('admin.access', 'Access admin dashboard'),
    ('user.manage', 'Manage user accounts and roles'),
    ('role.manage', 'Manage roles and permissions'),
    ('setting.manage', 'Manage application settings'),
    ('job.read', 'View background job status'),
    ('job.manage', 'Trigger background maintenance jobs'),
    ('system.log.read', 'View system logs in admin area'),
    ('system.backup', 'Backup and restore system data'),
    ('webhook.manage', 'Manage webhooks and integrations')
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
    ('guest_access.library_ids', '[]')
ON CONFLICT(key) DO NOTHING;

