-- Grant opds.download permission to GUEST role for existing databases
INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
SELECT r.id, 'opds.download', 'allow', '{}'
FROM roles r
WHERE r.name = 'GUEST'
ON CONFLICT(role_id, permission_key) DO NOTHING;
