CREATE TABLE IF NOT EXISTS permissions (
    key TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS role_permissions (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || substr(hex(randomblob(2)), 2) || '-' || substr('89ab', abs(random()) % 4 + 1, 1) || substr(hex(randomblob(2)), 2) || '-' || hex(randomblob(6)))),
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
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
    ('book.offline', 'Save books to the browser for offline reading'),
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
    ('book.repair', 'Diagnose and repair corrupted EPUB book files'),
    ('book.metadata.fetch', 'Fetch metadata automatically from online providers'),
    ('book.delete', 'Delete books or book files'),
    ('book.duplicate.manage', 'Manage and clean up duplicate book files'),
    ('book.archive', 'Archive or unarchive books'),
    ('book.bulk.manage', 'Perform bulk operations on books'),
    ('library.read', 'Read libraries and details'),
    ('library.manage', 'Manage libraries'),
    ('opds.read', 'Access OPDS catalog feed'),
    ('opds.download', 'Download books via OPDS'),
    ('webdav.read', 'Browse and access library books via WebDAV'),
    ('webdav.download', 'Download and sync book files via WebDAV'),
    ('kobo.sync', 'Sync reading progress with Kobo devices'),
    ('komga.sync', 'Sync manga and reading progress with Mihon/Tachiyomi'),
    ('calibre.sync', 'Sync and import from Calibre server'),
    ('podcast.manage', 'Manage podcast subscriptions and downloads'),
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
    ('server.url', '""'),
    ('sidebar.visible_items', '["books","hot_books","downloaded_books","top_rated_books","bookmarked_books","read_books","unread_books","subjects","series","authors","publishers","languages","file_formats","ratings","archived_books","collections"]'),
    ('home.sections', '{"random_books":true,"top_books":true}'),
    ('auth.registration_enabled', 'true'),
    ('auth.login_required', 'false'),
    ('tracker.anilist_enabled', 'true'),
    ('guest_access.mode', '"all"'),
    ('guest_access.library_ids', '[]'),
    ('limits.upload_chunk_bytes', '33554432'),
    ('limits.upload_chunks', '256'),
    ('limits.upload_sessions', '16'),
    ('limits.upload_bytes', '8589934592'),
    ('limits.upload_session_ttl_seconds', '21600'),
    ('limits.cover_bytes', '33554432'),
    ('limits.site_asset_bytes', '10485760'),

    ('metadata.auto_enrich_enabled', 'false'),
    ('metadata.webp_cover_enabled', 'false'),

    ('auth.proxy_auth_enabled', 'false'),
    ('auth.proxy_auth_headers', '["X-Forwarded-User", "Remote-User", "X-Forwarded-Email"]'),
    ('auth.proxy_auth_trusted_proxies', '["127.0.0.1", "::1"]'),
    ('auth.proxy_auth_auto_create', 'false'),

    ('oauth.google.enabled', 'false'),
    ('oauth.google.client_id', '""'),
    ('oauth.google.client_secret', '""'),
    ('oauth.google.redirect_uri', '""'),

    ('oauth.github.enabled', 'false'),
    ('oauth.github.client_id', '""'),
    ('oauth.github.client_secret', '""'),
    ('oauth.github.redirect_uri', '""'),

    ('oauth.discord.enabled', 'false'),
    ('oauth.discord.client_id', '""'),
    ('oauth.discord.client_secret', '""'),
    ('oauth.discord.redirect_uri', '""'),

    ('oauth.oidc.enabled', 'false'),
    ('oauth.oidc.name', '"OpenID Connect"'),
    ('oauth.oidc.issuer_url', '""'),
    ('oauth.oidc.client_id', '""'),
    ('oauth.oidc.client_secret', '""'),
    ('oauth.oidc.redirect_uri', '""'),
    ('oauth.oidc.scopes', '["openid", "profile", "email"]'),

    ('hardcover.enabled', 'false'),
    ('hardcover.client_id', '""'),
    ('hardcover.client_secret', '""')
ON CONFLICT(key) DO NOTHING;

INSERT INTO app_settings (key, value_json) VALUES
    ('smtp.enabled', 'false'),
    ('smtp.host', '""'),
    ('smtp.port', '587'),
    ('smtp.username', '""'),
    ('smtp.password', '""'),
    ('smtp.from_email', '""'),
    ('smtp.tls_mode', '"starttls"'),
    ('smtp.allow_private_networks', 'false'),
    ('smtp.max_attachment_mb', '50')
ON CONFLICT(key) DO NOTHING;
