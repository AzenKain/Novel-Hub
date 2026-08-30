# NovelHub v1.0.3

NovelHub **v1.0.3** introduces the **Book Doctor EPUB Repair Engine**, **Native WebDAV Server**, **AnkiConnect Flashcard Sync**, and **Fluid Multi-Device Bookshelves** with drag-to-scroll.

---

## What's New in v1.0.3

### Book Doctor & EPUB Repair Engine
- **Structural Diagnostics**: Deep inspection of EPUB files (ZIP headers, mimetype, OPF, duplicate manifest IDs/hrefs, missing spine items, broken internal anchor links, unescaped XML entities, missing NCX/Nav TOC).
- **Automated Repair**: One-click repair to deduplicate manifests, sanitize broken internal links, repair XML namespaces, unescape URL hrefs, rebuild mimetypes, and upgrade EPUB 2.0 to EPUB 3.0.
- **Admin Tools & Background Maintenance**: Interactive Book Doctor modal in Admin Books table, and scheduled batch repair background job (`repair_books`) with SHA-256 hash recalculation and RAM cache purging.
- **Permission**: Controlled via `book.doctor` permission.

### Native WebDAV Server (`/webdav`)
- **Direct Storage Mount**: Mount libraries directly from macOS Finder, Windows File Explorer, Linux, and reader apps (KOReader, Moon+ Reader).
- **Authentication & Isolation**: HTTP Basic Auth with NovelHub account credentials, gated by `system.webdav` permission.

### AnkiConnect Integration
- **Flashcard Export**: Export reading highlights, vocabulary, and quotes directly to Anki decks via AnkiConnect (`http://127.0.0.1:8765`).
- **Custom Mapping**: Configurable decks, note models (Basic, Cloze), field mappings, and tags in User Profile.

### Fluid Bookshelf Layout & Pagination
- **Responsive Grid**: Proportional 2 to 6 column layout with larger cover art and 2-line title display.
- **Mouse & Touch Drag-to-Scroll**: Physics-based drag-to-scroll and horizontal mouse wheel scrolling on carousels.
- **Page-Aligned Batching**: Optimized fetch pagination ($60$ items) ensuring completely filled rows across all screen breakpoints.

---

## How to Update

### Docker Compose Update
To upgrade your existing NovelHub instance to **v1.0.3**, run the following commands in the directory containing your `docker-compose.yml`:

```bash
# 1. Pull the latest image
docker compose pull

# 2. Recreate the container with zero data loss
docker compose up -d

# 3. (Optional) Remove old dangling images
docker image prune -f
```

### Docker CLI / Standalone Container Update
If you are running NovelHub with standalone `docker run`:

```bash
# 1. Pull latest image
docker pull azenkain/novel-hub:latest

# 2. Stop and remove the old container (your data in ./data volume is safe)
docker stop novelhub
docker rm novelhub

# 3. Start the new container with your existing volume
docker run -d \
  --name novelhub \
  --restart unless-stopped \
  -p 3434:3434 \
  -e JWT_SECRET=your_jwt_secret \
  -e JWT_REFRESH_SECRET=your_jwt_refresh_secret \
  -e DB_ENCRYPTION_KEY=your_db_key \
  -v $(pwd)/data:/data \
  azenkain/novel-hub:latest
```

### Standalone Native Binary Update
1. Download the new executable (`v1.0.3`) matching your OS/Arch from the **Assets** section below.
2. Stop the running NovelHub service/process.
3. Replace the executable with the new binary.
4. Restart NovelHub (database migrations apply automatically).

---

## Quick Start & Installation (New Deployments)

### Docker Compose

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

Start NovelHub:
```bash
docker compose up -d
```

Open `http://localhost:3434` in your browser.

---

## Verification & Checksums

All release binaries are accompanied by a `checksums.txt` file containing SHA-256 hashes in the Assets section below.
