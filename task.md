# NovelHub — Audit đệ quy FE → BE

Đi từ tính năng trên FE → service call → route BE → service → repository → SQL.
Mọi mục dưới đây đã **đọc code verify tay**, kèm `file:line`. Mục nào chỉ nghi ngờ thì ghi rõ là nghi ngờ.

Mức độ: 🔴 nặng · 🟠 vừa · 🟡 nhẹ

**File này chỉ giữ việc chưa xong + quyết định cần chống làm lại.** Chi tiết của ~60 mục đã đóng qua phase 1–5 nằm trong git (`git show 6342023:task.md`) — giữ lại đây chỉ làm loãng thứ còn phải đọc.

---

## Đã làm gần đây

### 🟠 E1. Controller trả status thô thay vì `apperrors.HandleError` — [x]

Rule (`AGENTS.md`, `CLAUDE.md`): controller phải trả lỗi service qua `return apperrors.HandleError(c, err)` để lỗi domain map đúng HTTP status qua **một** đường code.

**Bug lớn nhất không nằm trong 219 chỗ đó, mà ở chính `HandleError`.** Nó `switch appErr.Err` — so sánh con trỏ, không dùng `errors.Is` — nên chỉ nhận ra `*AppError` của chính nó. Repository trả `sql.ErrNoRows` **thô** theo convention (không repo nào import `apperrors`), và `bookService.GetBook` truyền thẳng lên. Hệ quả đo được bằng HTTP thật:

```
GET /api/v1/books/<id-không-tồn-tại>  →  500  {"message":"sql: no rows in result set"}
```

Sai status **và** ném văn bản driver ra ngoài. Tồn tại ở mọi endpoint gọi `GetBook`, kể cả những chỗ đã dùng `HandleError` "đúng".

**Sửa ở `HandleError`, không sửa từng repository** — một guard che hết mọi caller, kể cả những repo chưa ai rà. Đổi sang `errors.Is` (giữ được kind sau khi bọc `%w`) + thêm case `sql.ErrNoRows` → 404, và thay message bằng `"Not found"` vì văn bản driver không nói được gì client dùng được.

**Sửa kèm — nhóm flatten thật:**
- **`readerController` 4 chỗ** (`GetChapter`, `GetFile`, `GetAsset`, `ListImages`): `if err != nil || !CanReadBook(...)` gộp hai kết cục khác hẳn nhau thành một 403 *"You do not have access to this book"*. Sách không tồn tại báo 403 là sai status **và** sai thông điệp — nó hàm ý sách có tồn tại. `GetBootstrap` cùng file đã tách đúng từ đầu; 4 chỗ này là bản copy lệch.
- **`bookController.SearchInBook`**: lỗi đọc settings bị gộp vào check feature-flag → sự cố store báo thành *"in-book search is disabled by system administrator"*; và lỗi DB thật báo thành 404 *"Book not found"*. Hai lần kể sai chuyện cho người vận hành lúc đang có sự cố.
- **`file.Open()` fail → 400** ở `readerController.UpdateCover` + `settingsController.handleAssetUpload`: multipart đã parse xong, file server tự nhận rồi mở không được — lỗi phía server, 400 đổ cho client là sai.
- **`trackerController` 2 chỗ** trả `apperrors.New(...)` **trực tiếp** thay vì qua `HandleError` → Fiber render bằng handler mặc định thành 500, dù ý định là 400.
- **6 chỗ service trả `fmt.Errorf` trần** (`bookService_files.go` 3, `bookService.go` 2, `libraryService.go` 1) → `HandleError` ra 500. Đáng chú ý nhất là `libraryService.go:79`: nó **cố ý** báo "library not found" khi bị từ chối quyền (không tiết lộ library có tồn tại) — nhưng vì là error trần nên ra 500, mất luôn cả status lẫn ý định che giấu.

**Không đụng:**
- **`opdsController` 11 chỗ** trả `apperrors.New(ErrInternalError, ...)` trực tiếp — status 500 vốn đã đúng, và reader app cần **XML**; `HandleError` trả JSON.
- **~180 chỗ còn lại** — validation trả `Errors: errs` (`HandleError` không diễn đạt được field-error list), response thành công, và `uid`/`claims` thiếu → 401 (không có `err` nào để truyền).

