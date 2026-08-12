-- name: CreateWebhook :one
INSERT INTO webhooks (
    id, name, url, template_type, secret, custom_headers, events, is_active, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
RETURNING id, name, url, template_type, secret, custom_headers, events, is_active, created_at, updated_at;

-- name: GetWebhookByID :one
SELECT id, name, url, template_type, secret, custom_headers, events, is_active, created_at, updated_at
FROM webhooks
WHERE id = ?;

-- name: ListActiveWebhookIDs :many
SELECT id
FROM webhooks
WHERE is_active = 1
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListAllWebhookIDs :many
SELECT id
FROM webhooks
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateWebhook :one
UPDATE webhooks
SET name = ?,
    url = ?,
    template_type = ?,
    secret = ?,
    custom_headers = ?,
    events = ?,
    is_active = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, name, url, template_type, secret, custom_headers, events, is_active, created_at, updated_at;

-- name: DeleteWebhook :exec
DELETE FROM webhooks
WHERE id = ?;

-- name: GetWebhooksByIDs :many
SELECT id, name, url, template_type, secret, custom_headers, events, is_active, created_at, updated_at
FROM webhooks
WHERE id IN (sqlc.slice('ids'));

-- name: CountWebhooks :one
SELECT COUNT(*) FROM webhooks;
