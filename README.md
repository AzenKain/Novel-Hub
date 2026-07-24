# NovelHub

Self-hosted, local-first digital book library manager. Organize, read, and manage your entire digital book collection — EPUB, MOBI, PDF, DOCX, FB2, and more — from a single, high-performance web interface.

---

## Features

- **Multi-Format Reader**: Native browser rendering for 15+ formats (EPUB, MOBI, AZW3, PDF, DOCX, FB2, CBZ/CBR, TXT, MD, HTML). Includes 6 reader themes, custom typography, text highlighting, notes, Unicode in-book search, and 3 page layouts.
- **Audiobook Support**: Native HTML5 audio streaming for MP3, M4A, M4B, and FLAC (no FFmpeg/HLS dependencies). Features smart chunked streaming to bypass reverse proxy size limits (e.g., Cloudflare).
- **Calibre Library Sync**: Complete import & sync from Calibre's `metadata.db` including full metadata (authors, series, publishers, languages, tags, ratings, identifiers, comments).
- **Cross-device Sync & Reading History**: Accurately tracks reading progress (CFI/scroll positions/audio timestamps) across multiple devices in real-time.
- **Granular Library & Role Permission Policies**: Multi-role security model with 16 granular permissions (`book.read`, `book.download`, `book.share`, `book.bookmark`, `book.collection`, `book.review.create`, `book.review.delete`, `book.manage`, `library.read`, `library.manage`, `role.manage`, `user.manage`, `setting.manage`, `admin.access`, `job.read`, `webhook.manage`). Includes a modern **Library Scope Selector** UI with mode toggles, searchable multi-select dropdowns, and interactive library chips.
- **Customizable Book Engagement Stats**: Displays 6 cover engagement metrics (Reads, Downloads, Bookmarks, Collections, Rating, Shares) with custom admin visibility selection.
- **Webhooks & Live Builder**: Webhook notifications dispatcher for Discord, Telegram, Slack, and generic HTTP endpoints with HMAC SHA-256 signatures, custom headers, and a live preview builder.
- **Job Scheduler & Queue Engine**: Background job queue (`pkg/worker`) integrated with cron-like job schedule management (`job_schedules`). Supports automated interval execution, manual job triggering, and instant status tracking with singleflight protection and Cache-by-IDs performance.
- **System Logs & Live Tail Viewer**: Real-time log viewer and log tailing engine (`/admin/system/logs`) with log level filtering (`info`, `warn`, `error`), keyword search, and automatic file rotation (`LOG_MAX_SIZE_MB`, `LOG_MAX_FILES`).
- **Database Backup & Staged Restore**: Zero-downtime online database backup system using SQLite Online Backup API. Includes SHA-256 integrity checksum verification, database health check validation, isolated staged restore workflow, and optional process supervisor auto-restart (`RESTORE_AUTO_RESTART`).
- **Duplicate Detection & Management**: SHA-256 file hashing engine to detect and clean up duplicate book files across libraries.
- **High Performance Architecture**: Powered by `theine-go` in-memory RAM caching (Cache-by-IDs pattern), `singleflight` for thundering-herd protection, and 14 composite SQLite performance indexes for sub-millisecond query execution.
- **Path Traversal & SSRF Hardened**: Robust security engine featuring socket-level SSRF prevention via `pkg/netx` and `filepath.Rel` path traversal prevention in `pkg/localfs`.
- **Library Management**: Automatic file scanning, cover extraction, metadata editing, tag/author filtering, and SQLite FTS5 full-text search.
- **Chunked Uploads & Smart GC**: Robust upload system for massive files (bypassing Cloudflare's 100MB body limit) with smart background garbage collection for orphaned uploads using `pkg/worker`.
- **OPDS Server**: Built-in OPDS catalog (with basic auth and guest access controls) for seamless integration with mobile reader apps (Kobo, Moon+ Reader, KyBook).
- **Social Features**: Read and write reviews, rate books, and generate public share links.
- **First-Run Setup Wizard**: Intuitive setup wizard (`/setup`) for admin account creation, branding configuration (Logo & Favicon cropper/fetcher), sidebar navigation toggle, and default feature policy setup.
- **Security & RBAC**: JWT authentication with access + refresh token rotation and instant token version revocation. Socket-level SSRF protection via `pkg/netx`. Automatic **AES-256-GCM Envelope Encryption** via `pkg/crypto` (`DB_ENCRYPTION_KEY`) for sensitive tokens & credentials stored in SQLite.
- **Multi-Language Support**: i18n support with complete translation datasets in `web/public/locales/` (`en`, `vi`, `ja`, `ko`, `zh`).
- **Single Binary Deployment**: Embedded React frontend in Go binary. Zero-config SQLite (`modernc.org/sqlite`) with WAL mode, MMAP, and auto-migrations.

---

## Quick Start

### Run Locally

```bash
cp .env.example .env
make run
```

Access the UI at `http://127.0.0.1:3434`. On first launch, the Setup Wizard (`/setup`) will guide you through creating the root administrator account and configuring initial policies.

### Docker

```bash
docker compose up -d
```

---

## Development

```bash
# Frontend dev (port 5173 with Bun)
cd web && bun install && bun run dev

# Backend dev
go run ./cmd/api
```

### Make Targets

| Target | Description |
|---|---|
| `make run` | Build frontend with Bun and start Go server |
| `make test` | Run all Go unit tests |
| `make sqlc` | Regenerate SQLC database models |
| `make check` | Run full verification (sqlc + tests + web-build + go build) |

---

## Project Structure

```
novelhub/
├── .agents/            # AI agent workspace rules (AGENTS.md)
├── cmd/api/            # Application entry point & embedded web UI
├── internal/
│   ├── controllers/    # Fiber v3 HTTP handlers & DTO validation
│   ├── dtos/           # Request / Response payload definitions
│   ├── gen/sqlc/       # Type-safe generated database code
│   ├── middlewares/    # JWT auth, RBAC permissions, logger, compression
│   ├── models/         # Domain entities (FromSqlc / ToResponse helpers)
│   ├── repositories/   # Persistence layer with RAM Cache-by-IDs pattern
│   └── services/       # Domain business logic & multi-format parsers
├── pkg/
│   ├── apperrors/      # Application error definitions
│   ├── bookparser/     # Format parser engines (EPUB, MOBI, PDF, FB2, etc.)
│   ├── cache/          # In-memory RAM cache (`theine-go`) & key builders
│   ├── calibre/        # Calibre metadata.db import engine
│   ├── convert/        # ID parsing, null conversion & cursor helpers
│   ├── database/       # SQLite pragmas & transaction manager (`TxManager`)
│   ├── jsonx/          # High-performance JSON engine (sonic with std fallback)
│   ├── localfs/        # Path traversal prevention using SafeJoin & filepath.Rel
│   ├── netx/           # SSRF-safe HTTP client with IP validation
│   ├── validator/      # DTO struct validation engine
│   └── worker/         # Bounded background worker pool
├── db/
│   ├── schema/         # SQLite database migrations
│   └── query/          # Type-safe SQLC query definitions (explicit column projections)
└── web/                # React 19 + Vite + TailwindCSS + DaisyUI (Bun)
    ├── public/locales/ # i18n translation datasets (en, vi, ja, ko, zh)
    └── src/            # Components, Pages, Services, Stores (Zustand)
```

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `SERVER_HOST` | `127.0.0.1` | Server bind IP address |
| `SERVER_PORT` | `3434` | Server listening port |
| `FRONTEND_URL` | `http://your-domain.com` | Allowed frontend origin URL for CORS and auth redirects |
| `DATA_DIR` | `./data` | Root directory for uploaded books, covers, and media files |
| `SQLITE_DB_PATH` | `./data/novelhub.db` | SQLite database file path |
| `JWT_SECRET` | — | Required secret key for access token signing |
| `JWT_REFRESH_SECRET` | — | Required secret key for refresh token signing |
| `DB_ENCRYPTION_KEY` | — | Required 32-byte hex key for AES-256-GCM envelope encryption |
| `TOKEN_VERSION_CACHE` | `true` | Cache token versions in RAM for single-instance deployments |
| `SQLITE_CACHE_SIZE_KB` | `262144` | Total SQLite page-cache budget in KB (256 MB) |
| `SQLITE_MMAP_SIZE_BYTES` | `536870912` | Memory-mapped I/O ceiling in bytes (512 MB) |
| `COOKIE_SECURE` | `false` | Enable Secure flag on authentication cookies (requires HTTPS) |
| `COOKIE_DOMAIN` | — | Domain attribute for session cookies (leave empty for host-only) |
| `DISABLE_REQUEST_LOG` | `true` | Disable per-request console logging for maximum throughput |
| `DISABLE_RESPONSE_COMPRESSION` | `false` | Disable Fiber response gzip/brotli compression |
| `ENABLE_PREFORK` | `false` | Enable multi-process worker preforking across CPU cores |
| `DISABLE_STARTUP_MESSAGE` | `false` | Disable Fiber startup ASCII banner |
| `LOG_MAX_SIZE_MB` | `10` | Maximum size in MB of active log file before rolling rotation |
| `LOG_MAX_FILES` | `5` | Maximum number of rotated log backup files retained |
| `RESTORE_AUTO_RESTART` | `false` | Gracefully exit process after staging database restore for container supervisor restart |

Database restores are validated and staged first. With `RESTORE_AUTO_RESTART=true`, NovelHub exits gracefully so the process supervisor (Docker/systemd) can restart it and apply the staged restore before opening SQLite. Otherwise, restart NovelHub manually after the admin UI reports that the restore is ready.

---

## Supported Formats

| Format | Extensions | Reader | Metadata | Cover |
|---|---|---|---|---|
| EPUB / KePub | `.epub`, `.kepub.epub` | ✅ HTML | ✅ | ✅ |
| MOBI / AZW3 | `.mobi`, `.azw`, `.azw3` | ✅ HTML | ✅ | ✅ |
| Audiobooks | `.mp3`, `.m4a`, `.m4b`, `.flac` | ✅ Audio | ✅ | ❌ |
| PDF | `.pdf` | ✅ Native | ⚠️ Basic | ❌ |
| DOCX / DOC | `.docx`, `.doc` | ✅ HTML | ✅ | ❌ |
| FB2 | `.fb2` | ✅ HTML | ✅ | ✅ |
| Comics | `.cbz`, `.cbr`, `.cb7` | ✅ Images | ⚠️ Basic | ❌ |
| Plain Text / MD | `.txt`, `.md`, `.rtf`, `.odt` | ✅ HTML | ⚠️ Basic | ❌ |

---

## License

This project is for personal use. All rights reserved.
