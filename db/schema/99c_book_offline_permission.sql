-- book.offline is granted to exactly the roles that already hold book.download: an offline
-- copy is a download that lives in the browser, so anyone allowed one is allowed the other.
-- GUEST is deliberately excluded even though it holds book.read -- an offline copy outlives
-- the anonymous session it was made in and stays readable on a shared device.
INSERT INTO permissions (key, description) VALUES
    ('book.offline', 'Save books to the browser for offline reading')
ON CONFLICT(key) DO UPDATE SET
    description = excluded.description;

INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT rp.role_id, 'book.offline', rp.effect, rp.conditions_json
FROM role_permissions rp
WHERE rp.permission_key = 'book.download'
ON CONFLICT(role_id, permission_key) DO NOTHING;
