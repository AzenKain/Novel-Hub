# NovelHub v1.0.9

## What's New

### Security & RBAC

- **Owner Immunity**: Protected root owner account from deletion, ban, and role modification.
- **Session Protection**: Prevented admin token invalidation during role changes; immediate forced logout for banned accounts.

### Bulk User Operations

- **Bulk Actions**: Batch edit profiles, assign/remove roles, revoke sessions, send emails, and soft delete/restore users.
- **Profile Alignment**: Added age rating and safe-mode controls to user profile edit.

### Email System

- **Responsive Toolbar**: Clean 3-row layout for Markdown tools, template/language dropdowns, and preview tab.
- **UX Protections**: Fixed dropdown clipping on mobile and added unsaved change confirmation when switching templates/languages.

### UI/UX Fixes

- **Modal Standardization**: Replaced browser `alert()` and `confirm()` with DaisyUI modals; softened backdrop blur.
- **Review Toolbar**: Fixed search and filter overflow on mobile viewports.
- **Reading Journey Card**: Compacted vertical layout and enforced viewport bounds (`dvh`/`svh`) to prevent clipping on Android navigation bars.

### Localization

- **Translation Sync**: Corrected mistranslations and synchronized all keys across 16 supported languages.

---

## How to Update

### Docker Compose Update

To upgrade your existing NovelHub instance to **v1.0.9**, run the following commands in the directory containing your `docker-compose.yml`:

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

1. Download the new executable (`v1.0.9`) matching your OS/Arch from the **Assets** section.
2. Stop the running NovelHub service/process.
3. Replace the executable with the new binary.
4. Restart NovelHub (database migrations apply automatically).

---

## Verification & Checksums

All release binaries are accompanied by a `checksums.txt` file containing SHA-256 hashes in the Assets section.
