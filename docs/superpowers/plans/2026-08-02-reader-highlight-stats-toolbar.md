# Reader Highlight, Stats, and Responsive Toolbar Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make EPUB text highlighting reject invalid selections safely, make reading-session synchronization reliable, and keep the selection toolbar usable on narrow screens by wrapping its color controls.

**Architecture:** Keep selection/range calculation in `web/src/lib/readerHighlight.ts`, orchestration and API guards in `web/src/hooks/useReaderSelection.ts`, and presentation/positioning in `ReaderSelectionToolbar.tsx` and `useReaderSelection.ts`. Preserve the existing API and service boundaries; diagnose the stats 500 at the backend repository boundary before changing behavior, and only clear accumulated stats after a successful sync.

**Tech Stack:** Go/Fiber services and repositories, SQLite/sqlc, React/TypeScript, TanStack Query, TailwindCSS/DaisyUI, Vitest/Jest-style frontend tests if already configured.

## Global Constraints

- Controllers call `apperrors.HandleError(c, err)` for service errors.
- Database queries remain in `db/query/*.sql`; no inline SQL.
- Frontend API calls remain in centralized services and mutations/hooks.
- UI text must use existing `react-i18next` translations; do not add hardcoded visible text.
- Export shared frontend types from `web/src/types/` rather than defining service/component domain types inline.
- Do not reset reading-stat counters when the request fails.
- Toolbar color buttons must remain individually clickable and must not shrink below their intended hit target.
- Remove temporary highlight diagnostic logging after the root cause is confirmed unless the logging is intentionally retained as production-safe diagnostics.

---

### Task 1: Reproduce and isolate the highlight payload failure

**Files:**
- Modify: `web/src/hooks/useReaderSelection.ts:84-103`
- Modify: `web/src/lib/readerHighlight.ts:469-495`
- Test: existing frontend test location discovered from `web/package.json` and test config; otherwise create `web/src/lib/readerHighlight.test.ts`

**Interfaces:**
- Consumes: `selectionRange: Range | null`, `getCharacterOffsetOfRange(container, range)`.
- Produces: a validated `{ text, start, end }` payload passed to `addHighlight`, or no API call for an invalid selection.

- [ ] **Step 1: Add failing unit tests for selection extraction.** Cover:
  - a selection within one text node returns trimmed non-empty text and `end > start`;
  - a selection spanning multiple text nodes returns the full selected text and document-relative offsets;
  - whitespace-only selection returns no highlight request;
  - a range whose offsets cannot be resolved returns no highlight request;
  - an end offset equal to the start offset is rejected.

  Example assertion shape:

  ```ts
  expect(addHighlight).not.toHaveBeenCalled();
  expect(getCharacterOffsetOfRange(container, range)).toEqual({ start: 12, end: 27 });
  ```

- [ ] **Step 2: Run the focused frontend test and verify it fails against the current implementation.**

  Run from `web/`: `npm test -- --run src/lib/readerHighlight.test.ts` (or the repository’s configured equivalent).

  Expected: the multi-node/invalid-selection cases fail because the current fallback can submit `selectionRange.toString()` with invalid offsets.

- [ ] **Step 3: Make range offset calculation robust.** In `getCharacterOffsetOfRange`, resolve text-node boundaries explicitly and accumulate text lengths until both boundaries are found. Account for element boundary containers by resolving the boundary to the first/last descendant text node as appropriate. Return `null` when either boundary cannot be resolved or when `end <= start`.

  Preserve the current public return type:

  ```ts
  { start: number; end: number } | null
  ```

- [ ] **Step 4: Guard `handleHighlight` before calling the API.** Trim the selected text, require a non-empty string, require an offset result, and require `end > start`:

  ```ts
  const text = selectionRange.toString().trim();
  const offset = container ? getCharacterOffsetOfRange(container, selectionRange) : null;
  if (!text || !offset || offset.end <= offset.start) return;
  await addHighlight(text, offset.start, offset.end, color);
  ```

  Keep selection cleanup after a valid attempted operation; do not fabricate fallback offsets from `text.length` when a container exists but the range cannot be mapped.

