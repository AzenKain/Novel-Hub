-- name: GetSmartFilter :one
SELECT id, user_id, name, rules_json, is_pinned_sidebar, is_pinned_home, home_position, created_at, updated_at
FROM smart_filters
WHERE id = ? AND user_id = ?;

-- name: ListSmartFiltersByUser :many
SELECT id, user_id, name, rules_json, is_pinned_sidebar, is_pinned_home, home_position, created_at, updated_at
FROM smart_filters
WHERE user_id = ?
ORDER BY home_position ASC, created_at DESC;

-- name: CreateSmartFilter :one
INSERT INTO smart_filters (
    id, user_id, name, rules_json, is_pinned_sidebar, is_pinned_home, home_position
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
)
RETURNING id, user_id, name, rules_json, is_pinned_sidebar, is_pinned_home, home_position, created_at, updated_at;

-- name: UpdateSmartFilter :one
UPDATE smart_filters
SET name = ?, rules_json = ?, is_pinned_sidebar = ?, is_pinned_home = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND user_id = ?
RETURNING id, user_id, name, rules_json, is_pinned_sidebar, is_pinned_home, home_position, created_at, updated_at;

-- name: DeleteSmartFilter :exec
DELETE FROM smart_filters
WHERE id = ? AND user_id = ?;

-- name: UpdateSmartFilterPinSidebar :one
UPDATE smart_filters
SET is_pinned_sidebar = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND user_id = ?
RETURNING id, user_id, name, rules_json, is_pinned_sidebar, is_pinned_home, home_position, created_at, updated_at;

-- name: UpdateSmartFilterPinHome :one
UPDATE smart_filters
SET is_pinned_home = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND user_id = ?
RETURNING id, user_id, name, rules_json, is_pinned_sidebar, is_pinned_home, home_position, created_at, updated_at;

-- name: UpdateSmartFilterHomePosition :one
UPDATE smart_filters
SET home_position = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND user_id = ?
RETURNING id, user_id, name, rules_json, is_pinned_sidebar, is_pinned_home, home_position, created_at, updated_at;

-- name: ListSmartFilterIDsByUser :many
SELECT id
FROM smart_filters
WHERE user_id = ?
ORDER BY home_position ASC, created_at DESC;

-- name: GetSmartFiltersByIDs :many
SELECT id, user_id, name, rules_json, is_pinned_sidebar, is_pinned_home, home_position, created_at, updated_at
FROM smart_filters
WHERE id IN (sqlc.slice('ids'));
