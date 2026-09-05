# NovelHub v1.0.6

NovelHub **v1.0.6** delivers multi-device login persistence with concurrent refresh token rotation, enhanced reverse-proxy and CSRF compatibility, unified brand typography and logo sizing across all pages, and reader experience enhancements.

---

## What's New in v1.0.6

### 🔄 Multi-Device Login Persistence & Concurrent Refresh Token Rotation

- **Stay Logged In Across Multiple Devices**: Fixed a critical session issue where logging in on a new device (e.g., mobile phone) would overwrite the user's single refresh token digest in the database, causing the other device (e.g., PC or laptop) to fail token refresh and forcibly log out after the 30-minute access token expired.
- **Up to 10 Active Device Sessions**: NovelHub now tracks up to 10 concurrent active refresh token sessions per user account, allowing seamless simultaneous reading across phones, tablets, eReaders, and desktop browsers.
- **Race-Safe CAS Token Rotation**: Refresh token rotation now uses an atomic Compare-And-Swap (CAS) mechanism backed by a bounded exponential retry loop, completely eliminating race condition failures when multiple devices attempt token refresh at the exact same moment.
- **Global Invalidation on Logout**: Calling `/auth/logout` increments `token_version` and clears stored tokens, guaranteeing instant, secure session termination across all devices.

### 🛡️ Reverse Proxy, Cloudflare & CSRF Protection Compatibility

- **Reverse Proxy Origin Validation**: Enhanced `sameOrigin` security checks to evaluate `X-Forwarded-Host` and RFC 7239 `Forwarded: host=...` headers, resolving `403 Forbidden: "Cross-origin auth request rejected"` when accessing NovelHub behind Cloudflare, NGINX, Caddy, or Cloudflare Tunnels (`cloudflared`).
- **Dev Loopback Equivalence**: Handled loopback origin equivalence (`localhost` $\leftrightarrow$ `127.0.0.1`) to ensure seamless local development between Vite dev servers and backend APIs.
- **Double-Submit CSRF Verification**: Supported authenticated double-submit CSRF token validation (`csrf_token` cookie + `X-CSRF-Token` header) on authentication endpoints, and updated the client-side axios response interceptor to attach `X-CSRF-Token` on session refresh calls.
- **Accurate Cookie Cleanup**: Fixed non-HTTPOnly cookie clearing for `csrf_token` to ensure proper deletion across browsers.

### 🎨 Unified Brand Identity, Logo Sizing & Typography

- **Pixel-Consistent Brand Typography**: Resolved a font metric mismatch where the brand text "NovelHub" rendered with different widths and sizes between pages (such as `/offline` at 94px vs `/podcasts` at 106px). Configured `--font-sans: Inter, ...` inside Tailwind CSS v4 `@theme` so all page wrappers and components render the same font family.
- **Standardized Logo Dimensions**: Unified brand logo height and max-width across all application navigation bars:
  - Top Navigation (`TopNav`): `h-9 w-auto max-w-12 object-contain drop-shadow-xs` with `font-sans text-lg font-black tracking-tight`.
  - Sidebars (`LibrarySidebar`, `AdminLayout`): `h-11 w-auto max-w-14 object-contain drop-shadow-sm` with `font-sans text-lg font-black tracking-tight`.
  - Unified across HomeView, ReadingCardModal, CustomQRCode, and Auth pages.
- **Instant Branding Cache-Busting**: Logo and favicon uploads generate versioned asset paths with automatic cache-busting, preventing stale service worker and browser cache retention.

### 📖 Reader & UX Enhancements

- **Mobile TTS Page Flipping Stability**: Fixed subpixel range boundary detection in paged reader modes (single and double column), eliminating jarring page bounce when speech synthesis advances across line wraps and word boundaries.
- **Touch Screen Tooltip Optimization**: Auto-dismisses tooltips on button click, touch interaction, and drawer/modal toggles, preventing sticky tooltips on mobile devices.
- **Reading Statistics Card (`ReadingCardModal`)**: Introduced an interactive modal component allowing users to view, customize, and export their light novel reading achievements and statistics.

---

## How to Update

### Docker Compose Update

To upgrade your existing NovelHub instance to **v1.0.6**, run the following commands in the directory containing your `docker-compose.yml`:

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
  -e TRUST_PROXY=true \
  -v $(pwd)/data:/data \
  azenkain/novel-hub:latest
```

### Standalone Native Binary Update

1. Download the new executable (`v1.0.6`) matching your OS/Arch from the **Assets** section below.
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
