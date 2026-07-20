INSERT INTO roles (id, name, is_system, is_admin, auto_assign, description) VALUES
    (1, 'USER',   1, 0, 1, 'Default user role'),
    (2, 'ADMIN',  1, 1, 0, 'Built-in administrator role with full access'),
    (3, 'MOD',    1, 0, 0, 'Built-in moderator role'),
    (4, 'BANNED', 1, 0, 0, 'Built-in blocked account role')
ON CONFLICT(name) DO UPDATE SET
    is_system = excluded.is_system,
    is_admin = excluded.is_admin,
    auto_assign = excluded.auto_assign,
    description = excluded.description;

INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'ADMIN'
ON CONFLICT(role_id, permission_key) DO UPDATE SET
    effect = 'allow',
    conditions_json = '{}';

INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN (
    'book.read',
    'book.download',
    'book.bookmark',
    'book.collection',
    'book.review.create',
    'library.read'
)
WHERE r.name = 'USER'
ON CONFLICT(role_id, permission_key) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, p.key, 'allow', '{}'
FROM roles r
JOIN permissions p ON p.key IN (
    'book.read',
    'book.download',
    'book.bookmark',
    'book.collection',
    'book.review.create',
    'book.review.delete',
    'book.manage',
    'library.read',
    'library.manage',
    'admin.access',
    'job.read'
)
WHERE r.name = 'MOD'
ON CONFLICT(role_id, permission_key) DO NOTHING;
