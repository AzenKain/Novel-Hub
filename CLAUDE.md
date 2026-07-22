# NovelHub Project Rules & Guidelines (Claude Code)

Please refer to `AGENTS.md` for full system architecture rules.

## Core Rules Summary
- **Architecture Boundaries**: Controller <-> Service <-> Repository <-> DB/RAM Cache.
- **Controllers (`internal/controllers`)**: HTTP binding, validation via `pkg/validator` (`ValidateBodyDto`/`ValidateQueryDto`), response mapping. NO business/DB logic.
- **Services (`internal/services`)**: Business rules, DTO inputs/outputs (`internal/dtos`), split large service files (`bookService_reader.go`, etc.). Multi-mutation DB operations MUST use atomic transactions via `pkg/database/transaction.go`. NO 1-line wrapper functions.
- **Repositories (`internal/repositories`)**: Persistence & `theine-go` RAM Cache. Mandatory **Cache-by-IDs** pattern & **Singleflight protection (`sfg.Do(...)`)** on all DB fallback read operations. Accept `sqlc` params, return types MUST be converted to `internal/models`.
- **Domain Models (`internal/models`)**: Provide `FromSqlc` and `ToResponse` mapping helpers.
- **Shared Packages (`pkg/`)**:
  - `pkg/apperrors`: All custom errors (`ErrBadRequest`, `ErrNotFound`, etc.).
  - `pkg/convert`: ID parsing & null conversion helpers.
  - `pkg/constants`: Cache TTLs and app constants.
  - `pkg/jsonx`: MUST use `jsonx.Marshal`/`jsonx.Unmarshal` (never import std json or sonic directly).
  - `pkg/localfs`: Use `localfs.SafeJoin` (powered by `filepath.Rel`) for all file path resolution to prevent path traversal attacks.
  - `pkg/netx`: Use `netx.NewSafeHTTPClient` for external URL requests to prevent SSRF.
  - `pkg/validator`: Input validation for DTO struct bindings.
  - `pkg/worker`: Bounded worker pool queue for background tasks.
- **Frontend (`web/src` & `web/public`)**:
  - Permission utilities in `web/src/utils/permission.ts` (`isAdminUser`, `isModOrAdminUser`). Do NOT hardcode string role checks in components.
  - Translation files in `web/public/locales/` (`en.json`, `vi.json`, `ja.json`, `ko.json`, `zh.json`).
  - NO HARDCODED UI TEXT in TSX files. Always add keys to `web/public/locales/` and use `react-i18next` `t()`.
  - Types & Interfaces MUST be defined in `web/src/types/` and exported via `web/src/types/index.ts` (NO inline type definitions inside services/components).
  - Centralized API services in `web/src/services/` (Axios interceptor with auto 401 token refresh in `web/src/config/api.ts`). NEVER call direct `api.get` / `api.post` inside components or hooks.
  - Data fetching & server mutations MUST use TanStack React Query hooks in `web/src/hooks/` (`useQuery`, `useMutation`).
  - State management via Zustand stores in `web/src/stores/` (`useAuthStore`, `useThemeStore`, `useReaderSettingsStore`, `useSettingsAdminStore`, `useWebhookStore`). Transient form state in single modals may use `useState`.
  - Component modularization under `web/src/components/<domain>/` (~200-400 lines max).
  - Media URLs via `getMediaUrl(path)`.
  - TailwindCSS + DaisyUI styling.
- **Database (`db/query`)**: **ZERO INLINE / RAW SQL STATEMENTS**: Writing raw SQL strings or inline SQL in Go application logic/repositories is strictly forbidden. EVERY query for NovelHub DB MUST be defined in `db/query/*.sql` files and generated via `make sqlc` (except reading 3rd-party external SQLite files like Calibre `metadata.db` or reading `.sql` schema files from disk). Explicit column selection ONLY (no `SELECT *`). Max pagination limit = 100.
