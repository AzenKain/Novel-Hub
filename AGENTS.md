# NovelHub Comprehensive Project Rules & Guidelines

These project-scoped rules are mandatory for all AI coding agents working on the NovelHub project. Adhering to these design patterns ensures system stability, high performance, clean separation of concerns, and prevents technical debt.

---

## 🏛️ 1. System Architecture & Layered Boundaries

To maintain clean code maintainability, the project enforces a strict three-layer boundary:

```
[ Controller ] <--> [ Service ] <--> [ Repository ] <--> [ Database / RAM Cache ]
```

### A. Controller Layer (`internal/controllers`)
- **Responsibility**: Controllers must only handle parsing HTTP requests, triggering validations, invoking services, mapping outputs, and returning HTTP responses.
- **Strict Constraints**: 
  - Controllers **MUST NOT** directly handle any business logic, DB queries, file parsing, or HTML/CSS manipulation.
  - All input payloads (body, query, params) **MUST** be validated using the custom validator package [`pkg/validator`](./pkg/validator):
    - Use `validator.ValidateBodyDto(c, &dto)` for request bodies.
    - Use `validator.ValidateQueryDto(c, &dto)` for query parameters.
  - Return responses using standard DTO envelopes from `internal/dtos/response`.
  - Handle all service/repository errors using `return apperrors.HandleError(c, err)` to map domain errors to standard HTTP response envelopes.
  - **Context Handling**: Controllers MUST NOT pass `c.Context()` (the fiber context) down to services. Instead, they MUST create a new context with timeout:
    ```go
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    ```

### B. Service Layer (`internal/services`)
- **Responsibility**: Houses core business rules, file parsing (EPUB, PDF, MOBI, FB2, DOCX, etc.), HTML/CSS reader scoping, and transactional use-cases.
- **Modularization & File Size**:
  - **Do NOT overload a single service file**. If a service grows large, it **MUST** be split into focused domain sub-files (e.g. `bookService_reader.go`, `bookService_metadata.go`, `bookService_catalog.go`).
- **Database Transactions & Atomicity**:
  - Any service method performing multiple database mutations (e.g. creating/updating a book along with its chapters, metadata, and files) **MUST** execute atomically within a database transaction.
  - Use `pkg/database/transaction.go` (`TxManager`) or `repo.WithTx(tx)` to begin, commit, and rollback transactions cleanly.
- **Data Boundaries**:
  - Service functions **MUST** take input arguments and return results using structures defined under the [`internal/dtos`](./internal/dtos) package (`request/` and `response/`).
  - Services **MUST NOT** leak raw SQL structures (`sqlc`) or internal domain models directly to the controllers.
  - **No Trivial Wrapper Functions / Helper Overkill**: Avoid creating 1-line helper functions or private wrapper methods (e.g. `magicCodeKeyByCode`, `invalidate(...)`). Inline cache key creations (`cache.BuildKey(...)`) and cache deletions (`r.c.Del(...)`) directly inside the parent methods.
  - Wrap database `sql.ErrNoRows` in `apperrors.New(apperrors.ErrNotFound, ...)` and validation failures in `apperrors.New(apperrors.ErrBadRequest, ...)` to ensure clean HTTP status mapping.

### C. Repository Layer (`internal/repositories`)
- **Responsibility**: Data persistence layer wrapping database queries and RAM Caching.
- **Cache-by-IDs Pattern (Mandatory)**:
  - All repository queries returning list datasets or searches **MUST** enforce the **Cache-by-IDs** pattern:
    1. Query the database to retrieve a slice of IDs (e.g., via `SearchBookIDs`, `ListFilteredJobIDs`, `ListJobScheduleIDs`).
    2. Check the RAM cache for the corresponding entities by ID via `MGet` (`GetByIDs` / `GetJobsByIDs`).
    3. Query the database only for the IDs that missed the cache and cache missing entities via `MSet`.
  - **Singleflight Protection (Mandatory)**: EVERY Repository read operation that checks RAM Cache and falls back to Database queries MUST wrap the DB fallback execution inside `singleflight.Group` (`r.sfg.Do(key, func() (any, error) {...})`) to eliminate Thundering Herd / Cache Stampede issues under heavy load.
