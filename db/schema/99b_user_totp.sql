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
