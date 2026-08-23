# Audit Results — NovelHub (23/08/2026)

## 🔴 P0 — Critical

### 1. ✅ ProxyAuth IP Spoofing → Authentication Bypass
- **Files:** `internal/middlewares/proxyAuthMiddleware.go:41`, `cmd/api/server.go:73,101-104`, `docker-compose.yml:12`
- **What:** With `TRUST_PROXY=true` (Docker default), `c.IP()` returns `X-Forwarded-For` value. ProxyAuth trusts this spoofable IP against TrustedProxies list.
- **Exploit:** Any Docker-bridge/loopback peer sends `X-Forwarded-For: 127.0.0.1` + `X-Forwarded-User: admin@example.com` → JWT minted for that email → full admin session. `AutoCreate` provisions arbitrary users.
- **Fix:** Trust decision now uses the raw socket peer via `c.RequestCtx().RemoteAddr()`, never the proxy-header-resolved `c.IP()`. (Note: an earlier attempt used `c.Context().(*fasthttp.RequestCtx)`, but Fiber v3's `Context()` returns a `context.Context`, not the RequestCtx, so the assertion always failed and silently disabled proxy auth — `RequestCtx()` is the correct accessor.) `auditProxyTrust_test.go` rewritten as a regression test proving a spoofed XFF from an untrusted raw peer is blocked.

---

## 🟠 P1 — High

### 2. VBook Protocol Bypasses All Per-Book/Per-Library Authorization
- **Files:** `internal/routes/vbookRoutes.go:17-31`, `internal/controllers/vbookController.go:58-255`, `internal/services/vbookService.go:105-468`
- **What:** Route group only has VBookAuth (Basic/JWT/guest). No permission middleware. All content endpoints (`GetBooks`, `GetDetail`, `GetTOC`, `GetChapterContent`, `GetAudioBooks`, `GetAudioPlaylist`) call repo directly, skip `FilterReadableBooks`/`CanReadBook`.
- **Exploit:** Guest (default enabled) or any registered user → enumerate and read full content of every book in every library. Only `/audio/stream` checks `CanReadBook`.
- **Fix:** Pass claims through VBook controller → service. Apply `FilterReadableBooks`/`ReadableLibraryIDs` for listings, `CanReadBook` for detail/toc/chap/audio/playlist. Mirror main API pattern.

### 3. Audio Progress POST ~4×/second — Network Flood + Re-render
- **Files:** `src/pages/reader/ReaderWorkspace.tsx:1235-1253`, `src/components/reader/AudioPlayer.tsx:384-388`
- **What:** `onTimeUpdate` calls `featureService.recordReadingActivity()` every HTML `timeupdate` event (~4×/sec). No throttle. AudioPlayer (39KB) re-renders on every tick.
- **Cost:** Continuous POST flood for entire listening session + constant large-component re-render.
- **Fix:** Throttle progress saves to ~5-10s (also on pause/unmount). Keep `currentTime` in ref or isolated store slice for progress bar.

### 4. Advanced Search: Every Keystroke = Full Request Storm
- **Files:** `src/pages/library/AdvancedSearchPage.tsx:269-270`
- **What:** `onChange` writes `queryInput` → feeds `useBooksQuery` directly. No debounce.
- **Cost:** Typing 10 chars = 10 book queries + 5 parallel facet queries = 150 HTTP requests per search term.
- **Fix:** Run `queryInput` through existing `useDebounce` hook before `apiQueryParams`. Also: format filter applied client-side after pagination (lines 138-142) → wrong page counts when filter active.

### 5. Admin Users/Logs Search: Every Keystroke = 1 Request
- **Files:** `src/pages/admin/Users.tsx:281`, `src/components/admin/operations/LogsTab.tsx:50`
- **What:** Same debounce-omission pattern as #4. Users: each keystroke fetches user list. Logs: each keystroke fetches 500-line log tail.
- **Fix:** Add `useDebounce` to both search inputs.

### 6. No Route-Level Code Splitting — 1.5 MB Single Bundle
- **Files:** `src/main.tsx`, `vite.config.ts`
- **What:** All routes statically imported. Zero `React.lazy`. Built output `index-DEUJdVkl.js` is 1.5 MB + 319 KB CSS. `chunkSizeWarningLimit: 2000` hides the warning.
- **Cost:** Every visitor downloads admin panels, reader, analytics, podcasts on first load.
- **Fix:** `React.lazy` per route + manualChunks vendor split. Remove `hls.js` (dead dependency).

### 7. Reader Chapter: No Prefetch, No Cache
- **Files:** `src/pages/reader/ReaderWorkspace.tsx:720-743,1011-1021`, `src/services/readerService.ts`, `internal/controllers/readerController.go`
- **What:** Next/prev chapter fetched on demand only. No prefetching. Backend `GetChapter` sets no `Cache-Control`.
- **Cost:** Every chapter turn blocks on full network round-trip (most frequent interaction).
- **Fix:** Prefetch next chapter after current renders (query cache). Add `Cache-Control` server-side.

### 8. Paged Reader: Full DOM Scan Every Scroll Event
- **Files:** `src/hooks/useReaderPaging.ts:157-302`
- **What:** `getPagedScrollMetrics()` runs `querySelectorAll("p, div, figure, h1..h6, img")` over entire chapter + forced layout on every scroll event. Re-bound on each `pageIndex` change.
- **Cost:** Long chapters → full DOM query + layout per scroll frame → jank on trackpad/touch.
- **Fix:** Cache node list per chapter (invalidate on `htmlContent` change). Or compute `maxIndex` from `scrollWidth`, only scan when clamped.

---

## 🟡 P2 — Medium

### 9. i18n — 218 Missing Keys + Hundreds of Hardcoded Strings
- **Scope:** All 16 locale files are synced (1,458 keys each, no duplicates). But 218 keys referenced in code are missing from all locale files.
- **Hardcoded Vietnamese:** 200 lines across 32 files (raw JSX, attributes, t()-fallbacks)
- **Hardcoded English:** 94 JSX nodes across 23 files, 42 attributes, 7 toasts
- **Dead keys:** 248 keys in en.json never referenced in code
- **Worst files:**
  - `BulkEditMetadataModal.tsx` — 28 missing keys + 23 hardcoded Vietnamese
  - `SmartFilterBuilderModal.tsx` — 19 missing keys
  - `ReaderSettingsPanel.tsx` — 19 missing keys + 8 hardcoded English
  - `Books.tsx` (admin) — 19 missing keys + 9 hardcoded English
  - `AudioPlayer.tsx` — 17 missing keys + 26 hardcoded Vietnamese
  - `Users.tsx` — 15 missing keys
  - `Settings.tsx` — 12 missing keys + 12 hardcoded Vietnamese
  - `HomeView.tsx` — 14 hardcoded Vietnamese
  - `ReviewSection.tsx` — 22 hardcoded Vietnamese
  - `ReaderHighlightsPanel.tsx` — 18 hardcoded Vietnamese
  - `OAuthSettings.tsx` — 23 hardcoded English

### 10. 26 Components Over 400 Lines
- `ReaderWorkspace.tsx` — 1,476 lines
- `LibraryWorkspace.tsx` — 1,340 lines
- `BulkEditMetadataModal.tsx` — 1,229 lines
- `Books.tsx` (admin) — 1,118 lines
- `Settings.tsx` (admin) — 938 lines
- `MergeAudiobookModal.tsx` — 932 lines
- `AudioPlayer.tsx` — 912 lines
- `BookDetailPage.tsx` — 883 lines
- `ReaderSettingsPanel.tsx` — 819 lines
- `WebhookModal.tsx` — 801 lines
- `CustomizationTab.tsx` — 726 lines
- `Users.tsx` — 711 lines
- `ReaderTopBar.tsx` — 695 lines
- `Roles.tsx` — 661 lines
- `AdvancedSearchPage.tsx` — 624 lines
- Plus 11 more files in 400-600 line range

### 11. BookDetailPage Unmount Invalidates All Infinite-Query Pages
- **Files:** `src/pages/library/BookDetailPage.tsx:80-89`
- **What:** Unmount runs `invalidateQueries(["books"])` + `invalidateQueries(["library"])`. Library uses `useInfiniteQuery` → every loaded page refetched.
- **Fix:** Invalidate only user-state query or use `refetchType: "none"`.

### 12. LibraryWorkspace: Query→Zustand Mirror + Unmemoized Cards
- **Files:** `src/pages/library/LibraryWorkspace.tsx`, `src/components/ui/BookCard.tsx:24`
- **What:** `useEffect` mirrors query results into Zustand → render twice. `BookCard` not memo'd, `parseMetadata(book.metadata_json)` (JSON.parse) in render body of every card. Inline `openBookDetail` closure per card.
- **Admin variant:** `src/pages/admin/Books.tsx:399` — same `metadata_json` parse per table row per render.

### 13. Reader Page Turn Renders Whole Tree
- **Files:** `src/components/reader/ComicReader.tsx:46`, `src/pages/reader/ReaderWorkspace.tsx:443-457,859-901`
- **What:** `ComicReader` subscribes whole `useReaderStore()` inside `React.memo` → defeats memo. `ReaderTopBar`/`ReaderSidebar` receive inline closures. `comicImageTarget` useMemo re-parses entire chapter HTML on every comic page turn. `getVisibleChi` does full DOM scan + `getBoundingClientRect` loop per page turn.

### 14. Webtoon Loads All Images Eagerly
- **Files:** `src/components/reader/ComicReader.tsx:278-308`
- **What:** `dangerouslySetInnerHTML` with no `loading="lazy"` on images → all pages download on chapter open.

### 15. Ambient Theme Decodes Full-Res Cover for Avg Color
- **Files:** `src/pages/reader/ReaderWorkspace.tsx:405-423`
- **What:** `FastAverageColor` loads full-res cover via `new Image()` per book/theme. Multi-MB decode just for average color.

### 16. Admin Books Re-Renders on Every Upload-Progress Tick
- **Files:** `src/pages/admin/Books.tsx`, `src/stores/bookAdminStore.ts:419-421`
- **What:** Page subscribes to `uploadProgress`/`uploadSpeed`/`uploadBytesText` — updates per progress event → entire 58KB page re-renders many times/sec during upload.

### 17. Types Outside `types/` — 8 Violations
- `AudioPlayer.tsx:24` `AudioChapter` — duplicates `AudiobookChapter` in `types/audiobook.ts`
- `AudioPlayer.tsx:30` `AudioBookmark` — imported by `ReaderSidebar.tsx`, `ReaderWorkspace.tsx`
- `ReaderImageToolbar.tsx:7,17` `ImageBookmark`, `ActiveImageTarget`
- `LibrarySidebar.tsx:12` `LibraryNavItem`, `MetadataIndexView.tsx:7` `MetadataFacetSection`
- `uploadService.ts:7` `UploadProgressStats`, `customizationService.ts:13` `CustomizationListParams`
- `usePodcastQueries.ts:108` `DownloadEpisodeParams`

### 18. Hardcoded `/api/v1` Bypassing `getMediaUrl`
- **Files:** `src/pages/admin/Books.tsx:156-159`, `src/pages/reader/ReaderWorkspace.tsx:1227`, `src/pages/auth/LoginPage.tsx:135`
- **What:** Three places hardcode `/api/v1` prefix instead of using `getMediaUrl` or `API_BASE`.

### 19. useEffect Fetches Instead of TanStack Query
- **Files:** `src/pages/auth/SetupWizard.tsx:38-49`, `src/pages/library/LibraryWorkspace.tsx:455`

---

## 🔵 P3 — Low

### 20. ✅ Login CSRF on `/api/v1/auth/signin`
- **File:** `internal/middlewares/csrfMiddleware.go:20`
- **What:** CSRF middleware skips `/api/v1/auth/` entirely. Malicious page can submit victim's browser to sign in with attacker credentials → victim's activity logged under attacker account.
- **Fix:** Auth endpoints now go through a same-origin check (Origin/Referer vs Host) instead of being skipped outright.

### 21. ✅ MAL Tracker Path: No Numeric Validation on `mangaID`
- **File:** `internal/services/trackerService.go:282`
- **What:** `external_series_id` interpolated into URL path without numeric check. Host hardcoded, own token used → no SSRF or cross-user impact.
- **Fix:** Reject non-numeric `mangaID` with `ErrBadRequest` before building the URL.

### 22. ✅ Admin Images: Full-Res Covers for 40px Thumbnails
- **File:** `src/pages/admin/Books.tsx:433-437`
- **What:** Table thumbnails load full-resolution covers, no `loading="lazy"`. (Library BookCard has it correct.)
- **Fix:** Added `loading="lazy"` to the table thumbnail.

### 23. ✅ Always-Mounted Modals
- **Files:** `src/pages/admin/Books.tsx:572` (edit dialog), `src/pages/library/LibraryWorkspace.tsx:1226-1227` (LoginView/UserProfile)
- **Fix:** Edit-metadata, new-collection, and save-search dialogs now render conditionally on their open state.

### 24. ✅ `hls.js` Dead Dependency
- **File:** `package.json`
- **What:** Never imported in any source file.
- **Fix:** Removed.

### 25. ✅ `key={index}` on Mutable Rows
- **File:** `src/components/library/SmartFilterBuilderModal.tsx:161`
- **What:** Rows below a deletion get mismatched state. Use stable rule id.
- **Fix:** Added optional `id` to `SmartFilterRuleItem`, assign a stable id per rule, key rows by it, strip it from the save payload.

### 26. ✅ `Intl.NumberFormat` Inline Per Render
- **File:** `src/pages/library/LibraryWorkspace.tsx:852,1069`
- **Fix:** Hoisted to a module-level `numberFormatter`.

### 27. ✅ `DOMPurify.sanitize` Not Memo'd
- **File:** `src/pages/library/BookDetailPage.tsx:776`
- **Fix:** Wrapped in `useMemo` keyed on description + t.

### 28. ✅ `DownloadManagerPanel` Whole-Store Subscription
- **File:** `src/components/download/DownloadManagerPanel.tsx:31`
- **Fix:** Switched to a `useShallow` selector over only the fields/actions used.

### 29. 248 Dead Keys in `en.json`
- Defined but never referenced in code (informational — cleanup opportunity). Left as-is.

---

## ✅ Verified Clean

- **BE Permission:** 44 permission constants seeded in schema; all POST/PUT/PATCH/DELETE routes carry auth + permission middleware; IDOR checks on all user-scoped endpoints; magic-code/OTP/TOTP/OAuth all rate-limited with expiry + single-use; upload sessions verify owner; bulk ops check per-book permissions.
- **BE Security:** Path traversal blocked everywhere (SafeJoin + Lstat + symlink reject); SSRF blocked (netx.NewSafeHTTPClient on all outbound); zero raw SQL in app code (all sqlc); zip bombs handled (budget enforcement); command injection — no shell, server-generated args only; JWT HS256 pinned + token_version revocation; reader HTML rewrite + CSP sandbox; mailer header-injection rejection + private-IP block on SMTP.
- **FE Rules:** Zero `api.get/post` in components; permission utilities used correctly (66 call sites); 16 locale files perfectly synced (1,458 keys each, no duplicates).
- **FE Performance:** `staleTime: 5min` + `refetchOnWindowFocus: false` in queryClient; debounce present on TopNav/Library/Metadata search; `loading="lazy"` on library grid images; `ReaderContent` properly memo'd; infinite-query pagination; no lodash/moment; tree-shakeable lucide-react imports.

---

## Suggested Fix Order

| Priority | Action | Effort |
|----------|--------|--------|
| 🔴 P0 | Fix #1 — ProxyAuth: use `RemoteAddr` instead of `c.IP()` | Small |
| 🟠 P1 | Fix #2 — VBook: pass claims, add FilterReadableBooks + CanReadBook | Medium |
| 🟠 P1 | Fix #3 — Audio progress: throttle to 5-10s + refactor currentTime out | Small |
| 🟠 P1 | Fix #4-5 — Search debounce (useDebounce) | Small |
| 🟠 P1 | Fix #6 — React.lazy route splitting + remove hls.js | Medium |
| 🟠 P1 | Fix #7 — Prefetch next chapter + server Cache-Control | Medium |
| 🟠 P1 | Fix #8 — Cache DOM node list in paged reader | Medium |
| 🟡 P2 | Fix #9 — i18n: add 218 keys × 16 files, extract ~350 hardcoded strings | Large (mechanical) |
| 🟡 P2 | Fix #10 — Split oversized components | Large (structural) |
| 🟡 P2 | Fix #11-16 — React perf: targeted memo/useMemo/selector fixes | Medium |
| 🟡 P2 | Fix #17-19 — Move types, use getMediaUrl, migrate useEffect→useQuery | Small |
| 🔵 P3 | Fix #20-29 — Cleanup items | Small each |
