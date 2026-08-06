-- name: CreateAuditLog :one
INSERT INTO audit_logs (id, actor_id, actor_email, action, target_type, target_id, target_label, ip)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, actor_id, actor_email, action, target_type, target_id, target_label, ip, created_at;

-- name: ListAuditLogs :many
-- Keyset pagination on (created_at DESC, id ASC), the same shape SearchUserIDs uses, so
-- rows inserted while an admin is paging cannot shift the window and duplicate a row.
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
-- Age-based, unlike PruneFinishedJobs which keeps a row count: an audit trail is only
-- useful if "the last 90 days" means the same thing on a busy and a quiet instance.
DELETE FROM audit_logs
WHERE created_at < datetime('now', '-' || CAST(sqlc.arg('keep_days') AS TEXT) || ' days');
