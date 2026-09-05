-- name: CountJobs :one
SELECT COUNT(*) FROM jobs
WHERE (sqlc.arg('status') = '' OR status = sqlc.arg('status'))
  AND (sqlc.arg('type') = '' OR type = sqlc.arg('type'));

-- name: ListFilteredJobIDs :many
SELECT id FROM jobs
WHERE (sqlc.arg('status') = '' OR status = sqlc.arg('status'))
  AND (sqlc.arg('type') = '' OR type = sqlc.arg('type'))
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: MarkRunningJobsInterrupted :exec
UPDATE jobs
SET status = 'failed', error_msg = 'server restarted while job was running', updated_at = CURRENT_TIMESTAMP
WHERE status = 'running';

-- name: CreateJobSchedule :one
INSERT INTO job_schedules (
    id, name, task_type, payload_json, interval_minutes, enabled, next_run_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, name, task_type, payload_json, interval_minutes, enabled, next_run_at,
          last_run_at, last_job_id, created_at, updated_at;

-- name: GetJobSchedule :one
SELECT id, name, task_type, payload_json, interval_minutes, enabled, next_run_at,
       last_run_at, last_job_id, created_at, updated_at
FROM job_schedules WHERE id = ? LIMIT 1;

-- name: UpdateJobSchedule :one
UPDATE job_schedules
SET name = ?, task_type = ?, payload_json = ?, interval_minutes = ?, enabled = ?,
    next_run_at = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, name, task_type, payload_json, interval_minutes, enabled, next_run_at,
          last_run_at, last_job_id, created_at, updated_at;

-- name: DeleteJobSchedule :exec
DELETE FROM job_schedules WHERE id = ?;

-- name: ListJobScheduleIDs :many
SELECT id FROM job_schedules ORDER BY created_at DESC;

-- name: ListDueJobScheduleIDs :many
SELECT id FROM job_schedules
WHERE enabled = 1 AND next_run_at <= sqlc.arg('now')
ORDER BY next_run_at ASC
LIMIT 100;

-- name: ClaimJobSchedule :execrows
UPDATE job_schedules
SET last_run_at = sqlc.arg('now'), last_job_id = sqlc.arg('job_id'),
    next_run_at = sqlc.arg('next_run_at'), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg('id') AND enabled = 1 AND next_run_at <= sqlc.arg('now');

-- name: DatabaseHealthCheck :one
SELECT COUNT(*) FROM jobs;

-- name: ReleaseJobScheduleClaim :execrows
UPDATE job_schedules
SET last_job_id = NULL, next_run_at = sqlc.arg('retry_at'), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg('id') AND last_job_id = sqlc.arg('job_id');

-- name: PruneFinishedJobs :execrows
DELETE FROM jobs
WHERE id IN (
    SELECT id FROM jobs
    WHERE status IN ('completed', 'failed')
    ORDER BY updated_at DESC
    LIMIT -1 OFFSET sqlc.arg('keep')
);

-- name: GetJobSchedulesByIDs :many
SELECT id, name, task_type, payload_json, interval_minutes, enabled, next_run_at,
       last_run_at, last_job_id, created_at, updated_at
FROM job_schedules
WHERE id IN (sqlc.slice('ids'));

-- name: CountUnfinishedJobs :one
SELECT COUNT(*) FROM jobs WHERE status IN ('pending', 'running');
