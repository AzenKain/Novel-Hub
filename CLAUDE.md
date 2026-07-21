# NovelHub Project Rules & Guidelines (Claude Code)

Please refer to `AGENTS.md` for full system architecture rules.

## Core Rules Summary
- **Architecture Boundaries**: Controller <-> Service <-> Repository <-> DB/RAM Cache.
- **Controllers (`internal/controllers`)**: HTTP binding, validation via `pkg/validator` (`ValidateBodyDto`/`ValidateQueryDto`), response mapping. NO business/DB logic.
- **Services (`internal/services`)**: Business rules, DTO inputs/outputs (`internal/dtos`), split large service files (`bookService_reader.go`, etc.). Multi-mutation DB operations MUST use atomic transactions via `pkg/database/transaction.go`. NO 1-line wrapper functions.
- **Repositories (`internal/repositories`)**: Persistence & `theine-go` RAM Cache. Mandatory **Cache-by-IDs** pattern & `singleflight`. Accept `sqlc` params, return types MUST be converted to `internal/models`.
- **Domain Models (`internal/models`)**: Provide `FromSqlc` and `ToResponse` mapping helpers.
- **Shared Packages (`pkg/`)**:
  - `pkg/apperrors`: All custom errors (`ErrBadRequest`, `ErrNotFound`, etc.).
  - `pkg/convert`: ID parsing & null conversion helpers.
  - `pkg/constants`: Cache TTLs and app constants.
  - `pkg/jsonx`: MUST use `jsonx.Marshal`/`jsonx.Unmarshal` (never import std json or sonic directly).
  - `pkg/netx`: Use `netx.NewSafeHTTPClient` for external URL requests to prevent SSRF.
  - `pkg/validator`: Input validation for DTO struct bindings.
  - `pkg/worker`: Bounded worker pool queue for background tasks.
- **Frontend (`web/src` & `web/public`)**:
  - Translation files in `web/public/locales/` (`en.json`, `vi.json`, `ja.json`, `ko.json`, `zh.json`).
  - NO HARDCODED UI TEXT in TSX files. Always add keys to `web/public/locales/` and use `react-i18next` `t()`.
  - Component modularization under `web/src/components/<domain>/` (~200-400 lines max).
  - State management via Zustand stores in `web/src/stores/`.
  - Centralized API services in `web/src/services/` (Axios interceptor with auto 401 token refresh in `web/src/config/api.ts`).
  - Media URLs via `getMediaUrl(path)`.
  - TailwindCSS + DaisyUI styling.
- **Database (`db/query`)**: Explicit column selection ONLY (no `SELECT *`). Max pagination limit = 100.
