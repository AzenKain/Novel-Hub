# Fix Reader FE: UI/UX bugs and language switching not updating

## Context

The Reader (`/reader/:bookId`) has several reader-facing issues:
- Language switch from top nav / settings does **not** repaint Reader UI text (toolbar, panels, audio controls) because `t` is passed as a prop from `ReaderWorkspace` to children, and i18next language cache duplicates Zustand.
- Hardcoded English strings remain in accessibility labels, tooltips, and fallback defaults (`AudioPlayer`, `ReaderSelectionToolbar`, `ReaderSidebar`, `ReaderTopBar`).
- Japanese/Korean/Chinese locales miss `reader.close_toc`, `reader.back_to_previous`, `reader.reset_settings`; `ReaderTtsSettingsPanel` uses `reader.current_voice` but locales have `reader.active_voice`.
- Position lost on font/theme change because a reset effect depends on typography deps.
- Reading progress is chapter-only; paged-mode manual scroll does not persist.
- In-book search receives `offset` but drops it — match not scrolled into view.
- No keyboard shortcuts (next/prev chapter, close panels, Escape), no focus management when panels open/close.
- Selection toolbar uses hardcoded dark colors ignoring reader theme (`--reader-ui-*` variables exist but unused).
- `lineHeight` persisted and applied in CSS but missing UI control in Settings Panel.
- PDF/raw HTML path has hardcoded `bg-white` and no theming.
- Double-page silently falls back to single when viewport < 380px; no user feedback.

Goal: fix these reader-specific UI/UX bugs using existing patterns (theme variables, Zustand, `useTranslation`, locales) — no new infrastructure, no sweeping redesign.

## Plan

### 1. Single source of truth for language (fixes stale Reader on language switch)

**Files:** `web/src/i18n.ts`, `web/src/components/ui/LanguageSwitcher.tsx`

- Remove `LanguageDetector` and its `caches: ['localStorage']` from i18n init — `settingsStore` already persists `language` in `novelhub-settings`.
- Initialize i18next with `useSettingsStore.getState().language || "en"`.
- In `LanguageSwitcher`, on selection: set Zustand language, then `void i18n.changeLanguage(language)`. Remove the effect that syncs Zustand → i18n; keep only the handler.

**Validation step:** Manually switch language while Reader open; if any child still stale, move `useTranslation()` locally into that component (Phase 1b fallback).

### 2. Complete reader localization & accessibility labels

**Files:** `web/public/locales/{en,vi,ja,ko,zh}.json`, `web/src/components/reader/ReaderTtsSettingsPanel.tsx`, `web/src/components/reader/ReaderTopBar.tsx`, `web/src/components/reader/ReaderSidebar.tsx`, `web/src/components/reader/ReaderSelectionToolbar.tsx`, `web/src/components/reader/AudioPlayer.tsx`

Add missing keys (ja/ko/zh):
```json
"reader.close_toc": "Close Table of Contents",
"reader.back_to_previous": "Back to Previous",
"reader.reset_settings": "Reset Settings"
```

Align voice key:
- Change `ReaderTtsSettingsPanel.tsx` to use existing `reader.active_voice` (not `reader.current_voice`).

Add new keys for currently hardcoded text (all 5 locales):
```json
"reader.reading": "Reading",
"reader.audiobook": "Audiobook",
"reader.cover_art": "Cover art",
"reader.highlight_yellow": "Highlight Yellow",
"reader.highlight_green": "Highlight Green",
"reader.highlight_blue": "Highlight Blue",
"reader.highlight_purple": "Highlight Purple",
"reader.skip_back_15_seconds": "Skip back 15 seconds",
"reader.skip_forward_15_seconds": "Skip forward 15 seconds",
"reader.play_pause": "Play/Pause",
"reader.mute": "Mute",
"reader.unmute": "Unmute",
"reader.open_toc": "Open Table of Contents",
"reader.open_settings": "Open Reader Settings",
"reader.double_page_unavailable": "Double page requires wider viewport",
"reader.line_height": "Line Height"
```

Reuse existing where possible: `common.unknown` for author, `common.close` for close buttons.

Replace every literal title/aria-label/alt/fallback in the listed components with `t("reader.*")`.

### 3. Preserve reading location and track true progress

**File:** `web/src/pages/reader/ReaderWorkspace.tsx`

- Narrow the position-reset `useEffect` (currently deps on fontSize/fontFamily/lineHeight/maxWidth/effectiveReadingMode/htmlContent) to **only** reset on true chapter change (`htmlContent` / `currentChapter`). Remove typography/layout deps.
- Keep explicit reset inside `loadChapter` for chapter transitions.
- Reuse `getPagedScrollMetrics()` to derive `locationFraction` (0–1):
  - scroll/webtoon: `scrollTop / (scrollHeight - clientHeight)`
  - paged: `pageIndex / maxIndex`
