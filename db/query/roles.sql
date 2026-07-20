-- name: CreateRole :one
INSERT INTO roles (name, description, is_system, is_admin, auto_assign)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetRoleByName :one
SELECT *
FROM roles
WHERE name = ? AND is_deleted = 0;

-- name: GetRoleByID :one
SELECT *
FROM roles
WHERE id = ? AND is_deleted = 0;

-- name: GetRoleIDs :many
SELECT id
FROM roles
WHERE is_deleted = 0
ORDER BY name ASC;

-- name: GetAutoAssignRoleIDs :many
SELECT id
FROM roles
WHERE is_deleted = 0 AND auto_assign = 1 AND is_admin = 0;

-- name: GetRolesByIDs :many
SELECT *
FROM roles WHERE id IN (sqlc.slice('ids'));

-- name: UpdateRole :one
UPDATE roles
SET name = ?, description = ?, auto_assign = ?
WHERE id = ? AND is_deleted = 0 AND is_system = 0
RETURNING *;

-- name: UpdateSystemRoleDescription :one
UPDATE roles
SET description = ?
WHERE id = ? AND is_deleted = 0 AND is_system = 1
RETURNING *;

-- name: DeleteRole :exec
UPDATE roles
SET is_deleted = 1
WHERE id = ? AND is_system = 0;

-- name: RestoreRole :exec
UPDATE roles
SET is_deleted = 0
WHERE id = ?;

-- name: CreateUserRole :exec
INSERT INTO user_roles (user_id, role_id)
VALUES (?, ?)
ON CONFLICT DO NOTHING;

-- name: DeleteUserRole :exec
DELETE FROM user_roles
WHERE user_id = ? AND role_id = ?;

-- name: BulkDeleteRolesFromUser :exec
DELETE FROM user_roles
WHERE user_id = ?;

-- name: BulkDeleteUsersFromRole :exec
DELETE FROM user_roles
WHERE role_id = ?;

-- name: ListPermissions :many
SELECT * FROM permissions
ORDER BY key ASC;

-- name: UpsertPermission :exec
INSERT INTO permissions (key, description)
VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET description = excluded.description;

-- name: ListRolePermissions :many
SELECT *
FROM role_permissions
ORDER BY role_id ASC, permission_key ASC;

-- name: GetRolePermissions :many
SELECT *
FROM role_permissions
WHERE role_id = ?
ORDER BY permission_key ASC;

-- name: DeleteRolePermissions :exec
DELETE FROM role_permissions
WHERE role_id = ?;

-- name: UpsertRolePermission :exec
INSERT INTO role_permissions (role_id, permission_key, effect, conditions_json)
VALUES (?, ?, ?, ?)
ON CONFLICT(role_id, permission_key) DO UPDATE SET
    effect = excluded.effect,
    conditions_json = excluded.conditions_json;

-- name: ListPermissionKeys :many
SELECT key FROM permissions ORDER BY key ASC;

-- name: GetPermissionsByKeys :many
SELECT * FROM permissions WHERE key IN (sqlc.slice('keys'));

-- name: ListRolePermissionIDs :many
SELECT id FROM role_permissions ORDER BY role_id ASC, permission_key ASC;

-- name: GetRolePermissionsByIDs :many
SELECT * FROM role_permissions WHERE id IN (sqlc.slice('ids'));

-- name: GetRolePermissionIDs :many
SELECT id FROM role_permissions WHERE role_id = ? ORDER BY permission_key ASC;
