CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    full_name TEXT,
    avatar_url TEXT,
    password_hash TEXT,
    auth_provider TEXT NOT NULL DEFAULT 'LOCAL',
    is_deleted INTEGER NOT NULL DEFAULT 0,
    token_version INTEGER NOT NULL DEFAULT 1,
    refresh_token TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    CHECK (auth_provider IN ('LOCAL'))
);

CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_system INTEGER NOT NULL DEFAULT 0,
    is_admin INTEGER NOT NULL DEFAULT 0,
    is_banned INTEGER NOT NULL DEFAULT 0,
    auto_assign INTEGER NOT NULL DEFAULT 0,
    position INTEGER NOT NULL DEFAULT 0,
    is_deleted INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_users_provider_created_at ON users (auth_provider, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_active_created_at ON users (created_at DESC) WHERE is_deleted = 0;
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles (user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles (role_id);
CREATE INDEX IF NOT EXISTS idx_roles_active ON roles (name) WHERE is_deleted = 0;

CREATE TRIGGER IF NOT EXISTS trigger_users_updated_at
AFTER UPDATE ON users
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE users SET updated_at = datetime('now') WHERE id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS trigger_roles_updated_at
AFTER UPDATE ON roles
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE roles SET updated_at = datetime('now') WHERE id = OLD.id;
END;

-- Per-user TOTP (RFC 6238). Replaces the emailed login code: the secret lives on the user's
-- phone, so a broken SMTP server can no longer decide whether anyone can sign in.
-- The secret is stored with pkg/crypto EncryptAES like smtp.password -- a database copy or
-- backup must not hand over a working second factor.
CREATE TABLE IF NOT EXISTS user_totp (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    secret TEXT NOT NULL,
    confirmed_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Recovery codes are the only way back in when the phone is gone, so they are single-use and
-- stored hashed: a leaked table must not be a set of working bypasses.
CREATE TABLE IF NOT EXISTS user_totp_recovery_codes (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    used_at DATETIME,
    PRIMARY KEY (user_id, code_hash)
);

CREATE INDEX IF NOT EXISTS idx_user_totp_recovery_unused
    ON user_totp_recovery_codes(user_id) WHERE used_at IS NULL;

-- Trigram index for admin user search: LIKE '%term%' cannot use a B-tree index.
-- Same matches as LIKE ("miya" still finds "Omiya"), 160ms -> 0.4ms on selective terms.
-- ponytail: one concatenated column, not three. Multi-column MATCH needs the
-- `<table> MATCH` form, which sqlc rejects. Costs a bigger index (96MB/200k users).
CREATE VIRTUAL TABLE IF NOT EXISTS fts_users USING fts5(
    user_id UNINDEXED,
    haystack,
    tokenize="trigram"
);

-- Keep haystack identical in all three triggers and the backfill below.
CREATE TRIGGER IF NOT EXISTS t_users_fts_ai AFTER INSERT ON users BEGIN
  INSERT INTO fts_users(rowid, user_id, haystack)
  VALUES (new.rowid, new.id, new.id || ' ' || new.email || ' ' || COALESCE(new.full_name, ''));
END;

CREATE TRIGGER IF NOT EXISTS t_users_fts_ad AFTER DELETE ON users BEGIN
  DELETE FROM fts_users WHERE rowid = old.rowid;
END;

CREATE TRIGGER IF NOT EXISTS t_users_fts_au AFTER UPDATE ON users BEGIN
  UPDATE fts_users
  SET user_id = new.id,
      haystack = new.id || ' ' || new.email || ' ' || COALESCE(new.full_name, '')
  WHERE rowid = old.rowid;
END;

INSERT INTO fts_users(rowid, user_id, haystack)
SELECT u.rowid, u.id, u.id || ' ' || u.email || ' ' || COALESCE(u.full_name, '')
FROM users u
WHERE NOT EXISTS (SELECT 1 FROM fts_users f WHERE f.rowid = u.rowid);
