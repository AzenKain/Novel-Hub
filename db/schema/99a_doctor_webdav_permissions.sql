-- Migration 100: Book Doctor Repair & WebDAV Permissions

INSERT INTO permissions (key, description) VALUES
    ('book.repair', 'Diagnose and repair corrupted EPUB book files'),
    ('webdav.read', 'Browse and access library books via WebDAV'),
    ('webdav.download', 'Download and sync book files via WebDAV')
ON CONFLICT(key) DO UPDATE SET
    description = excluded.description;

-- Grant USER webdav permissions
INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('webdav.read', 'webdav.download')
WHERE r.name = 'USER'
ON CONFLICT(role_id, permission_key) DO NOTHING;

-- Grant MOD book.repair and webdav permissions
INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN ('book.repair', 'webdav.read', 'webdav.download')
WHERE r.name = 'MOD'
ON CONFLICT(role_id, permission_key) DO NOTHING;

-- Grant ADMIN all permissions
INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'ADMIN'
ON CONFLICT(role_id, permission_key) DO UPDATE SET
    effect = 'allow',
    conditions_json = '{}';