- [ ] **Step 5: Run the focused frontend tests and typecheck.**

  Run the configured test command and `npm run build` (or the repository’s frontend typecheck command).

  Expected: all new selection tests pass and no TypeScript errors are introduced.

- [ ] **Step 6: Commit the isolated highlight fix.**

  ```bash
  git add web/src/hooks/useReaderSelection.ts web/src/lib/readerHighlight.ts web/src/lib/readerHighlight.test.ts
  git commit -m "fix: validate reader highlight selections"
  ```

### Task 2: Diagnose and fix reading-session synchronization

**Files:**
- Read/modify: `web/src/hooks/useReadingStats.ts:7-51`
- Read/modify: `web/src/services/readerService.ts` at `syncReadingSession`
- Read/modify: `internal/controllers/featureController.go:703-725`
- Read/modify: `internal/services/featureService.go:621-637`
- Read/modify: the feature repository method used by `UpsertReadingSession`
- Test: matching Go service/repository tests and `web/src/hooks/useReadingStats.test.ts` if frontend test infrastructure exists

**Interfaces:**
- Consumes: `{ book_id: string, duration: number, words: number }` and existing `FeatureService.RecordReadingSession`.
- Produces: successful sync that resets counters exactly once; failed sync that retains counters for retry and returns a useful server-side error.

- [ ] **Step 1: Capture the exact failing request and server error.** Inspect the Network response body and server logs for `/reader/stats/session`; verify the request JSON has a valid UUID `book_id`, integer `duration >= 1`, and integer `words >= 0`. Trace `UpsertReadingSession` to its sqlc query and record the underlying error before it is converted to `Failed to record reading session`.

- [ ] **Step 2: Add a failing backend test for the reproduced failure.** Use the existing feature-service test style and mock repository. The test must assert that a repository error is wrapped as `apperrors.ErrInternalError` while preserving the cause in logs, and that an inaccessible book returns the expected forbidden error. If the actual failure is a SQL constraint, add a repository/integration test using the existing test database setup that reproduces the constraint with the exact payload.

- [ ] **Step 3: Fix the root cause at the correct layer.** Use the evidence from Step 1 to make one targeted change:
  - if the payload/book ID is invalid, validate the DTO with UUID validation and stop sending stats until `book_id` is valid;
  - if the SQL upsert/query is wrong, update only `db/query/*.sql`, regenerate sqlc, and update the repository test;
  - if the schema/data constraint is wrong, add the required migration/schema correction rather than swallowing the error;
  - if permission lookup is wrong, correct service access handling without bypassing `book.read`.

  Do not hide the underlying repository error behind a frontend-only retry.

- [ ] **Step 4: Make the frontend counter lifecycle reliable.** In `useReadingStats`, prevent overlapping interval/unmount sync calls with an in-flight ref. Snapshot counters before sending. Reset only the snapshot amount after a successful response; preserve any increments that occurred while the request was in flight. Keep retry behavior bounded by the existing interval/unmount lifecycle.

  Example state transition:

  ```ts
  const snapshot = { duration: durationRef.current, words: Math.floor(wordsRef.current) };
  if (snapshot.duration < 1 || syncInFlightRef.current) return;
  syncInFlightRef.current = true;
  try {
    await readerService.syncReadingSession(bookId, snapshot.duration, snapshot.words);
    durationRef.current = Math.max(0, durationRef.current - snapshot.duration);
    wordsRef.current = Math.max(0, wordsRef.current - snapshot.words);
  } finally {
    syncInFlightRef.current = false;
  }
  ```

- [ ] **Step 5: Run backend tests and frontend tests/build.**

  Run: `go test ./internal/services/... ./internal/repositories/...` and the configured frontend test/build commands.

  Expected: the reproduced 500 is resolved, failed requests do not lose accumulated stats, and successful requests do not double-count.

- [ ] **Step 6: Commit the stats fix separately.**

  ```bash
  git add web/src/hooks/useReadingStats.ts web/src/services/readerService.ts internal/controllers/featureController.go internal/services/featureService.go internal/repositories db/query internal/gen/sqlc
  git commit -m "fix: make reading session sync reliable"
  ```

