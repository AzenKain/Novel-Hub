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
  - **No Trivial Wrapper Functions**: Avoid creating 1-line helper functions in services (e.g. converting types or wrappers around simple conversions). Use package helpers directly.

### C. Repository Layer (`internal/repositories`)
- **Responsibility**: Data persistence layer wrapping database queries and RAM Caching.
- **Cache-by-IDs Pattern (Mandatory)**:
  - All repository queries returning list datasets or searches **MUST** enforce the **Cache-by-IDs** pattern:
    1. Query the database to retrieve a slice of IDs (e.g., via `SearchBookIDs`).
    2. Check the RAM cache for the corresponding entities by ID.
    3. Query the database only for the IDs that missed the cache.
    4. Fill the cache with the newly fetched entities and return the ordered slice.
  - Singleflight: Use `singleflight.Group` to prevent Cache Stampede (Thundering Herd) issues under heavy load.
- **Type Safety**:
  - Repositories are allowed to accept generated `sqlc` types for query params.
  - However, all repository return types **MUST** be converted into entities defined in the [`internal/models`](./internal/models) package.

### D. Domain Models (`internal/models`)
- **Responsibility**: Houses domain entity definitions (e.g., `BookEntity`, `UserEntity`, `ChapterEntity`).
- **Mapping Helpers**:
  - Entities **MUST** provide `FromSqlc` conversion methods to construct entities from `sqlc` generated rows.
  - Entities **MUST** provide `ToResponse` methods to convert domain entities to DTO response structs.
  - **Strict Field Checking**: When accessing entity fields, do not guess their names or types. For example, `BookEntity` uses `AuthorName *string` (not `Authors`) and `Description *string` (requires nil checks). Always look at the struct definition in `internal/models` before using it.

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
   - Define all cache durations and app invariants in the `constants` package. Never hardcode timeouts, role names, or magic strings.

4. **JSON Engine ([`pkg/jsonx`](./pkg/jsonx))**:
   - Direct imports of `encoding/json` or `github.com/bytedance/sonic` inside application logic are strictly forbidden.
   - You **MUST** use `jsonx.Marshal`, `jsonx.Unmarshal`, `jsonx.MarshalString`, `jsonx.UnmarshalString`, or `jsonx.MarshalIndent` for all JSON coding.

5. **SSRF-Safe HTTP Client ([`pkg/netx`](./pkg/netx))**:
   - Any outbound HTTP requests (e.g., cover downloads) **MUST** use `netx.NewSafeHTTPClient(timeout)` to prevent Server-Side Request Forgery and DNS Rebinding at the socket connection level.

6. **Input Validation ([`pkg/validator`](./pkg/validator))**:
   - Validate HTTP request DTOs using `validator.ValidateBodyDto(c, &dto)` or `validator.ValidateQueryDto(c, &dto)`.

7. **Background Queue ([`pkg/worker`](./pkg/worker))**:
   - Async parsing and maintenance tasks must be dispatched through the bounded worker pool in `pkg/worker`.

---

## 🎨 3. Frontend Guidelines (`web/src` & `web/public`)

- **Component Organization**:
  - Component files must be modularized under `web/src/components/` categorized by domain (`admin`, `book-detail`, `common`, `library`, `profile`, `reader`, `ui`).
  - Do not create monolithic 1,000+ line JSX files. Extract sub-views into dedicated, focused components (~200-400 lines max).
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
  - Translation JSON files are stored under [`web/public/locales/`](./web/public/locales/) (`en.json`, `vi.json`, `ja.json`, `ko.json`, `zh.json`).
  - **NO HARDCODED TEXT**: Never hardcode raw user-facing text strings in TSX components. All UI labels, buttons, messages, and placeholders **MUST** be added to `web/public/locales/` JSON files and rendered via `t('translation_key')` from `react-i18next`.
- **Styling**:
  - Use TailwindCSS and DaisyUI utility classes. Avoid raw inline `style={{...}}` props unless computing dynamic positioning/dimensions.

---

## 🔒 4. Performance & Security Safeguards

- **STRICT RULE: ZERO INLINE / RAW SQL IN APPLICATION CODE (SQLC MANDATORY)**:
  - **STRICTLY FORBIDDEN**: Writing inline SQL queries, raw SQL strings (`"SELECT ..."`), or executing `db.Exec(...)` / `db.Query(...)` directly inside Go application logic, repositories, controllers, or services for NovelHub DB is **STRICTLY PROHIBITED**.
  - **SQLC ONLY**: Every database query for NovelHub **MUST** be defined in `.sql` files under `db/query/` and generated via `make sqlc`.
  - **Exceptions**: The ONLY exception is reading external 3rd-party SQLite database files (such as Calibre's `metadata.db`) or applying schema migrations by reading `.sql` schema files from disk.
- **Unbounded Queries**: Every list or search query **MUST** enforce a pagination boundary: `if limit <= 0 || limit > 100 { limit = 20 }`.
- **SQL Projection**: Always specify explicit columns in SQL queries under `db/query` (avoid `SELECT *` projection at runtime).
- **SQLite Performance**: Maintain SQLite connection limits: `MaxOpenConns` should be CPU-bound (typically 4-16) and `journal_mode=WAL` with `synchronous=NORMAL` must be preserved.
- **Authentication**: JWT authentication uses token versioning (`token_version`). Validating token version in middleware guarantees instant logout across all devices.

---

## 🛑 5. Agent Workflow & Feature Proposal

- **Feature Proposals**: Do not propose or re-implement features that already exist in the codebase (e.g., Metadata Fetching/Scraping is already handled by frontend; Cross-device sync, Reviews/Ratings, Share Links are already built-in).
- **Core Engine Mechanics**: The codebase already handles **Chunked Uploads** (for files >100MB), **Smart Garbage Collection** for orphaned uploads, and **Native Audio Streaming** (MP3, M4B, FLAC) without FFmpeg/HLS. Do NOT propose adding FFmpeg or HLS back into the project.
- **Codebase Review**: You **MUST** thoroughly search the codebase and database schemas (`db/schema/*.sql`) to verify if a feature or schema exists before planning to build it.

