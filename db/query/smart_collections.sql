-- name: GetSmartCollection :one
SELECT * FROM smart_collections
WHERE id = ? AND user_id = ?;

-- name: ListSmartCollectionsByUser :many
SELECT * FROM smart_collections
WHERE user_id = ?
ORDER BY created_at DESC;

-- name: CreateSmartCollection :one
INSERT INTO smart_collections (
    id, user_id, name, rule_json
) VALUES (
    ?, ?, ?, ?
)
RETURNING *;

-- name: DeleteSmartCollection :exec
DELETE FROM smart_collections
WHERE id = ? AND user_id = ?;
