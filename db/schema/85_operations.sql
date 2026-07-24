CREATE TABLE IF NOT EXISTS job_schedules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    task_type TEXT NOT NULL,
    payload_json TEXT,
    interval_minutes INTEGER NOT NULL CHECK (interval_minutes >= 5),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    next_run_at DATETIME NOT NULL,
    last_run_at DATETIME,
    last_job_id TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_job_schedules_due ON job_schedules(enabled, next_run_at);
