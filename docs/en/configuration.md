# Configuration

NovelHub splits configuration in two. Environment variables cover what has to be
known before the database opens — or what protects the database and therefore
cannot live inside it. Everything else is in the admin UI and applies without a
restart.

Four variables matter. The rest have working defaults and most auto-tune to the
machine.

---

## Required

Copy `.env.example` to `.env` and fill in three secrets. The server will not
issue tokens without them.

```bash
cp .env.example .env
openssl rand -hex 32   # run three times, one value each
```

| Variable | Purpose |
|---|---|
| `JWT_SECRET` | Signs access tokens |
| `JWT_REFRESH_SECRET` | Signs refresh tokens |
| `DB_ENCRYPTION_KEY` | Encrypts third-party tokens (AniList, MAL) and the SMTP password stored in the database |

Use a different random value for each.

**Changing them later.** Changing either JWT secret signs everyone out — no data
is lost. Changing `DB_ENCRYPTION_KEY` makes already-encrypted tracker tokens and
the saved SMTP password permanently unreadable; users have to reconnect those
accounts and an admin has to re-enter the SMTP password. Back up your database
before touching it.

These stay environment variables because they sign and encrypt what is *in* the
database. Storing them in the database would be circular.

---

## Reverse proxy

```bash
TRUST_PROXY=false
```

This decides whether NovelHub believes two headers a proxy sends:

- `X-Forwarded-For` — who the client really is, used for rate limiting
- `X-Forwarded-Proto` — whether the original request was HTTPS, which decides
  whether the login cookie gets the `Secure` flag

