# NovelHub v1.0.7

NovelHub **v1.0.7** introduces a full OPDS 2.0 JSON Catalog implementation, Calibre Content Server API emulation, KOReader Kosync progress sync enhancements, RFC 4918 WebDAV streaming improvements, and guest OPDS acquisition permissions.

---

## What's New in v1.0.7

### Full OPDS 2.0 JSON Catalog Specification
- **OPDS 2.0 Protocol Implementation**: Full support for the modern OPDS 2.0 JSON format (`application/opds+json`) alongside classic OPDS 1.2 Atom XML feeds, fully compatible with Thorium Reader, Cantook, and modern e-readers.
- **Dedicated Feeds & Facets**: Complete catalog endpoints for recent additions, authors, series, tags/genres, publishers, and languages with dynamic filtering and offset pagination.
- **Direct Acquisition & Cover Resolution**: Dedicated download and cover endpoints for OPDS v1 and v2 with accurate MIME types and cache validation.

### Calibre Content Server API Emulation
- **Ecosystem Compatibility**: Implemented Calibre Content Server JSON/AJAX API endpoints (`/calibre`, `/calibre/ajax/library-info`) allowing Calibre Companion, Aldiko, and Moon+ Reader to query metadata and browse library contents natively.

### KOReader 2-Way Progress Sync (Kosync)
- **Real-Time Progress Synchronization**: Dedicated Kosync protocol endpoint (`/api/v1/sync/koreader`) for real-time two-way reading progress synchronization across KOReader devices and web readers.

### WebDAV Protocol & Streaming (RFC 4918)
- **Audio & Media Streaming**: Enhanced range request handling and HTTP Basic Auth verification for seamless streaming of large audiobooks (MP3, M4B, FLAC) and books.
- **Automated WebDAV Probe**: Added internal WebDAV compliance probe utility (`cmd/webdavprobe`) for connection testing and validation.

### RBAC & Guest Acquisition Permissions
- **Guest OPDS Download**: Granted `opds.download` permission to the `GUEST` role for public libraries via database migration `99b_guest_opds_download.sql`.
- **Query Cache Invalidation**: Optimized operations query invalidation keys to ensure cache freshness across background tasks and schedules.

---

## How to Update

### Docker Compose Update

To upgrade your existing NovelHub instance to **v1.0.7**, run the following commands in the directory containing your `docker-compose.yml`:

```bash
# 1. Pull the latest image
docker compose pull

# 2. Recreate the container with zero data loss
docker compose up -d

# 3. (Optional) Remove old dangling images
docker image prune -f
```

### Docker CLI / Standalone Container Update

```bash
# 1. Pull latest image
docker pull azenkain/novel-hub:latest

# 2. Stop and remove the old container
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
  -e TRUST_PROXY=true \
  -v $(pwd)/data:/data \
  azenkain/novel-hub:latest
```

### Standalone Native Binary Update

1. Download the new executable (`v1.0.7`) matching your OS/Arch from the **Assets** section.
2. Stop the running NovelHub service/process.
3. Replace the executable with the new binary.
4. Restart NovelHub (database migrations apply automatically).

---

## Verification & Checksums

All release binaries are accompanied by a `checksums.txt` file containing SHA-256 hashes in the Assets section.
