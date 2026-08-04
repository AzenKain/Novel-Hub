-- name: CreateRole :one
INSERT INTO roles (id, name, description, is_system, is_admin, auto_assign)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, name, description, is_system, is_admin, is_banned, auto_assign, position, is_deleted, created_at, updated_at;

-- name: GetRoleByName :one
SELECT id, name, description, is_system, is_admin, is_banned, auto_assign, position, is_deleted, created_at, updated_at
FROM roles
WHERE name = ? AND is_deleted = 0;

-- name: GetRoleByID :one
SELECT id, name, description, is_system, is_admin, is_banned, auto_assign, position, is_deleted, created_at, updated_at
FROM roles
WHERE id = ? AND is_deleted = 0;

-- name: GetRoleIDs :many
SELECT id
FROM roles
WHERE is_deleted = 0
ORDER BY position DESC, name ASC;

-- name: UpdateRolePosition :exec
UPDATE roles
SET position = ?
WHERE id = ? AND is_deleted = 0;

-- name: GetAutoAssignRoleIDs :many
SELECT id
FROM roles
WHERE is_deleted = 0 AND auto_assign = 1 AND is_admin = 0;

-- name: GetRolesByIDs :many
SELECT id, name, description, is_system, is_admin, is_banned, auto_assign, position, is_deleted, created_at, updated_at
FROM roles WHERE id IN (sqlc.slice('ids'));

-- name: UpdateRole :one
UPDATE roles
SET name = ?, description = ?, auto_assign = ?
WHERE id = ? AND is_deleted = 0 AND is_system = 0
RETURNING id, name, description, is_system, is_admin, is_banned, auto_assign, position, is_deleted, created_at, updated_at;

-- name: UpdateSystemRoleDescription :one
UPDATE roles
SET description = ?, auto_assign = ?
WHERE id = ? AND is_deleted = 0 AND is_system = 1 AND is_admin = 0
RETURNING id, name, description, is_system, is_admin, is_banned, auto_assign, position, is_deleted, created_at, updated_at;

-- name: DeleteRole :exec
UPDATE roles
SET is_deleted = 1
WHERE id = ? AND is_system = 0;

-- name: CreateUserRole :exec
INSERT INTO user_roles (user_id, role_id)
VALUES (?, ?)
ON CONFLICT DO NOTHING;

-- name: CountActiveAdminUsers :one
SELECT COUNT(DISTINCT u.id)
FROM users u
JOIN user_roles ur ON u.id = ur.user_id
JOIN roles r ON ur.role_id = r.id
WHERE u.is_deleted = 0 AND r.is_deleted = 0 AND (r.name = 'ADMIN' OR r.is_admin = 1);

-- name: BulkDeleteRolesFromUser :exec
DELETE FROM user_roles
WHERE user_id = ?;

-- name: ListPermissions :many
SELECT key, description, created_at, updated_at FROM permissions
ORDER BY key ASC;

-- name: ListRolePermissions :many
SELECT id, role_id, permission_key, effect, conditions_json, created_at, updated_at
FROM role_permissions
ORDER BY role_id ASC, permission_key ASC;

-- name: GetRolePermissions :many
SELECT id, role_id, permission_key, effect, conditions_json, created_at, updated_at
FROM role_permissions
WHERE role_id = ?
ORDER BY permission_key ASC;

-- name: DeleteRolePermissions :exec
DELETE FROM role_permissions
WHERE role_id = ?;

-- name: UpsertRolePermission :exec
INSERT INTO role_permissions (id, role_id, permission_key, effect, conditions_json)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(role_id, permission_key) DO UPDATE SET
    effect = excluded.effect,
    conditions_json = excluded.conditions_json;

-- name: ListPermissionKeys :many
SELECT key FROM permissions ORDER BY key ASC;

-- name: GetPermissionsByKeys :many
SELECT key, description, created_at, updated_at FROM permissions WHERE key IN (sqlc.slice('keys'));

-- name: ListRolePermissionIDs :many
SELECT id FROM role_permissions ORDER BY role_id ASC, permission_key ASC;

-- name: GetRolePermissionsByIDs :many
SELECT id, role_id, permission_key, effect, conditions_json, created_at, updated_at FROM role_permissions WHERE id IN (sqlc.slice('ids'));

-- name: GetRolePermissionIDs :many
SELECT id FROM role_permissions WHERE role_id = ? ORDER BY permission_key ASC;
