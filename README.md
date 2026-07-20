# 📚 NovelHub

Self-hosted, local-first digital book library manager. Organize, read, and manage your entire digital book collection — EPUB, MOBI, PDF, DOCX, FB2, and more — all from a single, sleek web interface.

---

## ✨ Features

### 📖 Multi-Format Book Reader
- **15+ formats supported**: EPUB, MOBI, AZW/AZW3, PDF, DOCX, DOC, RTF, ODT, FB2, CBZ/CBR/CB7 (comics), plain text (TXT, MD), HTML — all rendered natively in the browser.
- **Three reading modes**: Scroll, Single Page, and Double Page layouts.
- **Six reader themes**: Light, Sepia, Warm, Dark, Dim, and Coffee — each carefully tuned for comfortable reading.
- **Customizable typography**: Font family, font size, line height, and content width controls.
- **Chapter-based navigation**: Table of contents sidebar, previous/next chapter buttons, and paginated page navigation.

### 🗂️ Library Management
- **Automatic scanning**: Drop files into your library folder — NovelHub detects and indexes them automatically.
- **Cover extraction**: Pulls cover images from EPUB, MOBI, FB2, and comic archives.
- **Metadata editing**: Edit title, author, description, series, tags, and more from the UI.
- **Smart filtering**: Filter by format, author, and category with chip-based navigation.
- **Duplicate detection**: SHA-256 fingerprinting identifies identical files to reclaim disk space.
- **Full-text search (FTS5)**: Search across the actual content of your books, not just titles and metadata.
- **Reading activity tracking**: Records reading progress and history per book.

### 🔐 Authentication & Admin
- **JWT-based auth**: Email/password login with access + refresh token rotation.
- **Role-based access control**: Admin, Moderator, and User roles with configurable permissions.
- **Admin dashboard**: Manage books, users, and roles from a dedicated admin panel.
- **User profiles**: Per-user settings, avatar upload, and password management.

### 🌐 Internationalization
- **Multi-language UI**: Built-in i18n support with language detection and switcher.

### 🏗️ Architecture Highlights
- **Single binary deployment**: The Go backend embeds the built React frontend — one process serves everything.
- **SQLite database**: Fully local, zero-config, easy to backup.
- **Background job queue**: Custom bounded worker pool for async parsing, indexing, and maintenance.
- **Auto-migration**: Database schema applied on startup, no manual migration steps.
- **Auto-maintenance**: Periodically purges database records for books deleted from disk.

---

## 🚀 Quick Start

### Prerequisites

- **Go** 1.22+
- **Node.js** 18+ and **npm**
- **SQLite** (bundled via modernc.org/sqlite, no CGO needed)

### Run Locally

```bash
# Clone the repository
git clone https://github.com/youruser/novelhub.git
cd novelhub

# Copy environment config
cp ".env copy.example" .env

# Build frontend + start the server
make run
```

The API starts at **http://127.0.0.1:3434** with the embedded web UI.

### Default Admin Account

```
Email:    admin@novelhub.local
Password: Admin@123456
```

### Docker

```bash
docker compose up -d
```

Mount your book library into `/libraries`:

```yaml
volumes:
  - ./your-books:/libraries
```

### Health Check

```bash
curl http://127.0.0.1:3434/api/v1/health
```

---

## 🛠️ Development

### Frontend (React + Vite + TailwindCSS + DaisyUI)

```bash
cd web
npm install
npm run dev    # Dev server with HMR at localhost:5173
```

### Backend (Go + Fiber + SQLite)

```bash
go run ./cmd/api
```

### Available Make Targets

| Command         | Description                             |
|-----------------|-----------------------------------------|
| `make run`      | Build frontend + start Go server        |
| `make web-dev`  | Start frontend dev server with HMR      |
| `make web-build`| Build frontend for production           |
| `make test`     | Run all Go tests                        |
| `make sqlc`     | Regenerate SQLC type-safe query models  |
| `make build`    | Build the Go binary                     |
| `make check`    | Run sqlc + tests + web-build + go build |
| `make docker-up`| Start Docker containers                 |
| `make docker-down` | Stop Docker containers               |

---

## 📁 Project Structure

