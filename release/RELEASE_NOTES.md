# NovelHub v1.0.5

NovelHub **v1.0.5** brings mobile TTS page flipping fixes, enhanced tooltip behavior across touch and desktop interfaces, instant branding asset cache-busting, and web reader visual refinements.

---

## What's New in v1.0.5

### Mobile TTS Page Flipping & Word Boundary Stability

- **Eliminated Page Bounce**: Fixed an issue on mobile devices where advancing TTS playback across line wraps or word boundaries would abruptly jump to the previous page before snapping back to the current position.
- **Accurate Subpixel Paging**: Paged reader modes (single and double column) now use enhanced subpixel range boundary detection, keeping pages stable and continuous during speech synthesis and reading tracking.

### Tooltip Behavior & Touch Screen Optimization

- **Smart Auto-Dismissal**: Tooltips now automatically hide whenever a button is clicked, an active element is pressed, or when opening drawers and modal dialogs.
- **Mobile Touch Friendly**: Suppressed persistent hover tooltips on touchscreens and mobile devices, preventing tooltips from sticking to the screen or obscuring navigation icons and toolbar controls.
- **Reader Controls Polish**: Seamlessly dismisses tooltips when toggling reader settings panels, table of contents, and toolbar menus.

### Instant Branding & Logo Updates

- **Automatic Cache-Busting**: Uploading a new site logo or favicon now generates versioned asset paths on the server and cleans up outdated files, ensuring immediate updates across browsers and CDN caches without manual cache purging.
- **Fresh Public Settings**: Fixed stale branding and site metadata persistence in browser local storage so configuration changes apply instantly upon page load.

### Reader UI & Visual Refinements

- **Enhanced Highlights**: Improved rendering for active TTS word tracking, search result matches, and reader selections.
- **Toolbar & Navigation**: Smoother transition states for sidebars, full-screen mode toggles, and reader action buttons.

---

## How to Update

### Docker Compose Update

To upgrade your existing NovelHub instance to **v1.0.5**, run the following commands in the directory containing your `docker-compose.yml`:

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

1. Download the new executable (`v1.0.5`) matching your OS/Arch from the **Assets** section below.
2. Stop the running NovelHub service/process.
3. Replace the executable with the new binary.
4. Restart NovelHub (database migrations apply automatically).

---

## Quick Start & Installation (New Deployments)

### Docker Compose

```yaml
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