- **Type Safety**:
  - Repositories are allowed to accept generated `sqlc` types for query params.
  - However, all repository return types **MUST** be converted into entities defined in the [`internal/models`](./internal/models) package.

### D. Domain Models (`internal/models`)
- **Responsibility**: Houses domain entity definitions (e.g., `BookEntity`, `UserEntity`, `JobEntity`, `JobScheduleEntity`).
- **Mapping Helpers**:
  - Entities **MUST** provide `FromSqlc` conversion methods to construct entities from `sqlc` generated rows.
  - Entities **MUST** provide `ToResponse` methods to convert domain entities to DTO response structs.
  - **Strict Field Checking**: When accessing entity fields, do not guess their names or types. Always look at the struct definition in `internal/models` before using it.

---

## ⚡ 2. Shared Packages (`pkg/`)

All utility logic must be centralized within the `pkg/` directory:

1. **Error Handling ([`pkg/apperrors`](./pkg/apperrors))**:
   - Controllers and services **MUST** return errors using standard errors defined in `apperrors` (e.g., `apperrors.New(apperrors.ErrBadRequest, "Invalid ID")`).
   - Standard error variables: `ErrNotFound`, `ErrConflict`, `ErrUnauthorized`, `ErrForbidden`, `ErrBadRequest`, `ErrInternalError`.

2. **Data Conversion ([`pkg/convert`](./pkg/convert))**:
   - Do not write custom data parsers or conversions in services.
   - Use `convert.ParseID(value string) (int64, error)` to validate and parse database IDs.
   - Use `convert.StrPtrToNullString` or `convert.NullStringToStrPtr` for `sql.NullString` mappings.
   - Use `convert.EncodeCursor` / `convert.DecodeCursor` for pagination cursors.

3. **Invariants & Cache Times ([`pkg/constants`](./pkg/constants))**:
   - Centralize all cache key constants (`CacheKeyJobScheduleList`, `CacheKeyJobListPattern`, etc.) in `pkg/constants/cache_keys.go`. Never hardcode raw key strings, timeouts, role names, or magic strings in repositories or services.

4. **JSON Engine ([`pkg/jsonx`](./pkg/jsonx))**:
   - Direct imports of `encoding/json` or `github.com/bytedance/sonic` inside application logic are strictly forbidden.
   - You **MUST** use `jsonx.Marshal`, `jsonx.Unmarshal`, `jsonx.MarshalString`, `jsonx.UnmarshalString`, or `jsonx.MarshalIndent` for all JSON coding.

5. **SSRF-Safe HTTP Client ([`pkg/netx`](./pkg/netx))**:
   - Any outbound HTTP requests (e.g., cover downloads) **MUST** use `netx.NewSafeHTTPClient(timeout)` to prevent Server-Side Request Forgery and DNS Rebinding at the socket connection level.

6. **Path Traversal Protection ([`pkg/localfs`](./pkg/localfs))**:
   - All file path joins and absolute resolution **MUST** use `localfs.SafeJoin(base, parts...)` which uses `filepath.Rel` to eliminate path traversal vulnerabilities.

7. **Input Validation ([`pkg/validator`](./pkg/validator))**:
   - Validate HTTP request DTOs using `validator.ValidateBodyDto(c, &dto)` or `validator.ValidateQueryDto(c, &dto)`.

8. **Background Queue ([`pkg/worker`](./pkg/worker))**:
   - Async parsing, maintenance tasks, and job schedules must be dispatched through the bounded worker pool in `pkg/worker`.

---

## 🎨 3. Frontend Guidelines (`web/src` & `web/public`)

