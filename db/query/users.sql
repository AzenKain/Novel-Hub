-- name: UpsertUser :one
INSERT INTO users (
    id,
    email,
    password_hash,
    auth_provider,
    full_name,
    avatar_url
) VALUES (
    ?, ?, ?, ?, ?, ?
)
ON CONFLICT(email)
DO UPDATE SET
    full_name = COALESCE(excluded.full_name, users.full_name),
    avatar_url = COALESCE(excluded.avatar_url, users.avatar_url)
RETURNING id, email, full_name, avatar_url, password_hash, auth_provider, is_deleted, token_version, refresh_token, created_at, updated_at;

-- name: UpdateProfile :one
UPDATE users
SET
    full_name = COALESCE(sqlc.narg('full_name'), full_name),
    avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url)
WHERE id = sqlc.arg('id') AND is_deleted = 0
RETURNING id, email, full_name, avatar_url, password_hash, auth_provider, is_deleted, token_version, refresh_token, created_at, updated_at;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = ?
WHERE id = ? AND is_deleted = 0;

-- name: UpdateUserRefreshToken :exec
UPDATE users
SET refresh_token = ?
WHERE id = ? AND is_deleted = 0;

-- name: RotateUserRefreshToken :execrows
UPDATE users
SET refresh_token = sqlc.arg('new_refresh_token')
WHERE id = sqlc.arg('id')
  AND is_deleted = 0
  AND refresh_token = sqlc.arg('current_refresh_token');

-- name: DeleteUser :exec
UPDATE users
SET is_deleted = 1
WHERE id = ?;

-- name: RestoreUser :exec
UPDATE users
SET is_deleted = 0
WHERE id = ?;

-- name: GetUserTokenVersion :one
SELECT token_version
FROM users
WHERE id = ? AND is_deleted = 0;

-- name: UpdateUserTokenVersion :exec
UPDATE users
SET token_version = ?
WHERE id = ? AND is_deleted = 0;

-- name: GetUserByID :one
SELECT id, email, full_name, avatar_url, password_hash, auth_provider, is_deleted, token_version, refresh_token, created_at, updated_at
FROM users
WHERE id = ? AND is_deleted = 0;

-- name: GetUserByIDWithoutDeleted :one
SELECT id, email, full_name, avatar_url, password_hash, auth_provider, is_deleted, token_version, refresh_token, created_at, updated_at
FROM users
WHERE id = ?;

-- name: GetUserByEmail :one
SELECT id, email, full_name, avatar_url, password_hash, auth_provider, is_deleted, token_version, refresh_token, created_at, updated_at
FROM users
WHERE email = ? AND is_deleted = 0;

-- name: SearchUserIDs :many
SELECT u.id
FROM users u
WHERE
    (sqlc.narg('is_deleted') IS NULL OR u.is_deleted = sqlc.narg('is_deleted'))
    AND (sqlc.narg('role_id') IS NULL OR EXISTS (
        SELECT 1 FROM user_roles ur
        WHERE ur.user_id = u.id AND ur.role_id = sqlc.narg('role_id')
    ))
    AND (sqlc.narg('auth_provider') IS NULL OR u.auth_provider = sqlc.narg('auth_provider'))
    AND (sqlc.narg('created_from') IS NULL OR u.created_at >= sqlc.narg('created_from'))
    AND (sqlc.narg('created_to') IS NULL OR u.created_at <= sqlc.narg('created_to'))
    AND (
        sqlc.narg('search_text') IS NULL OR
        CAST(u.id AS TEXT) LIKE '%' || sqlc.narg('search_text') || '%' OR
        lower(u.email) LIKE '%' || lower(sqlc.narg('search_text')) || '%' OR
        lower(COALESCE(u.full_name, '')) LIKE '%' || lower(sqlc.narg('search_text')) || '%'
    )
    AND (
        u.created_at <= COALESCE(CAST(sqlc.narg('cursor_created_at') AS TEXT), '9999-12-31 23:59:59')
        AND (sqlc.narg('cursor_created_at') IS NULL OR u.created_at < CAST(sqlc.narg('cursor_created_at') AS TEXT) OR u.id > sqlc.narg('cursor_id'))
    )