| Value | Use when | Trusts |
|---|---|---|
| `false` | Browsers connect straight to NovelHub | Nothing |
| `true` | nginx/Caddy on the same host, or another container on the same Docker network | Loopback, private and link-local addresses: `127.0.0.0/8`, `::1`, `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `fc00::/7`, `169.254.0.0/16` |
| `1.2.3.0/24,5.6.7.8` | The proxy is on a public address — most often Cloudflare | Exactly the listed IPs and CIDRs |

Do not set `true` without a real proxy in front. Any client could then send those
headers itself: a fresh rate-limit bucket per request (defeating the sign-in
limiter entirely) and a forged `https` claim that puts `Secure` on cookies served
over plain HTTP, which browsers silently discard.

Setting it is only half the job — your proxy has to actually send the headers.
See [Reverse Proxy](reverse-proxy.md) for per-proxy configuration and how to
verify it.

`TRUST_PROXY` is read once at startup and cannot be an admin setting: it decides
whether the login cookie gets `Secure`, so a wrong value in the database would
mean signing in to fix the thing that stops you signing in.

---

## Optional

Everything below is commented out in `.env.example`. Uncomment only what you
need to override.

### Network

| Variable | Default | Notes |
|---|---|---|
| `SERVER_HOST` | `127.0.0.1` | Docker sets `0.0.0.0`; do not override it there or the published port is unreachable |
| `SERVER_PORT` | `3434` | |

### Storage

| Variable | Default | Notes |
|---|---|---|
| `DATA_DIR` | `./data` | Root for everything below |
| `SQLITE_DB_PATH` | `$DATA_DIR/novelhub.db` | |
| `CALIBRE_IMPORT_DIR` | `$DATA_DIR/calibre` | Only directories under this root can be imported. Point it at your Calibre library if it lives elsewhere. |

`DATA_DIR` contains:

```
data/
├── novelhub.db      SQLite database
├── books/           imported books and covers
├── calibre/         Calibre libraries available for import
├── inbox/           drop files here for automatic import
├── uploads/         in-progress chunked uploads
├── public/          uploaded site logo and favicon
├── logs/            rotating application logs
└── backups/         database backups
```

Back up `DATA_DIR` and it is the whole installation.

### Performance

All of these auto-tune. Set them only to cap resource use deliberately.

| Variable | Default |
|---|---|
| `SQLITE_CACHE_SIZE_KB` | Sized from system memory (64 MB–512 MB) |
| `SQLITE_MMAP_SIZE_BYTES` | Sized from system memory (256 MB–2 GB) |
| `SQLITE_MAX_OPEN_CONNS` | CPU count × 2, clamped to 4–16 |
| `SQLITE_MAX_IDLE_CONNS` | Same as max open |
| `CACHE_MAX_COST_BYTES` | Sized from system memory |
| `ASSET_CACHE_MAX_COST_BYTES` | System memory ÷ 32, clamped to 32 MB–512 MB — comic pages and covers, held as raw bytes in their own budget so they cannot evict book records |
| `JOB_WORKERS` | `1` — background job concurrency |
| `GOGC` | `200` — Go GC target; lower trades CPU for memory |
| `FIBER_CONCURRENCY` | Fiber default |
| `FIBER_READ_BUFFER_SIZE` | Fiber default |
| `FIBER_WRITE_BUFFER_SIZE` | Fiber default |

### Logging

| Variable | Default | Notes |
|---|---|---|
| `LOG_MAX_SIZE_MB` | `10` | Size at which the active log rotates |
| `LOG_MAX_FILES` | `5` | Rotated files kept |
| `DISABLE_REQUEST_LOG` | `true` | Turn off per-request logging for throughput |
| `DISABLE_STARTUP_MESSAGE` | `false` | |

These stay environment variables because logging starts before the database
opens — a database failure has to be loggable.

### Behaviour

| Variable | Default | Notes |
|---|---|---|
| `TOKEN_VERSION_CACHE` | `true` | Set `false` when several instances share one database, where an in-memory cache would go stale |
| `DISABLE_RESPONSE_COMPRESSION` | `false` | |
| `ENABLE_PREFORK` | `false` | Multi-process workers. Disables the token-version cache |
| `RESTORE_AUTO_RESTART` | `false` | Exit after staging a database restore so Docker or systemd restarts and applies it. Docker sets `true` |

---

## Configured in the admin UI

Not environment variables. Set during the setup wizard on first launch, then
under **Admin → Settings**. Changes apply immediately.

| Area | Covers |
|---|---|
| Site | Title, description, logo, favicon, sidebar items, home sections |
| Server URL | Absolute base URL used in OPDS catalog and Kobo sync links. Empty means detect it from each request — set it only if the detected host is wrong, for example behind a path-rewriting proxy |
| Access | Registration on/off, sign-in required, guest access mode, per-library guest visibility |
| Permissions | Per-role control of all 37 permissions — reading, personal features, library content, integrations, administration |
| Email (SMTP) | Host, port, username, password, sender address, TLS mode, max attachment size (MB, default 50MB), private-network dialling, plus a connection test. Also whether email verification and password reset are enabled |
| Reader features | In-book deep search, custom font upload, which cover engagement stats are visible |
| Trackers | AniList / MyAnimeList sync on or off |
| Upload limits | Chunk size, chunk count, concurrent sessions, total size, session TTL, cover and site asset size |
| Rate limits | Sign-in and OPDS attempts per window, and the window length |

### Rate limits

NovelHub rate-limits exactly two things, both guarded by the same pair of
settings: **sign-in** (`/api/v1/auth/*`) and **OPDS** (`/api/opds/*`).

Both run bcrypt password verification, which costs roughly 50–100 ms of CPU per
attempt — about 600× the cost of everything else on the request. That is the
resource worth protecting. OPDS is included because it uses HTTP Basic auth,
which carries no session, so bcrypt runs on *every* request.

Default: 5 attempts per 60 seconds, keyed by client IP.

For OPDS, only failed attempts count. A reader app polling the catalog with
valid credentials is normal traffic and is never throttled.

There is deliberately no general API rate limit. A comic chapter renders as one
image request per page, so opening a 200-page volume legitimately fires 200
requests — a general limit would throttle readers, not attackers.

---

### OPDS 1.2 & 2.0 Server

NovelHub includes full OPDS 1.2 (Atom XML) and OPDS 2.0 (JSON) catalog servers:

- **OPDS 1.2 Catalog**: `/api/opds/v1` (Atom XML format, compatible with KOReader, Moon+ Reader, Calibre, PocketBook, Aldiko). Includes navigation feeds (`/recent`, `/authors`, `/series`, `/tags`), OpenSearch XML (`/api/opds/v1/opensearch.xml`), and full-text search (`/api/opds/v1/search?q={searchTerms}`).
- **OPDS 2.0 Catalog**: `/api/opds/v2/catalog` (JSON format `application/opds+json`, compatible with modern readers like Thorium). Includes root navigation links, publication metadata, cover image links, and acquisition links.
- **Authentication**: Supports HTTP Basic Auth (using user account email and password) as well as Guest Access policies configured per library in Admin Settings.

### PWA & Offline Reading

NovelHub is a full Progressive Web App (PWA) with native installability:

- **Offline Engine**: Users can save complete books, chapters, and embedded images directly into browser IndexedDB storage for 100% offline reading without active server connection.
- **Service Worker & Updates**: Powered by `vite-plugin-pwa` and `workbox` with automatic update notification banners and storage quota monitoring.
- **Permissions**: Offline book saving is controlled per-role via the `book.offline` permission.

### Read Lists & `.cbl` Import

A collection answers "which group is this book in". A read list answers "which
book comes next": every entry carries an explicit position, so the order is
yours instead of the order the files were imported in.

- **Per-user**: read lists are private to the account that created them, like collections, and are gated by the same `book.collection` permission.
- **Reordering**: drag an entry or use the up/down buttons on `/read-lists`. The whole order is saved in one request.
- **Read in order**: opens the first entry with `?readlist=<id>`. At the end of the last chapter the reader's existing next button carries over to the next book in the list instead of stopping. Archived books are skipped. The position is not remembered — **Read in order** always starts at the first entry.
- **`.cbl` import**: upload a ComicRack reading list (max 8 MB). Document order *is* the reading order; nothing is re-sorted. Entries match on series name (case-insensitive) plus issue number, where `01`, `1` and `1.0` count as the same number. `Year` and `Volume` are ignored, since books carry no year column. Entries with no match in the library come back in an import report with their series and number; when two books share a series and number, the first one found wins.
- **Endpoints** (all under `/api/v1/read-lists`): `GET /`, `POST /`, `POST /import`, `GET|PUT|DELETE /:id`, `GET|POST /:id/books`, `DELETE /:id/books/:bookId`, `PUT /:id/order`, `GET /:id/next`.

---

## Authentication cookies

Nothing to configure; both properties are derived per request.

**`Secure`** is set whenever the request arrived over HTTPS — directly, or via a
trusted proxy's `X-Forwarded-Proto` (see `TRUST_PROXY` above). Over plain HTTP it
is omitted, because a browser silently discards a `Secure` cookie on an insecure
connection, which presents as a wrong password.

**`Domain`** is never set, so the cookie stays scoped to the host that served it.

**`SameSite`** is `Lax` and should stay there. Loosening it to `None` hands an
attacker's page the ability to send your cookies.

**`csrf_token`** is a third cookie, readable by JavaScript on purpose. The
frontend copies it into an `X-CSRF-Token` header and the server compares the two
on every POST/PUT/PATCH/DELETE. Requests carrying an `Authorization` header, and
the `/kobo/`, `/komga/`, `/api/opds/` and `/api/v1/sync/` prefixes, are exempt —
they authenticate per request and send no cookie, so there is nothing to forge.

---

## Verifying

```bash
# Server is up
curl http://127.0.0.1:3434/api/v1/health

# Rate limit engages (expect 429 within the first handful)
for i in $(seq 10); do
  curl -s -o /dev/null -w "%{http_code}\n" \
    -X POST http://127.0.0.1:3434/api/v1/auth/signin \
    -H 'Content-Type: application/json' \
    -d '{"email":"nobody@example.com","password":"wrong"}'
done
```

Behind a proxy, sign in over HTTPS and check **DevTools → Application → Cookies**.
The `access_token` row must show `Secure`. If it does not, the proxy is not
sending `X-Forwarded-Proto` or `TRUST_PROXY` does not cover its address.