- **Package Manager Mandatory (Bun Only)**:
  - Frontend development, dependency installation, typechecking, and production builds **MUST** use `bun` (`bun run build`, `bun run typecheck`, `bun add <pkg>`). Using `npm` or `yarn` is strictly forbidden.
- **Component Organization**:
  - Component files must be modularized under `web/src/components/` categorized by domain (`admin`, `book-detail`, `common`, `library`, `profile`, `reader`, `ui`).
  - Do not create monolithic 1,000+ line JSX files. Extract sub-views into dedicated, focused components (~200-400 lines max).
- **Permission Checking & Security Utilities**:
  - Use centralized permission helpers in `web/src/utils/permission.ts` (`isAdminUser`, `isBannedUser`, `hasPermission`). Do NOT write raw inline string checks for role names (`role.name === "ADMIN"`) in UI components.
- **TypeScript Types Location**:
  - ALL TypeScript interfaces, DTOs, and entity types **MUST** be defined under `web/src/types/` (e.g. `book.ts`, `admin.ts`, `auth.ts`) and exported via `web/src/types/index.ts`.
  - NEVER define inline interfaces or duplicate types inside `web/src/services/` or `web/src/components/`.
- **API Services & HTTP Client**:
  - ALL HTTP requests to the backend **MUST** be made via centralized client services in `web/src/services/`.
  - NEVER write direct `api.get` / `api.post` or raw `fetch` calls directly inside React components, pages, or custom hooks.
  - Use the configured Axios instance in `web/src/config/api.ts`, which includes automatic 401 token refresh queueing and cookie handling.
  - Use `getMediaUrl(path)` for constructing image/media URLs.
- **Data Fetching & Mutations (React Query)**:
  - ALL async data fetching and server mutations in components/pages **MUST** use TanStack React Query hooks (`useQuery`, `useMutation`) defined under `web/src/hooks/` (e.g. `useBooksQuery`, `useAdminQueries`, `useLibraryQueries`).
  - NEVER write raw `useEffect` with inline `fetch` or `api.get` calls inside component/page render loops for data fetching.
- **State Management**:
  - Use Zustand stores under `web/src/stores/` (`useAuthStore`, `useThemeStore`, `useReaderSettingsStore`, `useSettingsAdminStore`, `useWebhookStore`) for global UI state, theme, user authentication, and reader settings.
  - Transient form state local to a single modal (e.g., typing dở in an input field) may use `useState`.
- **Internationalization (i18n & `web/public/locales/`)**:
  - Translation JSON files are stored under [`web/public/locales/`](./web/public/locales/) (`en.json`, `vi.json`, `ja.json`, `ko.json`, `zh-CN.json`, `zh-TW.json`, `es.json`, `fr.json`, `de.json`, `pt.json`, `ru.json`, `ar.json`, `hi.json`, `id.json`, `th.json`, `it.json`).
  - **NO HARDCODED TEXT**: Never hardcode raw user-facing text strings in TSX components. All UI labels, buttons, messages, and placeholders **MUST** be added to `web/public/locales/` JSON files and rendered via `t('translation_key')` from `react-i18next`.
  - **Multi-Language Synchronization**: When adding or updating translation keys, agents MUST synchronize changes across ALL locale files in `web/public/locales/`, support dynamic parameter interpolation (`{{param}}`), and verify zero duplicate keys exist.
- **Admin Settings vs Environment Variables**:
  - Environment variables in `.env` are reserved for core system secrets (`JWT_SECRET`, `JWT_REFRESH_SECRET`, `DB_ENCRYPTION_KEY`) and low-level proxy/network infrastructure (`TRUST_PROXY`).
  - All dynamic feature configuration (e.g. SMTP server credentials, email max attachment size `smtp.max_attachment_mb`, registration, guest access policies, upload limits, rate limits) **MUST** be persisted in the `app_settings` database table and configured dynamically via Admin Settings UI.

---

## 🔒 4. Performance & Security Safeguards

