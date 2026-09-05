# NovelHub v1.0.8

NovelHub **v1.0.8** delivers an extensive mobile and tablet UI/UX overhaul across the library workspace and administration panels, optimizes content hierarchy for reading resumption, resolves touch gestures on book carousels, streamlines roles and permissions matrix interactions, and ensures 100% translation key parity across all 16 supported languages.

---

## What's New in v1.0.8

### Mobile & Tablet Library Workspace Redesign

- **Reading Resumption First**: Reordered homepage content hierarchy on mobile/tablet viewports (`< xl`) so that **Recently Read (`RecentlyReadPanel`)** appears directly beneath the welcome banner for instantaneous one-tap reading resumption.
- **Natural Reading Flow**: Restructured sections into a clean progression: Welcome & Quick Stats &rarr; Recently Read &rarr; Random & Top Hot Books &rarr; Reading Activity Heatmap &rarr; Full Book Catalog & Filter Chips at the bottom of the page.
- **Independent Desktop Layout**: Maintained the optimized two-column desktop layout (`≥ xl`) with sticky sidebar widgets and zero layout regressions.
- **Bidirectional Touch Navigation**: Enabled smooth dual-axis touch gestures (`touch-pan-y` and horizontal swipe) on `HorizontalBookShelf` and `SmartFilterShelf` carousels.

### Role Detail & Permissions Matrix Mobile Experience

- **Eliminated Vertical Text Wrapping**: Granted full-width flow to permission descriptions and key badges on narrow screens, eliminating awkward single-word wrapping and right-edge button clipping.
- **Dedicated Mobile Action Rows**: Structured permission effect toggles (`Allow` / `Deny`) and library scoping controls into an indented sub-tier with compact segmented controls and responsive labels.
- **Floating Save Action Bar**: Introduced a persistent bottom notification bar on mobile when permission modifications are detected (`hasChanges`), allowing administrators to save immediately without scrolling back to the top of long lists.
- **Scope Context Isolation**: Filtered out library scoping selectors from global system administrative permissions where library boundaries are inapplicable.

### Universal Navigation & Component Polish

- **Responsive Back Buttons**: Standardized back-to-library navigation buttons across `OfflineBooksPage`, `ReadListPage`, `PodcastsPage`, and `ReadingAnalyticsPage` with `btn-square sm:w-auto sm:px-3` and `whitespace-nowrap`, rendering an icon-only square button on mobile and an auto-expanding label on desktop.
- **Admin Header Synchronization**: Removed obsolete back buttons from `OAuthSettings` and aligned header typography and padding with other admin sub-pages.
- **Action Toolbar Consolidation**: Streamlined action controls on the admin Books management page (`Library`, `Calibre`, `Upload`) into a single unified row with responsive short labels.
- **Search Icon Layering**: Standardized `z-10 pointer-events-none` layering across all daisyUI search inputs to eliminate click-blocking and visual clipping.

### Complete Internationalization & 16-Language Synchronization

- **100% Translation Parity**: Verified and synchronized exactly 1,797 translation keys across all 16 locale dictionaries (`ar`, `de`, `en`, `es`, `fr`, `hi`, `id`, `it`, `ja`, `ko`, `pt`, `ru`, `th`, `vi`, `zh-CN`, `zh-TW`) with zero missing keys and zero empty values.
- **New Localization Keys**: Added localized strings for compact library scoping (`library_scope_short`, `scope_all_short`, `scope_specific_short`), admin effect indicators (`admin.role_effect_label`), and mobile unsaved changes notifications (`admin.unsaved_changes`).

---

## How to Update

### Docker Compose Update

To upgrade your existing NovelHub instance to **v1.0.8**, run the following commands in the directory containing your `docker-compose.yml`:

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

1. Download the new executable (`v1.0.8`) matching your OS/Arch from the **Assets** section.
2. Stop the running NovelHub service/process.
3. Replace the executable with the new binary.
4. Restart NovelHub (database migrations apply automatically).

---

## Verification & Checksums

All release binaries are accompanied by a `checksums.txt` file containing SHA-256 hashes in the Assets section.
