# NovelHub v1.0.2 🚀

NovelHub **v1.0.2** brings major upgrades to the reader Text-to-Speech (TTS) engine, deep in-book full-text search navigation, full-screen modal overlays, and multiple UI/UX refinements across the reader sidebar, authentication modals, and user profile navigation.

---

## 🌟 What's New & Improved in v1.0.2

### 🔊 Text-to-Speech (TTS) Engine Upgrades
- **Universal Multilingual Sentence Segmentation**: Integrated Unicode Standard Annex #29 via `Intl.Segmenter(..., { granularity: 'sentence' })` with natural chunk merging (250–400 characters). Guarantees natural pauses and smooth playback across all 100+ global languages (Vietnamese, CJK, Latin, Arabic, Cyrillic, Indic, etc.).
- **Eliminated Word-Level Repetition/Stutter**: Added smart word-boundary resolution (`getNextWordOffset`) on pause and resume, preventing repeating already spoken words.
- **Exact 1-to-1 Spoken Word Highlighting**: Fixed character offset drift during TTS highlighting by strictly preserving raw DOM TreeWalker whitespace indices.

### 🔍 Reader In-Book Search & Deep-Linking
- **Accurate Cross-Chapter Navigation**: Resolved chapter ID mapping between client file references and database UUIDs. Clicking any search result instantly loads the correct chapter.
- **DOM Match Positioning & Auto-Scroll**: Reader automatically locates the target snippet phrase in the rendered DOM, smooth-scrolls to the exact sentence in both paged and vertical scroll layouts, and highlights the match in bright yellow with an outline (`::highlight(search-result-match)`).
- **Intelligent Highlight Auto-Dismiss**: The search match highlight automatically fades away after 2 seconds or dismisses immediately upon user click or keypress.
- **Search State Persistence**: Retains search keywords and query results across modal open/close actions.

### 🎨 UI/UX & Layout Refinements
- **Full-Viewport Modal Portaling**: Wrapped `OfflineWarningModal`, `ShareDialog`, and `SendToKindleModal` in React Portals (`createPortal(..., document.body)`), ensuring dark backdrops dim the entire screen without being constrained by page animations.
- **Navbar Layout & Modal Isolation**: Resolved duplicate login/register modal instances and flexbox layout disruption in the header navbar by centralizing authentication modals at the application root (`main.tsx`).
- **Reader Sidebar Dual Action Buttons**: Added side-by-side action buttons in Reader Sidebar footer (`Trang trước` / `Chi tiết sách` — Previous / Book Details), fully localized across all 16 supported languages.
- **Profile Page Tab Navigation**: Configured profile tab switching with history replacement (`{ replace: true }`), allowing the Back button to directly return to the previous library/book workspace.

---

## 🔄 How to Update

### 🐳 Docker Compose Update
To upgrade your existing NovelHub instance to **v1.0.2**, run the following commands in the directory containing your `docker-compose.yml`:

```bash
# 1. Pull the latest image
docker compose pull

# 2. Recreate the container with zero data loss
docker compose up -d

# 3. (Optional) Remove old dangling images
docker image prune -f
```

### 🐳 Docker CLI / Standalone Container Update
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

### 💻 Standalone Native Binary Update
1. Download the new executable (`v1.0.2`) matching your OS/Arch from the **Assets** section below.
2. Stop the running NovelHub service/process.
3. Replace the executable with the new binary.
4. Restart NovelHub (database migrations apply automatically).

---

## 📦 Quick Start & Installation (New Deployments)

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

## 🛡️ Verification & Checksums

All release binaries are accompanied by a `checksums.txt` file containing SHA-256 hashes in the Assets section below.
