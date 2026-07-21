# NovelHub

Self-hosted, local-first digital book library manager. Organize, read, and manage your entire digital book collection — EPUB, MOBI, PDF, DOCX, FB2, and more — from a single, high-performance web interface.

---

## Features

- **Multi-Format Reader**: Native browser rendering for 15+ formats (EPUB, MOBI, AZW3, PDF, DOCX, FB2, CBZ/CBR, TXT, MD, HTML). Includes 6 reader themes, custom typography, text highlighting, notes, and 3 page layouts.
- **Audiobook Support**: Native HTML5 audio streaming for MP3, M4A, M4B, and FLAC (no FFmpeg/HLS dependencies). Features smart chunked streaming to bypass reverse proxy size limits (e.g., Cloudflare).
- **Cross-device Sync & Scroll Tracking**: Accurately tracks reading progress (CFI/scroll positions/audio timestamps) across multiple devices in real-time.
- **High Performance Architecture**: Powered by `theine-go` in-memory RAM caching (Cache-by-IDs pattern) and `singleflight` for thundering-herd protection.
- **Library Management**: Automatic file scanning, cover extraction, metadata editing, tag/author filtering, duplicate detection (SHA-256), and SQLite FTS5 full-text search.
- **Chunked Uploads & Smart GC**: Robust upload system for massive files (bypassing Cloudflare's 100MB body limit) with smart background garbage collection for orphaned uploads using `pkg/worker`.
- **OPDS Server**: Built-in OPDS catalog (with basic auth and guest access controls) for seamless integration with mobile reader apps.
- **Social Features**: Read and write reviews, rate books, and generate public share links.
- **Security & RBAC**: JWT authentication with access + refresh token rotation and instant token version revocation. Socket-level SSRF and DNS Rebinding protection via `pkg/netx`.
- **Multi-Language Support**: i18n support with translation datasets in `web/public/locales/` (`en`, `vi`, `ja`, `ko`, `zh`).
- **Single Binary Deployment**: Embedded React frontend in Go binary. Zero-config SQLite (`modernc.org/sqlite`) with WAL mode, MMAP, and auto-migrations.

---

## Quick Start

### Run Locally

```bash
cp .env.example .env
make run
```

Access the UI at `http://127.0.0.1:3434`. Default admin credentials: `admin@novelhub.local` / `Admin@123456`.

### Docker

```bash
docker compose up -d
```

---

## Development

```bash
# Frontend dev (port 5173)
cd web && npm install && npm run dev

# Backend dev
go run ./cmd/api
```

### Make Targets

| Target | Description |
|---|---|
| `make run` | Build frontend and start Go server |
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
│   ├── gen/sqlc/        # Type-safe generated database code
│   ├── middlewares/    # JWT auth, RBAC permissions, logger, compression
│   ├── models/         # Domain entities (FromSqlc / ToResponse helpers)
│   ├── repositories/   # Persistence layer with RAM Cache-by-IDs pattern
│   └── services/       # Domain business logic & multi-format parsers
├── pkg/
│   ├── apperrors/      # Application error definitions
│   ├── bookparser/     # Format parser engines (EPUB, MOBI, PDF, FB2, etc.)
│   ├── cache/          # In-memory RAM cache (`theine-go`) & key builders
│   ├── convert/        # ID parsing, null conversion & cursor helpers
│   ├── database/       # SQLite pragmas & transaction manager (`TxManager`)
│   ├── jsonx/          # High-performance JSON engine (sonic with std fallback)
│   ├── netx/           # SSRF-safe HTTP client with IP validation
│   ├── validator/      # DTO struct validation engine
│   └── worker/         # Bounded background worker pool
├── db/
│   ├── schema/         # SQLite database migrations
│   └── query/          # Type-safe SQLC query definitions (explicit column projections)
└── web/                # React 18 + Vite + TailwindCSS + DaisyUI
    ├── public/locales/ # i18n translation datasets (en, vi, ja, ko, zh)
    └── src/            # Components, Pages, Services, Stores (Zustand)
```

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `SERVER_HOST` | `127.0.0.1` | Server bind IP |
| `SERVER_PORT` | `3434` | Server listening port |
| `SQLITE_DB_PATH` | `./data/novelhub.db` | SQLite database file path |
| `JWT_SECRET` | — | Access token signing secret |
| `JWT_REFRESH_SECRET` | — | Refresh token signing secret |
| `ADMIN_EMAIL` | `admin@novelhub.local` | Default admin email |
| `ADMIN_PASSWORD` | `Admin@123456` | Default admin password |
| `FIBER_BODY_LIMIT` | `1073741824` | Max request body limit (1 GB) |
| `GOGC` | `200` | GC percent tuning for high throughput |

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
