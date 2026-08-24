# NovelHub v1.0.0 — Initial Release 🚀

Welcome to the first official release of **NovelHub** — a self-hosted, local-first digital library manager and reader for eBooks, Audiobooks, Comics/Manga, and Podcasts, designed with a modern, high-performance web interface.

---

## 🌟 Key Features & Highlights

### 📖 Universal Multi-Format Reading Experience

- **20+ Supported Formats**: Read EPUB, KePub, MOBI, AZW, AZW3, PDF, DOCX, DOC, ODT, RTF, FB2, FBZ, Comic (CBZ, CBR, CBT, CB7), TXT, Markdown, HTML, Slide Decks (PPTX, PPT, ODP), Spreadsheets (XLSX, XLS, ODS, CSV, TSV), and LaTeX documents (`.tex`) natively.
- **High-Fidelity Typography & DTP Layout**: Preserves original publisher styling (bold, italic, underline, small-caps), custom alignment (Left, Center, Right, Justify, or Original CSS), responsive tables, and embedded audio/video media.
- **Dedicated Comic & Manga Reader**:
  - Single-page and double-page spread layouts.
  - Continuous vertical scroll mode (Webtoon).
  - Right-to-left (RTL) mode optimized for Japanese Manga.
  - Image bookmarking and visual thumbnail navigation.
- **Built-in Pure-Go eBook Converter**: Seamlessly convert between EPUB, MOBI, KePub, PDF, DOCX, FB2, CBZ, and TXT formats without external dependencies.
- **Text-to-Speech (TTS)**: Built-in multi-voice reader with custom speed control and real-time active word highlighting.
- **PWA & Offline Reading**: Progressive Web App with browser-level IndexedDB storage allowing full offline reading support.
- **Highlights, Annotations & Bookmarks**: Save favorite quotes, apply multi-color text highlights, write personal notes, and export directly to Readwise or Markdown.
- **Theme & Font Customization**: Comprehensive reader themes (Light, Dark, Sepia, Coffee, Dim, E-Ink) with support for custom uploaded fonts and user CSS overrides.

---

### 🎧 Audiobooks & Podcast Management

- **Native Audio Streaming**: Smooth, direct streaming for MP3, M4A, M4B, and FLAC files without requiring FFmpeg or HLS transcoding.
- **Audiobook Merger**: Combine multiple audio tracks into a single, chaptered M4B audiobook file.
- **Automatic Chapter Lookup (Audnexus ASIN)**: Automatically retrieve and populate audiobook chapter metadata via the Audnexus API.
- **Podcast Manager**: Subscribe to RSS XML feeds, automatically sync new episodes, and save podcast episodes directly to your library.

---

### 📱 Integrations & Cross-Device Sync

- **Kobo Wi-Fi Sync**: Wirelessly sync your library, bookmarks, reading progress, and on-the-fly KePub conversion directly with Kobo e-readers.
- **Mihon / Tachiyomi Sync**: Built-in Komga-compatible REST API for seamless comic reading progress synchronization.
- **VBook (Android) Integration**: Built-in Plugin Registry API and downloadable ZIP package for the VBook reader app on Android.
- **OPDS 1.2 & 2.0 Server**: Server-wide OPDS catalogs compatible with KOReader, Moon+ Reader, Thorium Reader, and more.
- **Two-Way Tracker Scrobbling**: Automatically sync reading progress with AniList, MyAnimeList, and Hardcover.app.
- **Send-to-Kindle**: Deliver eBooks directly to Kindle devices via built-in secure SMTP with attachment size control.

---

### 🔍 Discovery & Library Organization

- **Advanced Faceted Search**: Multi-facet filtering by authors, publishers, languages, formats, tags, series, and ratings.
- **Deep Full-Text Search**: Unicode SQLite FTS5 search across all chapter contents of all books in your library.
- **Smart Collections & Read Lists**: Dynamic rule-based shelves pinned to your home or sidebar, plus sequential reading lists with ComicRack (`.cbl`) import support.
- **Fuzzy Duplicate Management**: Automatic duplicate detection powered by Jaro-Winkler and Dice algorithms with a file merge tool.
- **Calibre Library Import**: Seamlessly import metadata and books from existing Calibre `metadata.db` databases.

---

### 📊 Reading Analytics & Engagement

- **Reading Analytics Dashboard**: Personal reading heatmaps, session duration tracking, daily/annual goal tracking, and library breakdown stats.
- **Estimated Reading Time (ETA)**: Accurate reading speed calculation and time-to-finish estimates per book.
- **Reviews & Quote Cards**: Write reviews, rate books with star ratings, and generate shareable visual quote cards.

---

### 🔒 Security & Access Control

- **OAuth2 / OIDC Single Sign-On**: Authenticate using external identity providers (Google, GitHub, Keycloak, Authelia, etc.).
- **PIN-Protected Kids Mode**: Restrict and hide mature books based on age ratings (G, PG, R, R18) behind a PIN code.
- **Magic Code Login**: 6-digit passwordless login designed specifically for E-Ink e-readers and smart devices.
- **Granular RBAC**: Comprehensive role-based access control with 39 granular permissions.
- **Hardened Security**: Socket-level SSRF prevention and filesystem path traversal protection via `localfs.SafeJoin`.

---

### ⚙️ Operations & Deployment

- **Single Binary Deployment**: React frontend and SQLite database engine embedded directly into a single executable — zero external runtime required.
- **Multi-Arch Docker Images**: Optimized container images supporting both `linux/amd64` and `linux/arm64`.
- **Zero-Downtime Backups**: SQLite Online Snapshot API ensures non-blocking, consistent live database backups.
- **First-Run Setup Wizard**: Intuitive administrator setup wizard and branding customizer on first launch.
- **16 UI Languages (i18n)**: English, Vietnamese, Japanese, Korean, Simplified Chinese, Traditional Chinese, French, German, Spanish, Portuguese, Russian, Arabic, Hindi, Indonesian, Thai, and Italian.

---

## 📦 Quick Start & Installation

### 1. Docker Compose (Recommended)

Create a `docker-compose.yml` file:

```yaml
version: "3.8"

services:
  novelhub:
    image: azenkain/novel-hub:latest
    container_name: novelhub
    restart: unless-stopped
    ports:
      - "3434:3434"
    environment:
      - JWT_SECRET=replace_with_a_random_32_char_secret
      - JWT_REFRESH_SECRET=replace_with_a_random_32_char_secret
      - DB_ENCRYPTION_KEY=replace_with_a_random_32_char_secret
      - TRUST_PROXY=true
    volumes:
      - ./data:/data
```

Start the container:

```bash
docker compose up -d
```

Open `http://localhost:3434` in your browser to complete the setup wizard.

---

### 2. Standalone Native Binary

Download the executable matching your operating system and architecture from the **Assets** section below:

- **Windows**: `novelhub-windows-amd64.exe` or `novelhub-windows-arm64.exe`
- **Linux**: `novelhub-linux-amd64` or `novelhub-linux-arm64`
- **macOS**: `novelhub-darwin-amd64` (Intel) or `novelhub-darwin-arm64` (Apple Silicon M1/M2/M3/M4)

Run the binary:

```bash
./novelhub-linux-amd64
```

Then navigate to `http://localhost:3434`.

---

## 🛡️ Verification & Checksums

All release binaries are accompanied by a `checksums.txt` file containing SHA-256 hashes in the Assets section below.
