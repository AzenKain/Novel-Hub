-- System role IDs are fixed UUIDv7 literals so seeds stay idempotent across restarts.
INSERT INTO roles (id, name, is_system, is_admin, is_banned, auto_assign, description) VALUES
    ('01920000-0000-7000-8000-000000000001', 'USER',   1, 0, 0, 1, 'Default user role'),
    ('01920000-0000-7000-8000-000000000002', 'ADMIN',  1, 1, 0, 0, 'Built-in administrator role with full access'),
    ('01920000-0000-7000-8000-000000000003', 'MOD',    1, 0, 0, 0, 'Built-in moderator role'),
    ('01920000-0000-7000-8000-000000000004', 'BANNED', 1, 0, 1, 0, 'Built-in blocked account role'),
    ('01920000-0000-7000-8000-000000000005', 'GUEST',  1, 0, 0, 0, 'Built-in unauthenticated visitor role')
ON CONFLICT(name) DO UPDATE SET
    is_system = excluded.is_system,
    is_admin = excluded.is_admin,
    is_banned = excluded.is_banned,
    auto_assign = excluded.auto_assign,
    description = excluded.description;

-- ADMIN gets all permissions
INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'ADMIN'
ON CONFLICT(role_id, permission_key) DO UPDATE SET
    effect = 'allow',
    conditions_json = '{}';

-- GUEST default permissions. Deliberately no book.offline even though it has book.read:
-- an offline copy outlives the anonymous session that made it and stays readable on a shared device.
INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN (
    'book.read',
    'book.tts',
    'library.read',
    'opds.read'
)
WHERE r.name = 'GUEST'
ON CONFLICT(role_id, permission_key) DO NOTHING;

-- USER default permissions
INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN (
    'book.read',
    'book.tts',
    'book.search.deep',
    'book.download',
    'book.offline',
    'book.send_email',
    'book.share',
    'book.bookmark',
    'book.collection',
    'book.highlight',
    'book.review.create',
    'user.stats.read',
    'tracker.sync',
    'library.read',
    'opds.read',
    'opds.download',
    'kobo.sync'
)
WHERE r.name = 'USER'
ON CONFLICT(role_id, permission_key) DO NOTHING;

-- MOD default permissions
INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN (
    'book.read',
    'book.tts',
    'book.search.deep',
    'book.download',
    'book.offline',
    'book.send_email',
    'book.share',
    'book.bookmark',
    'book.collection',
    'book.highlight',
    'book.review.create',
    'book.review.delete',
    'user.stats.read',
    'tracker.sync',
    'book.upload',
    'book.edit',
    'book.metadata.fetch',
    'book.delete',
    'book.duplicate.manage',
    'book.archive',
    'book.bulk.manage',
    'library.read',
    'library.manage',
    'opds.read',
    'opds.download',
    'kobo.sync',
    'calibre.sync',
    'admin.access',
    'job.read'
)
WHERE r.name = 'MOD'
ON CONFLICT(role_id, permission_key) DO NOTHING;

