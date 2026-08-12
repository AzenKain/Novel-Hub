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

-- Admin actions only. Deliberately stores no before/after values: UpdateSettings carries
-- smtp.password, so a value diff would copy the mail credential straight back out of the
-- encrypted setting it was put into.
CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY,
    -- Kept as SET NULL rather than CASCADE: deleting a user must not erase the record of
    -- what that user did, and actor_email preserves who it was after the row is gone.
    actor_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    actor_email TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    target_type TEXT NOT NULL DEFAULT '',
    target_id TEXT,
    target_label TEXT NOT NULL DEFAULT '',
    ip TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Matches the keyset order (created_at DESC, id ASC) the list query pages on.
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC, id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs(actor_id, created_at DESC);

-- Insert a default schedule for metadata enrichment (runs daily, disabled by default)
INSERT INTO job_schedules (id, name, task_type, payload_json, interval_minutes, enabled, next_run_at)
VALUES ('sched-metadata-enrich', 'Auto-enrich Books Metadata', 'scan_metadata_enrich', NULL, 1440, 0, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO NOTHING;