```
novelhub/
├── cmd/
│   ├── api/            # Main application entry point + embedded frontend dist
│   └── parsecheck/     # CLI tool for testing book parsers
├── internal/
│   ├── controllers/    # HTTP handlers (auth, book, reader, library, admin, etc.)
│   ├── dtos/           # Request/response data transfer objects
│   ├── gen/            # SQLC-generated database code
│   ├── middlewares/    # JWT auth, RBAC, logging, compression
│   ├── models/         # Domain models
│   ├── repositories/   # Database access layer
│   ├── routes/         # Fiber route definitions
│   └── services/       # Business logic (book, auth, library, reader, jobs, etc.)
├── pkg/
│   ├── bookparser/     # Multi-format book parsing engine
│   │   ├── epub/       #   EPUB parser
│   │   ├── mobi/       #   MOBI/AZW parser (PalmDOC decompression)
│   │   ├── pdf/        #   PDF renderer (native browser viewer)
│   │   ├── fb2/        #   FictionBook 2 parser
│   │   ├── docx/       #   DOCX parser
│   │   ├── doc/        #   Legacy DOC parser (OLE2/CFBF)
│   │   ├── rtf/        #   RTF parser
│   │   ├── odt/        #   OpenDocument Text parser
│   │   ├── comic/      #   CBZ/CBR/CB7 comic archive parser
│   │   ├── htmlfile/   #   HTML file parser
│   │   ├── plain/      #   Plain text / Markdown parser
│   │   ├── archivebook/#   Generic archive-based book parser
│   │   └── external/   #   External format utilities
│   ├── cache/          # In-memory caching
│   ├── config/         # Environment configuration
│   ├── convert/        # Data conversion utilities
│   ├── database/       # SQLite connection management
│   ├── localfs/        # Local filesystem operations
│   ├── validator/      # Input validation
│   └── worker/         # Background job queue (bounded worker pool)
├── db/
│   ├── schema/         # SQLite migration files
│   ├── queries/        # SQLC query definitions
│   └── query/          # Additional query files
├── web/                # React frontend (Vite + TailwindCSS + DaisyUI)
│   └── src/
│       ├── components/ # Reusable UI components
│       ├── pages/      # Page-level components (Library, Reader, Admin, etc.)
│       ├── services/   # API client services
│       ├── stores/     # Zustand state management
│       ├── config/     # Frontend configuration
│       └── types/      # TypeScript type definitions
├── Makefile
├── docker-compose.yml
├── sqlc.yaml
└── go.mod
```

---

## 📄 Supported Formats

| Format | Extensions | Reader | Metadata | Cover |
|--------|-----------|--------|----------|-------|
| EPUB | `.epub`, `.kepub.epub` | ✅ HTML | ✅ | ✅ |
| MOBI/AZW | `.mobi`, `.azw`, `.azw3` | ✅ HTML | ✅ | ✅ |
| PDF | `.pdf` | ✅ Native | ⚠️ Basic | ❌ |
| DOCX | `.docx` | ✅ HTML | ✅ | ❌ |
| DOC | `.doc` | ✅ HTML | ✅ | ❌ |
| RTF | `.rtf` | ✅ HTML | ⚠️ Basic | ❌ |
| ODT | `.odt` | ✅ HTML | ✅ | ❌ |
| FB2 | `.fb2` | ✅ HTML | ✅ | ✅ |
| Comics | `.cbz`, `.cbr`, `.cb7` | ✅ Images | ⚠️ Basic | ✅ |
| HTML | `.html`, `.htm` | ✅ HTML | ⚠️ Basic | ❌ |
| Plain Text | `.txt` | ✅ HTML | ⚠️ Basic | ❌ |
| Markdown | `.md` | ✅ HTML | ⚠️ Basic | ❌ |

---

## ⚙️ Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_HOST` | `127.0.0.1` | Server bind address |
| `SERVER_PORT` | `3434` | Server port |
| `DATA_DIR` | `./data` | Data storage directory |
| `SQLITE_DB_PATH` | `./data/novelhub.db` | SQLite database path |
| `JWT_SECRET` | — | Access token signing secret |
| `JWT_REFRESH_SECRET` | — | Refresh token signing secret |
| `ADMIN_EMAIL` | `admin@novelhub.local` | Default admin email |
| `ADMIN_PASSWORD` | `Admin@123456` | Default admin password |
| `FIBER_BODY_LIMIT` | `1073741824` | Max upload size (1 GB) |
| `COOKIE_SECURE` | `false` | Secure cookie flag |
| `DISABLE_REQUEST_LOG` | `false` | Disable request logging |
| `DISABLE_RESPONSE_COMPRESSION` | `false` | Disable gzip/brotli |
| `ENABLE_PREFORK` | `false` | Enable Fiber prefork mode |

---

## 🧪 Testing

```bash
# Run all tests
make test

# Run specific package tests
go test ./pkg/bookparser/mobi/ -v
go test ./internal/services/ -v

# Run with coverage
go test ./... -cover
```

---

## 🔧 Tech Stack

| Layer | Technology |
|-------|-----------|
| **Backend** | Go, Fiber v3, SQLite (modernc.org) |
| **Frontend** | React 18, Vite, TypeScript, TailwindCSS, DaisyUI |
| **State** | Zustand |
| **Database** | SQLite with FTS5, SQLC for type-safe queries |
| **Auth** | JWT (access + refresh), bcrypt |
| **Serialization** | Sonic (bytedance high-perf JSON) |
| **i18n** | i18next + react-i18next |
| **Validation** | go-playground/validator |
| **Logging** | Zerolog |

---

## 📜 License

This project is for personal use. All rights reserved.