### Task 3: Make the selection toolbar wrap on narrow screens

**Files:**
- Modify: `web/src/components/reader/ReaderSelectionToolbar.tsx:25-139`
- Modify: `web/src/hooks/useReaderSelection.ts:36-73`
- Test: `web/src/components/reader/ReaderSelectionToolbar.test.tsx` if configured; otherwise verify with browser tooling and component build

**Interfaces:**
- Consumes: viewport-relative `toolbarPos` and existing toolbar actions.
- Produces: a fixed toolbar that wraps naturally, preserves button hit areas, and stays within the viewport.

- [ ] **Step 1: Add a focused component test or browser assertion for narrow viewports.** Assert that the toolbar has wrapping layout, color buttons have `shrink-0`, and the toolbar’s computed width does not exceed the viewport. Verify all four color buttons remain present.

- [ ] **Step 2: Update toolbar layout classes.** Keep the toolbar fixed and centered on desktop, but add a maximum viewport width, wrapping, and non-shrinking controls. Use classes equivalent to:

  ```tsx
  className="... flex max-w-[calc(100vw-1rem)] flex-wrap ..."
  ```

  and for the color group:

  ```tsx
  className="flex shrink-0 flex-wrap items-center gap-1.5 px-1"
  ```

  Add `shrink-0` to each color button and retain accessible `title` attributes.

- [ ] **Step 3: Clamp the fixed toolbar position.** In `useReaderSelection`, calculate `left` from the selection center but clamp it to a small viewport margin. Keep `top` based on the selection rectangle and clamp it to the top margin. Use `window.innerWidth` only at selection time; do not introduce a resize listener unless browser verification shows it is needed.

  Example:

  ```ts
  const margin = 8;
  const left = Math.min(
    Math.max(rect.left + rect.width / 2, margin),
    Math.max(margin, window.innerWidth - margin),
  );
  setToolbarPos({ top: Math.max(10, rect.top - 40), left });
  ```

  If the toolbar’s own width is required to center accurately, use CSS translate plus `max-width` and rely on wrapping; do not read layout synchronously before rendering.

- [ ] **Step 4: Verify the responsive behavior with browser tooling.** Use the installed Chrome DevTools plugin or the project run skill to open the reader, test a desktop viewport and a narrow mobile viewport, select text, and click each color. Confirm the toolbar wraps instead of shrinking and no control is clipped.

- [ ] **Step 5: Run frontend build/tests and commit.**

  ```bash
  git add web/src/components/reader/ReaderSelectionToolbar.tsx web/src/hooks/useReaderSelection.ts web/src/components/reader/ReaderSelectionToolbar.test.tsx
  git commit -m "fix: wrap reader selection toolbar on mobile"
  ```

### Task 4: Remove temporary diagnostics and run full verification

**Files:**
- Modify: `internal/services/highlightService.go` to remove temporary `zerolog` instrumentation added during the prior investigation, unless the logs are explicitly needed after review
- Modify: any files changed by Tasks 1–3 only

- [ ] **Step 1: Review the final diff for accidental scope.** Confirm no generated bundle (`index-*.js`), screenshot, database file, or unrelated pre-existing user change is staged.

- [ ] **Step 2: Run Go formatting and tests.**

  ```bash
  gofmt -w internal/services/highlightService.go internal/controllers/featureController.go internal/services/featureService.go
  go test ./...
  ```

- [ ] **Step 3: Run the frontend quality checks.** From `web/`, run the project’s configured lint, test, and production build commands.

- [ ] **Step 4: Perform an end-to-end manual check.** On the EPUB:
  - select text within one paragraph and across inline elements;
  - choose every color on desktop and narrow viewport;
  - verify invalid/collapsed selections do not issue a POST;
  - leave the reader open for at least 30 seconds and confirm stats sync succeeds;
  - close/reopen the reader and confirm highlights and reading stats persist.

- [ ] **Step 5: Report verification faithfully.** Include commands run, pass/fail results, any remaining environment-only issue, and the exact response body if stats still fails.
