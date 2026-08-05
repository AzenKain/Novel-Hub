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
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Matches the keyset order (created_at DESC, id ASC) the list query pages on.
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC, id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs(actor_id, created_at DESC);