- Replace chapter-only `progressPercent` with location-aware:
  `((chapterPosition + locationFraction) / chapters.length) * 100`
- Add passive `scroll` listener on paged container (`getPagedScrollContainer()`) to update `pageIndex` from manual horizontal scroll (`Math.abs(scrollLeft) / scrollStep`).
- Persist the location-aware percentage to reading progress / session.

### 4. Restore exact in-book search navigation (use `offset`)

**Files:** `web/src/pages/reader/ReaderWorkspace.tsx`, `web/src/lib/readerHighlight.ts`

- Preserve `onSelectResult(chapterId, offset)` contract.
- Before `loadChapter(chapter)`, store pending `textOffsetTarget`.
- After chapter render, walk rendered text nodes in `readerHighlight.ts` to resolve the character offset to a DOM range, then `range.scrollIntoView({ block: "center" })`.
- Clear pending target.
- If backend `offset` proves to be raw-HTML not rendered-text offset, stop at chapter navigation and note API contract mismatch for BE follow-up.

### 5. Keyboard navigation & focus management

**Files:** `web/src/pages/reader/ReaderWorkspace.tsx`, `web/src/components/reader/ReaderTopBar.tsx`, `web/src/components/reader/ReaderSidebar.tsx`, `web/src/components/reader/ReaderSettingsPanel.tsx`, `web/src/components/reader/ReaderInBookSearch.tsx`

- Add single `keydown` handler in `ReaderWorkspace` (guard `target` not input/textarea/select):
  - `Escape`: close open surface priority — search → settings/TTS → sidebar → selection toolbar; restore focus to originating top-bar button.
  - Arrow Left/Right: only in paged modes, navigate prev/next page (reverse for RTL).
  - Leave native arrows in scroll/webtoon & inputs.
- Track opener refs (`tocBtnRef`, `searchBtnRef`, `settingsBtnRef`) in `ReaderTopBar`.
- On open: `openerRef.current.focus()` primary control (sidebar close btn, settings first input, search input).

### 6. Apply reader theme consistently

**Files:** `web/src/components/reader/ReaderSelectionToolbar.tsx`, `web/src/components/reader/ReaderSettingsPanel.tsx`, `web/src/pages/reader/ReaderWorkspace.tsx`, `web/src/styles.css`

- Give selection toolbar a class (`.reader-selection-toolbar`) and style via existing CSS variables: `--reader-ui-surface`, `--reader-ui-text`, `--reader-ui-border`, `--reader-ui-hover`.
- Retain yellow/green/blue/purple swatch colors for highlights; only chrome adapts.
- Add line-height control in `ReaderSettingsPanel`:
  - Pass `lineHeight` / `setLineHeight` from `ReaderWorkspace` → `ReaderTopBar` → `ReaderSettingsPanel`.
  - Range `1.2–2.5` step `0.1` (match font-size pattern).
- Replace iframe `bg-white` with reader-theme surface class (note: embedded PDF page itself cannot be recolored).

### 7. Clarify double-page fallback

**Files:** `web/src/components/reader/ReaderSettingsPanel.tsx`, `web/public/locales/*.json`

- When `canUseDoubleMode` false: keep option disabled, add tooltip/hint with `t("reader.double_page_unavailable")`.
- No toast on resize; stored preference `"double"` auto-activates when width sufficient.

### 8. Fragment link handling (verify existing, fix if needed)

**Files:** `web/src/pages/reader/ReaderWorkspace.tsx` (existing `handleContentClick` + `scrollToFragment`)

- Do not add new fragment code initially.
- Browser test: same-chapter `#fragment` in scroll and paged; cross-chapter `section:path#fragment`.
- If paged fails, revise `scrollToFragment` to compute target page via `getPagedScrollMetrics()` and call `scrollToPageIndex`.

## Verification

- `npm run typecheck && npm run build` from `web/`
- Language test: switch all 5 languages **without reload**; verify Reader top bar, sidebar, settings/TTS/search panels, selection toolbar, audio controls repaint.
- Position preservation: scroll into chapter → change font size, family, line-height, width, theme → confirm approximate location stable; paged manual scroll → reload → confirm restored page.
- Search: pick result in same/other chapter → match centered on screen.
- Keyboard: open/close each panel with Escape; arrow paging in paged mode; focus restored to originating button.
- Themes: light/sepia/dark/dim/warm/coffee → selection toolbar contrast readable.
- Fragment links: EPUB `#id` links work in both scroll and paged.
- Narrow viewport: double-page stored pref disables with hint, re-enables when wide.

## Critical files
- `web/src/pages/reader/ReaderWorkspace.tsx`
- `web/src/components/ui/LanguageSwitcher.tsx`
- `web/src/i18n.ts`
- `web/src/components/reader/ReaderSettingsPanel.tsx`
- `web/public/locales/en.json`