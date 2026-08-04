-- Kobo devices cannot send an Authorization header: the reader only lets you configure a
-- single api_endpoint URL, and it authenticates by possessing that URL. calibre-web solves
-- this by putting a random token in the path (/kobo/<token>/v1/...), which is the scheme
-- mirrored here -- see cps/kobo_auth.py.
--
-- Consequences worth stating plainly, because they are unusual for this codebase:
--   * the token IS the credential; anyone with the URL can read that user's library
--   * it therefore lands in reverse-proxy access logs like any other path
--   * it is not bound to a device and does not expire on its own
-- Hence it is stored separately from the JWT machinery, is revocable on its own, and never
-- grants anything beyond the Kobo endpoints.
CREATE TABLE IF NOT EXISTS kobo_auth_tokens (
    token TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME
);

-- One active token per user: regenerating replaces the old one, which is how a user revokes
-- a lost device.
CREATE UNIQUE INDEX IF NOT EXISTS idx_kobo_auth_tokens_user ON kobo_auth_tokens(user_id);

-- Which books each device has already been told about. Kobo's sync protocol distinguishes
-- "NewEntitlement" from "ChangedEntitlement", and calibre-web decides between them using a
-- per-user record of what has been synced rather than timestamps alone -- a book restored
-- from backup keeps its old created_at but is still new to the device.
CREATE TABLE IF NOT EXISTS kobo_synced_books (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    synced_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, book_id)
);