ORDER BY u.created_at DESC, u.id ASC
LIMIT sqlc.arg('limit');

-- name: CountUsers :one
SELECT count(*)
FROM users u
WHERE
    (sqlc.narg('is_deleted') IS NULL OR u.is_deleted = sqlc.narg('is_deleted'))
    AND (sqlc.narg('role_id') IS NULL OR EXISTS (
        SELECT 1 FROM user_roles ur
        WHERE ur.user_id = u.id AND ur.role_id = sqlc.narg('role_id')
    ))
    AND (sqlc.narg('auth_provider') IS NULL OR u.auth_provider = sqlc.narg('auth_provider'))
    AND (sqlc.narg('created_from') IS NULL OR u.created_at >= sqlc.narg('created_from'))
    AND (sqlc.narg('created_to') IS NULL OR u.created_at <= sqlc.narg('created_to'))
    AND (
        sqlc.narg('search_text') IS NULL OR
        CAST(u.id AS TEXT) LIKE '%' || sqlc.narg('search_text') || '%' OR
        lower(u.email) LIKE '%' || lower(sqlc.narg('search_text')) || '%' OR
        lower(COALESCE(u.full_name, '')) LIKE '%' || lower(sqlc.narg('search_text')) || '%'
    );

-- name: GetUsersByIDs :many
SELECT id, email, full_name, avatar_url, password_hash, auth_provider, is_deleted, token_version, refresh_token, created_at, updated_at
FROM users
WHERE id IN (sqlc.slice('ids'));

-- name: GetUserRoles :many
SELECT r.id, r.name, r.is_deleted, r.created_at, r.updated_at
FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = ? AND r.is_deleted = 0
ORDER BY r.name ASC;

-- name: SearchUserIDsFTS :many
SELECT u.id
FROM users u
WHERE u.id IN (SELECT user_id FROM fts_users WHERE haystack MATCH sqlc.arg('match_query'))
    AND (sqlc.narg('is_deleted') IS NULL OR u.is_deleted = sqlc.narg('is_deleted'))
    AND (sqlc.narg('role_id') IS NULL OR EXISTS (
        SELECT 1 FROM user_roles ur
        WHERE ur.user_id = u.id AND ur.role_id = sqlc.narg('role_id')
    ))
    AND (sqlc.narg('auth_provider') IS NULL OR u.auth_provider = sqlc.narg('auth_provider'))
    AND (sqlc.narg('created_from') IS NULL OR u.created_at >= sqlc.narg('created_from'))
    AND (sqlc.narg('created_to') IS NULL OR u.created_at <= sqlc.narg('created_to'))
    AND (
        u.created_at <= COALESCE(CAST(sqlc.narg('cursor_created_at') AS TEXT), '9999-12-31 23:59:59')
        AND (sqlc.narg('cursor_created_at') IS NULL OR u.created_at < CAST(sqlc.narg('cursor_created_at') AS TEXT) OR u.id > sqlc.narg('cursor_id'))
    )
ORDER BY u.created_at DESC, u.id ASC
LIMIT sqlc.arg('limit');

-- name: CountUsersFTS :one
SELECT count(*)
FROM users u
WHERE u.id IN (SELECT user_id FROM fts_users WHERE haystack MATCH sqlc.arg('match_query'))
    AND (sqlc.narg('is_deleted') IS NULL OR u.is_deleted = sqlc.narg('is_deleted'))
    AND (sqlc.narg('role_id') IS NULL OR EXISTS (
        SELECT 1 FROM user_roles ur
        WHERE ur.user_id = u.id AND ur.role_id = sqlc.narg('role_id')
    ))
    AND (sqlc.narg('auth_provider') IS NULL OR u.auth_provider = sqlc.narg('auth_provider'))
    AND (sqlc.narg('created_from') IS NULL OR u.created_at >= sqlc.narg('created_from'))
    AND (sqlc.narg('created_to') IS NULL OR u.created_at <= sqlc.narg('created_to'));

-- name: ProbeUserSearchMatches :one
SELECT count(*) FROM (
    SELECT user_id FROM fts_users
    WHERE haystack MATCH sqlc.arg('match_query')
    LIMIT sqlc.arg('cap')
);
