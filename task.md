# Remaining audit tasks

These are the remaining findings from the earlier full-codebase audit after the completed HIGH fixes (token revocation, cursor pagination, atomic file/book deletion, and collection ownership lookup).

## Medium priority

- [ ] **Calibre path traversal** — validate `cb.Path` with `localfs.SafeJoin` before reading/writing Calibre-supplied paths.
- [ ] **Controller error mapping** — replace generic `400`/`404` translations with `apperrors.HandleError` where service errors carry a meaningful status.
- [ ] **ZIP export copy errors** — propagate `io.Copy` failures in `StreamLibraryZip` instead of silently producing a corrupt/truncated archive.
- [ ] **Reading activity lock contention** — replace the global mutex in `RecordReadingActivity` with a narrower per-key strategy only if profiling confirms contention.
- [ ] **Progress percentage constraint** — add a schema `CHECK` ensuring reading progress remains in `[0, 100]`.
- [ ] **Comic archive safety cap** — limit archive entries and uncompressed expansion to prevent resource exhaustion from hostile comic files.
- [ ] **Refresh JWT algorithm pinning** — validate refresh tokens only against the expected signing method/algorithm.
- [ ] **Singleflight gaps** — add `sfg.Do` to `SearchBooks` and `GetTagByName` DB fallbacks.
- [ ] **Metadata sync errors** — stop swallowing metadata synchronization failures; return or log with a retry path according to caller semantics.
- [ ] **Library list truncation** — decide API pagination/limit behavior for permission-filtered libraries; current fixed limit can omit visible libraries above the cap.
- [ ] **Library ZIP memory use** — stream/page book export instead of materializing up to one million book records.

## Low priority

- [ ] **`file_hash` index** — correct the index column/order if it does not match duplicate-hash lookup queries.
- [ ] **Jobs composite index** — add the index matching unfinished-job scheduling predicates if query plans show a scan.
- [ ] **User search wildcard** — avoid or document leading-wildcard search cost; consider FTS only if needed.
- [ ] **Avatar URL validation** — allow only `http`/`https` URL schemes at the validation boundary.
- [ ] **Calibre filesystem mutations in transactions** — commit database mutations before external filesystem effects where rollback safety matters.
- [ ] **Explicit `RETURNING` columns** — replace remaining `RETURNING *` with selected columns.
- [ ] **Frontend mutation failures** — surface failed mutations rather than silently ignoring them.
- [ ] **Frontend permission/i18n cleanup** — use centralized role helpers and translation keys consistently.

## Deliberately not included

- Bulk collection assignment checks only caller-supplied collection IDs; it does not load the user’s full collection list.
- Job processing’s 1,000-row cap is an intentional worker batch.
- Role/permission catalog loads are cache initialization/global catalog reads.