**Test:** `TestMissingRowIsNotFoundNotServerError` — 7 route, assert 404 + envelope chuẩn + message **không** chứa `sql:`/`no rows`. Revert `HandleError` → đỏ `status = 500 ... "sql: no rows in result set"`; revert 4 chỗ tách ở `readerController` → đỏ `status = 403 ... "You do not have access to this book"`.

### 🟡 E5. `bookAdminStore.ts` gọi thẳng service từ Zustand — [x]

**Không chỉ lệch pattern — có bug thật.** Store giữ bản sao `libraries` riêng, fetch thẳng từ service qua `loadLibraries()`; mọi mutation gọi `invalidateQueries({ queryKey: ["libraries"] })` mà **không consumer nào của bản sao đó đọc key này**. Hệ quả: đổi tên library thì modal vừa đổi hiện đúng còn mọi nơi khác vẫn tên cũ; tạo library mới thì trang khác không thấy cho tới khi F5. Query `["libraries"]` có tồn tại (`useLibrariesQuery`) và được 3 nơi khác dùng — chỉ riêng trang admin Books đọc bản sao.

**Đã sửa:** thêm `useCreateLibraryMutation`/`useUpdateLibraryMutation`/`useDeleteLibraryMutation` vào `useLibraryQueries.ts` (file này **đã có sẵn** 3 mutation khác — báo cáo cũ ghi "chưa có mutation nào" là sai). Xoá `libraries` + `loadLibraries` + 3 handler khỏi store (522 → 445 dòng). `Books.tsx` đọc qua `useLibrariesQuery()`.
Xoá library còn dọn thêm: nếu library vừa xoá đang là filter hoặc đích upload thì reset về rỗng — logic này có trong handler cũ, dễ rơi mất khi chuyển.
i18n 6 key × 5 locale (3 lỗi + 3 thành công) theo đúng phân công sẵn có: hook lo `onError`, page lo toast thành công.

**Test:** `web/src/hooks/libraryListOwnership.test.ts` ghim *nơi danh sách sống* — đọc file bằng `?raw` của Vite chứ không import runtime (runtime không thấy được "state nằm ở đâu"), và không cần thêm `@types/node` cho một test. Bắt được thật: lần chạy đầu đỏ vì còn sót import `libraryService` chết trong store mà `tsc` không thấy. Revert 1 trong 3 invalidation → đỏ `expected 2 to be 3`; bỏ `useLibrariesQuery()` → đỏ.

---

## Deferral có ý thức — KHÔNG phải sót

### 🟠 B5. Không có CSRF ở đâu cả — [ ]
Auth bằng cookie + CORS `AllowCredentials: true`. `SameSite=Lax` là phòng tuyến duy nhất — chính comment `internal/controllers/authController.go:35` thừa nhận. `Secure` chỉ bật khi `c.Scheme()=="https"`, phụ thuộc `TRUST_PROXY` cấu hình đúng.

**Chưa sửa** vì đổi sẽ ảnh hưởng mọi client (kể cả app đọc sách ngoài). **Điều kiện phải làm:** còn nhận cookie auth cho request non-GET thì cần CSRF token hoặc origin check.

### 🟠 G2. `GET /kobo/v1/user/profile` trả identity giả cứng — [ ] **KHÔNG sửa**
calibre-web cũng không implement local (forward về store, trả `{}` khi tắt proxy), nên **không có payload đúng nào để copy**. Đã trả identity thật từ claims thay vì `novelhub-kobo-user` hardcode, nhưng key set là suy đoán và ghi rõ trong comment. Không có gì trong luồng sync đọc nó.

### 🟡 `LibraryController.DownloadLibraryZip` — [ ] **KHÔNG xoá**
Không có route, nhưng `StreamLibraryZip` đã được hardening (phân trang cursor 100, propagate lỗi copy/close). Tức là **tính năng chưa xong**, không phải code chết. Xoá là mất công đó; tự ý mở route thì thêm endpoint tải cả library mà không ai yêu cầu.

### ⚠️ Nhóm K (Kobo sync) — mức bảo đảm thật, nói thẳng