- **STRICT RULE: ZERO INLINE / RAW SQL IN APPLICATION CODE (SQLC MANDATORY)**:
  - **STRICTLY FORBIDDEN**: Writing inline SQL queries, raw SQL strings (`"SELECT ..."`), or executing `db.Exec(...)` / `db.Query(...)` directly inside Go application logic, repositories, controllers, or services for NovelHub DB is **STRICTLY PROHIBITED**.
  - **SQLC ONLY**: Every database query for NovelHub **MUST** be defined in `.sql` files under `db/query/` and generated via `make sqlc`.
  - **Exceptions**: The ONLY exception is reading external 3rd-party SQLite database files (such as Calibre's `metadata.db`) or applying schema migrations from the `db/schema/*.sql` files embedded into the binary via `db.SchemaFS`.
- **Unbounded Queries**: Every list or search query **MUST** enforce a pagination boundary: `if limit <= 0 || limit > 100 { limit = 20 }`.
- **SQL Projection**: Always specify explicit columns in SQL queries under `db/query` (avoid `SELECT *` projection at runtime).
- **SQLite Performance**: Maintain SQLite connection limits: `MaxOpenConns` should be CPU-bound (typically 4-16) and `journal_mode=WAL` with `synchronous=NORMAL` must be preserved.
- **System Operations Safety**:
  - All log file downloads and tail operations MUST enforce `localfs.SafeJoin` and check `os.Lstat` for regular file types to eliminate path traversal vulnerabilities.
  - Database backups MUST use the SQLite Online Backup API (`sqliteSnapshot`) to prevent database file locking.
  - Staged restores MUST verify SHA-256 database hashes and run `DatabaseHealthCheck` before enabling restore flags (`RESTORE_AUTO_RESTART`).
  - **Library Inbox**: the `scan_library_inbox` job imports files dropped into `DATA_DIR/inbox/<libraryID>/`, copies them into managed storage via the normal upload flow, then deletes the source file. Files newer than 10s are skipped (still copying). Libraries never reference arbitrary paths on disk: `bookFileRepository` is `os.Root`-sandboxed to `DATA_DIR/books` and `maintenance` deletes DB records for files it cannot see.
- **Authentication**: JWT authentication uses token versioning (`token_version`). Validating token version in middleware guarantees instant logout across all devices.

---

## 🛑 5. Agent Workflow & Feature Proposal

- **Feature Proposals**: Do not propose or re-implement features that already exist in the codebase (e.g., Metadata Fetching/Scraping is already handled by frontend; Cross-device sync, Reviews/Ratings, Share Links are already built-in).
- **Ordered Reading Lists**: `read_lists` / `read_list_books` already provide per-user lists with an explicit `position`, drag reordering, a next-book handoff in the reader, and ComicRack `.cbl` import (`pkg/bookparser/comic/cbl.go`), all gated by `book.collection`. `position` is deliberately NOT `UNIQUE` — swapping adjacent entries must pass through a duplicate value mid-transaction.
- **Core Engine Mechanics**: The codebase already handles **Chunked Uploads** (for files >100MB), **Smart Garbage Collection** for orphaned uploads, and **Native Audio Streaming** (MP3, M4B, FLAC) without FFmpeg/HLS. Do NOT propose adding FFmpeg or HLS back into the project.
- **Codebase Review**: You **MUST** thoroughly search the codebase and database schemas (`db/schema/*.sql`) to verify if a feature or schema exists before planning to build it.
- **eReader Standards & Detailed Test Cases**:
  - All eReader integration features (OPDS feeds, Kobo Sync, Kindle Send-to-Kindle, eReader Passwordless/QR Login, eReader Web view) **MUST** study and reference established implementation patterns from [Calibre-Web (`janeczku/calibre-web`)](https://github.com/janeczku/calibre-web).
  - Every eReader and core system feature **MUST** be backed by comprehensive, detailed unit tests (`*_test.go` or `*.test.ts`) covering success, failure, edge cases, and pagination behavior. Laziness, skipping test cases, or breaking project rules is strictly prohibited.
