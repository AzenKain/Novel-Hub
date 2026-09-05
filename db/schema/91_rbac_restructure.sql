
INSERT INTO roles (id, name, is_system, is_admin, is_banned, auto_assign, description)
VALUES ('01920000-0000-7000-8000-000000000005', 'GUEST', 1, 0, 0, 0, 'Built-in unauthenticated visitor role')
ON CONFLICT(name) DO UPDATE SET
    is_system = excluded.is_system,
    is_banned = excluded.is_banned,
    description = excluded.description;

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

INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('book.read', 'book.tts', 'library.read', 'opds.read', 'opds.download')
WHERE r.name = 'GUEST'
ON CONFLICT(role_id, permission_key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN (
    'book.read', 'book.tts', 'book.search.deep', 'book.download', 'book.send_email',
    'book.share', 'book.bookmark', 'book.collection', 'book.highlight',
    'book.review.create', 'user.stats.read', 'tracker.sync', 'library.read',
    'opds.read', 'opds.download', 'kobo.sync', 'komga.sync'
)
WHERE r.name = 'USER'
ON CONFLICT(role_id, permission_key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN (
    'book.read', 'book.tts', 'book.search.deep', 'book.download', 'book.send_email',
    'book.share', 'book.bookmark', 'book.collection', 'book.highlight',
    'book.review.create', 'book.review.delete', 'user.stats.read', 'tracker.sync',
    'book.upload', 'book.edit', 'book.metadata.fetch', 'book.delete',
    'book.duplicate.manage', 'book.archive', 'book.bulk.manage', 'library.read',
    'library.manage', 'opds.read', 'opds.download', 'kobo.sync', 'komga.sync', 'calibre.sync',
    'podcast.manage',
    'admin.access', 'job.read'
)
WHERE r.name = 'MOD'
ON CONFLICT(role_id, permission_key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'ADMIN'
ON CONFLICT(role_id, permission_key) DO UPDATE SET
    effect = 'allow',
    conditions_json = '{}';

UPDATE roles SET position = 100 WHERE name = 'ADMIN';
UPDATE roles SET position = 80 WHERE name = 'MOD';
UPDATE roles SET position = 50 WHERE name = 'USER';
UPDATE roles SET position = 10 WHERE name = 'GUEST';
UPDATE roles SET position = -1 WHERE name = 'BANNED';