**Chưa test trên máy Kobo thật. Không có máy.** 17 contract test chứng minh **byte trên dây khớp shape calibre-web** và các cổng auth/permission hoạt động. Nó **không** chứng minh máy thật sync được. Đây là trần bảo đảm đạt được trong điều kiện hiện tại, không phải "đã chạy".

Còn thiếu so với calibre-web, **có ý thức**:
- **Tags/collections (5 endpoint)** — chưa làm. Máy vẫn sync sách bình thường, chỉ không có shelf.
- **`DELETE /v1/library/<uuid>`** (archive từ máy) — chưa làm. Xoá trên máy không phản ánh về server.
- **`ChangedReadingState`** dạng item độc lập — chỉ gắn `ReadingState` kèm entitlement. Sách đã sync rồi mà chỉ đổi progress thì lần sync sau không đẩy được state; đợi có máy thật xác nhận đây có phải vấn đề thực tế trước khi thêm.
- **Kobo store proxy** (`config_kobo_proxy`) — không làm. Tính năng gộp kết quả từ store thật, không cần cho self-hosted.
- **Sync scan cả trang thư viện rồi filter trong Go** — có comment `ponytail:` ghi rõ ceiling và đường nâng cấp (cursor theo timestamp trong sync token).

### 🟡 C6. `CountJobs` scan toàn bảng — **đo xong, quyết định KHÔNG sửa**

Đo tại **500k row** bằng `EXPLAIN QUERY PLAN` + timing thật:

| Query | Plan | Thời gian |
|---|---|---|
| `CountJobs` OR-guard, có filter status | `SCAN jobs` | 60ms |
| `CountJobs` dạng `narg IS NULL` | **vẫn `SCAN jobs`** | 45ms |
| `COUNT(*) WHERE status = ?` (không guard) | `SEARCH ... USING COVERING INDEX` | 7ms |
| `ListFilteredJobIDs` (LIMIT 50) | `SCAN jobs USING INDEX idx_jobs_created` | **0.13ms** |

Lý do không sửa, bằng số: đổi sang `narg` **không** giúp planner dùng index (vẫn `SCAN`, nhanh hơn 25%) mà phải đổi contract cả controller + service. Đường list — chạy thường xuyên hơn nhiều — **đã tối ưu rồi** (0.13ms). `CountJobs` **có cache** (`job:count:*`, TTL 10 phút, singleflight) nên 60ms chỉ trả 1 lần mỗi 10 phút cho mỗi tổ hợp filter. Muốn xuống 7ms phải viết 4 query riêng cho từng tổ hợp filter — đổi lấy 53ms mỗi 10 phút.

**Ghi lại để lần sau không ai "tối ưu" lại rồi phát hiện vô ích.**

### 🟡 SQLite: WRITE trước READ trong transaction

Phát hiện khi làm C1 (mutex global chặn mọi lượt đọc). Bỏ mutex ra thì **không** mất lượt đếm như dự đoán — nó fail thẳng bằng `database is locked (517)` = `SQLITE_BUSY_SNAPSHOT`. Thí nghiệm riêng, 20 goroutine, DSN production:

| Thứ tự trong transaction | Lỗi |
|---|---|
| READ rồi WRITE (deferred, như code cũ) | **11/20** |
| WRITE trước | **0/20**, count chính xác |

Transaction `deferred` mở ở chế độ read-only rồi mới nâng cấp lên write ở câu lệnh ghi đầu tiên. Nếu giữa lúc đó có writer khác commit thì snapshot đã cũ → nâng cấp fail với 517, và **`busy_timeout` KHÔNG retry được** lỗi này (khác hẳn `SQLITE_BUSY` thường). **Quy tắc: ghi trước, đọc sau, trong cùng một transaction.**

---

## Đã kiểm tra và KHÔNG phải lỗi — đừng sửa lại

Verify tay, không chỉ tin báo cáo:

- **Chunked upload** — `internal/services/uploadService.go:395-450`. Commit verify từng chunk, cross-check size 3 lớp (`total != TotalBytes`, `storedChunks`, `storedBytes`), marker `O_EXCL` chống race, owner check (`:307`), chunk index bounds (`:346`), `SafeJoin` 12 site. **Đường code chắc nhất repo — không đụng.**
- **JWT refresh** — `internal/middlewares/jwtMiddleware.go:26,56,81` pin `JWTAlg: "HS256"` cả 3 đường kể cả refresh.
- **Highlight ownership** — `internal/services/highlightService.go:151,169` check `items[0].UserID != userID` + `canHighlight`. Chuẩn, dùng làm mẫu.
- **Collection ownership** — `featureService.go:564-578` check `CollectionOwnedByUser`; smart collection scope bằng SQL `WHERE id=? AND user_id=?`.
- **Reader XSS** — `web/src/utils/readerHtml.ts:9-15` DOMPurify, `FORBID_TAGS` gồm script/style/iframe/svg/math/link, `ALLOW_UNKNOWN_PROTOCOLS: false`.
- **SSRF** — 0 chỗ `http.Get`/`http.DefaultClient` trong `internal/`; đều qua `netx.NewSafeHTTPClient`.
- **SQL projection** — 0 chỗ `SELECT *` / `RETURNING *` trong `db/query/`.
- **CBZ/TAR archive budget** — đủ cả 2 chiều, cả 2 đường.
- **OPDS guest fallback** — `internal/routes/opdsRoutes.go:21-32` có gate bằng setting `GuestLoginRequired`, vẫn lọc `opds.read` per-library. Cố ý.
- **`GET /library/stats` public** — chỉ trả 3 con số tổng (`db/query/user_features.sql:1-6`). Ghi nhận, không sửa.
- **7/8 site `sql.ErrNoRows` còn lại trong `featureRepository`** — trả **giá trị zero có nghĩa** (`&BookReadStatsEntity{BookID}` = "0 lượt đọc") hoặc nil được caller đọc đúng (`GetBookmark` nil = chưa bookmark, `GetBookReview` nil = chưa review). Đổi thành lỗi là biến "chưa có dữ liệu" thành 404 ở màn hình chi tiết sách. Chỉ `GetReadingProgress` là bug thật (đã sửa ở P5-7).

---

## Bài học rút ra, áp cho lần sau

Ghi lại vì đều đã tốn thời gian một lần:

1. **Báo cáo audit lỗi thời nhanh hơn code.** P5-7: "19 chỗ `errors.Is(sql.ErrNoRows)` ở 10 file service" → grep ra **0**, đã xong từ phase 4; nhưng đúng lúc đó lại tìm ra một bug thật chưa ai báo (`GetReadingProgress` trả `(nil, nil)` làm nhánh `IsNotFound` thành code chết → Kobo nhận `ReadingState` rỗng cho **mọi** sách chưa mở). **Verify lại bằng grep/đọc code trước khi bắt tay, không tin mô tả.**

2. **Con số trong audit hay đếm sai đơn vị.** E3: "~140 cache key literal" đếm cả **tham số** của `BuildKey` (`"book"`, `"id"`) — cách dùng đúng; full-key literal thật chỉ **23**. E4: "62 key i18n trùng" gộp cả key ở section khác nhau; thật là **2**.

3. **Danh sách song song do người viết tay sẽ lệch.** P5-9: một setting key phải có mặt ở **4 danh sách rời nhau**, và `limits.rate_limit_api` đã sống sót nhiều tháng vì nằm trong `allowedSettingKey` mà thiếu ở `UnknownKeys()` → request bị chặn từ lúc decode JSON nên `allowedSettingKey` không bao giờ được hỏi tới. **Có danh sách song song thì viết test bắc cầu chúng.**

4. **Test phải đỏ trước khi tin nó xanh.** Mỗi fix trong phase 5 đều revert lại để xem test fail đúng lý do. Riêng loại "cấu hình không có tác dụng" thì kiểm bằng cách gọi thẳng service, bỏ qua tầng validate.

5. **Test cả hai chiều của một nhánh.** P5-9: test `server.url` phải có case **không** seed — một "fix" luôn dùng giá trị cấu hình sẽ phá mọi bản cài không đặt gì mà vẫn xanh nếu chỉ test chiều có seed.

6. **Sửa chỗ sinh ra vấn đề, không sửa chỗ biểu hiện.** `make build` ra `./api` còn docs 5 ngôn ngữ ghi `novelhub` → sửa Makefile một dòng, không sửa 10 chỗ trong docs.
