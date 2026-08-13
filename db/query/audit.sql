-- name: CreateAuditLog :one
INSERT INTO audit_logs (id, actor_id, actor_email, action, target_type, target_id, target_label, ip)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, actor_id, actor_email, action, target_type, target_id, target_label, ip, created_at;

-- name: ListAuditLogs :many
SELECT id, actor_id, actor_email, action, target_type, target_id, target_label, ip, created_at
FROM audit_logs
WHERE
    (sqlc.narg('action') IS NULL OR action = sqlc.narg('action'))
    AND (sqlc.narg('actor_id') IS NULL OR actor_id = sqlc.narg('actor_id'))
    AND (
        created_at <= COALESCE(CAST(sqlc.narg('cursor_created_at') AS TEXT), '9999-12-31 23:59:59')
        AND (sqlc.narg('cursor_created_at') IS NULL OR created_at < CAST(sqlc.narg('cursor_created_at') AS TEXT) OR id > sqlc.narg('cursor_id'))
    )
ORDER BY created_at DESC, id ASC
LIMIT sqlc.arg('limit');

-- name: CountAuditLogs :one
SELECT count(*)
FROM audit_logs
WHERE
    (sqlc.narg('action') IS NULL OR action = sqlc.narg('action'))
    AND (sqlc.narg('actor_id') IS NULL OR actor_id = sqlc.narg('actor_id'));

-- name: ListAuditActions :many
SELECT DISTINCT action FROM audit_logs ORDER BY action ASC;

-- name: PruneAuditLogs :execrows
DELETE FROM audit_logs
WHERE created_at < datetime('now', '-' || CAST(sqlc.arg('keep_days') AS TEXT) || ' days');
