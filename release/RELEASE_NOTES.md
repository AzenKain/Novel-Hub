# NovelHub v1.0.4

NovelHub **v1.0.4** introduces social reading achievement cards, a bulk book title cleaner, Calibre app connectivity, and standalone Anki deck export.

---

## What's New in v1.0.4

### Shareable Reading & Quote Cards

- **Achievement Cards**: Generate and share beautiful summary cards of your reading stats, reading streak, and activity heatmap directly from the Analytics page.
- **Quote Cards**: Highlight any memorable passage in a book and turn it into a styled quote card with book cover and author info.

### Bulk Book Title & Author Cleaner

- Clean up messy book titles in bulk from the Admin panel: strip tags like `[Light Novel]` or `(2024)`, auto-split author and title names, and preview changes before applying.

### Calibre App Integration

- Connect popular e-reader apps (like Calibre Companion or Aldiko) directly to your NovelHub library using Calibre's Content Server protocol.

### Direct Anki Deck Export (.apkg / .csv)

- Export your highlighted vocabulary and notes directly into ready-to-use `.apkg` deck files to study on Anki Desktop, AnkiMobile, or AnkiDroid without needing extra plugins.

### Performance & Stability

- Improved two-factor authentication (2FA) security, smoother background metadata fetching, and various stability improvements.

---

## How to Update

### Docker Compose Update

To upgrade your existing NovelHub instance to **v1.0.4**, run the following commands in the directory containing your `docker-compose.yml`:

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

1. Download the new executable (`v1.0.4`) matching your OS/Arch from the **Assets** section below.
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
