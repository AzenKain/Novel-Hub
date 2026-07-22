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

-- name: ListActiveWebhooks :many
SELECT id, name, url, template_type, secret, custom_headers, events, is_active, created_at, updated_at
FROM webhooks
WHERE is_active = 1;

-- name: ListAllWebhooks :many
SELECT id, name, url, template_type, secret, custom_headers, events, is_active, created_at, updated_at
FROM webhooks
ORDER BY created_at DESC;

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
