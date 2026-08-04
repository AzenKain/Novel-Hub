# NovelHub — Audit đệ quy FE → BE

Đi từ tính năng trên FE → service call → route BE → service → repository → SQL.
Mọi mục dưới đây đã **đọc code verify tay**, kèm `file:line`. Mục nào chỉ nghi ngờ thì ghi rõ là nghi ngờ.

Mức độ: 🔴 nặng · 🟠 vừa · 🟡 nhẹ

---

## A. Tính năng giả / hỏng hẳn

### 🔴 A1. SMTP là sân khấu, hai tầng — [ ]
- `web/src/components/admin/settings/SmtpSettingsTab.tsx:16-39` — `handleSave` và `handleTestConnection` có `try` block **rỗng**, chỉ gọi `toast.success`. Không có request nào. `catch` không bao giờ chạy tới.
- Component này **không được mount ở đâu** (grep 0 chỗ import).
- `internal/services/bookService_email.go:42-46` — host/user/pass hardcode chuỗi rỗng → `SendEmail` luôn fail "SMTP configuration incomplete".

**Hệ quả:** Send-to-Kindle chưa từng chạy được. Admin bấm Save thấy báo thành công.
**Sửa:** thêm SMTP vào app settings (schema mới + `db/query/settings.sql` + `settingsService`), đọc config trong `bookService_email.go`, nối tab vào `useAdminSettingsQuery`, mount vào `pages/admin/Settings.tsx`, thêm endpoint test thật.

### 🔴 A2. Phân trang admin books chết — [x] *(gộp E5)*
- `web/src/stores/bookAdminStore.ts:186` destructure `page` rồi **không dùng**; luôn gửi `cursor: undefined, limit: 24`.
- `pages/admin/Books.tsx:321-341` vẫn render nút prev/next; `:69` refetch theo `[page]` → bấm "»" fetch lại trang 1.
- `hasMore` đoán bằng `length === 24`.

**Hệ quả:** admin kẹt ở 24 quyển, nút phân trang là giả.

**Đã sửa — bỏ list state khỏi Zustand, `useBooksQuery` (infinite) làm chủ cursor:**
- Store bỏ `books`/`loading`/`page`/`hasMore`/`loadData` (−70 dòng). Còn lại là modal + upload state, đúng phần Zustand nên giữ.
- `Books.tsx` dùng `useBooksQuery` + nút "Tải thêm" (`common.load_more`, 5 locale đã có sẵn) thay 2 nút prev/next giả. Thêm `useDebounce(search, 500)` — mỗi giá trị search là một query key riêng, không debounce là mỗi phím một request kèm cursor mới.
- Poll `refetchInterval` khi còn dòng `status === "processing"` (tiền lệ `useOperationsQueries.ts:13`), thay vòng `for attempt < 12 { loadData(); sleep(1000) }` trong store.
- 12 lời gọi `invalidateQueries` rời rạc → `invalidateBooks()`/`invalidateLibraries()`. Bỏ `loadData()` ở callback của `useCalibreImportMutation` và `BulkDeleteModal` — cả hai mutation đã tự invalidate `["books"]`.

**Dọn kèm (dead từ trước, cùng file):** `error`/`notice` + `setError`/`setNotice` được set nhưng **không render ở đâu** → lỗi tạo/xoá library im lặng; đổi sang `toast.error`. `closeEditModal`/`setCoverPreview` được select + destructure rồi không dùng (`setEditingBook(null)` mới là cái đóng modal) → xoá.

### 🔴 A2b. `next_cursor` tính sau khi lọc quyền → phân trang dừng sớm — [x]
`internal/controllers/bookController.go:81` lấy cursor từ `filtered` (sau `FilterReadableBooks`) thay vì `books` (trang thô từ DB). Một quyển bị lọc trên trang → `len(filtered) < limit` → `next_cursor = null` → client dừng **dù còn sách đọc được phía sau**.

Trước A2, FE đoán `length === 24` nên bug bị che. Sau A2, FE tin hẳn vào `next_cursor` → nó thành toàn bộ cơ chế phân trang, bug lộ ra thành mất sách.

**Test `TestBooksNextCursorSurvivesFilteredOutBooks`** — guest + 2 library (1 ẩn), 6 quyển xen kẽ, đi theo cursor như client. Fail trước khi sửa: *"Book 0"/"Book 2" was never returned* — **2 trong 3 quyển guest được phép đọc không thể tới được client**. Test seed settings **trước** `SetupServer` (`settingsService.Reload` chạy một lần lúc khởi động) và có sanity check "guest thấy đúng 3 quyển" — không có nó thì filter không áp dụng và test pass mà chẳng kiểm tra gì (đúng bẫy đã dính ở K6).

4 chỗ `next_cursor` khác (`featureController.go:84/224/443/603`) tính thẳng từ kết quả service, không có bước lọc sau query → không dính. Đã kiểm, không sửa.

### 🟠 A3. Facet A–Z lọc trên 20 dòng — [x]
- `web/src/services/metadataService.ts:123-131` gọi `/metadata/authors` **không truyền limit** → BE default 20.
- `pages/library/LibraryWorkspace.tsx:468-476` search + lọc A–Z + sort **client-side** trên 20 dòng đó.

**Hệ quả:** thư viện lớn, bấm "Z" ra rỗng dù có tác giả vần Z.

**Đã sửa — đẩy `search` + `alpha` xuống SQL** cho cả 6 facet, FE thêm "Tải thêm" theo cursor. BE vốn đã có cursor + limit + `next_cursor` đầy đủ; bug thuần FE (không truyền param, rồi lọc trên tập cắt cụt).

**Chi tiết đáng ghi lại — sqlc thay placeholder theo BYTE offset.** `make sqlc` fail liên tục ở query *sau* query tôi sửa. Bisect 6 bước mới ra: SQL của tôi **hợp lệ** (SQLite chạy được), nhưng sqlc lệch vì **bất kỳ ký tự multi-byte nào trong file** — kể cả `Đ` trong **comment** tôi tự viết — cũng đẩy lệch mọi placeholder phía sau. Nên:
- `Đ`/`đ` truyền **từ Go** làm tham số, không viết literal trong `.sql`.
- Comment trong `db/query/metadata.sql` giữ **ASCII-only**, có ghi lý do để lần sau không ai vô tình thêm lại.
- Dùng `BETWEEN 'A' AND 'Z'` thay `GLOB '[A-Z]'` (sqlc cũng rối với `[...]` sau khi thay placeholder).

**Đúng semantics `Đ`:** SQLite `UPPER()` chỉ xử lý ASCII → **không** fold `đ` → `Đ`. Nếu làm hớ bằng `UPPER()` thuần thì tên chữ thường vần Đ sẽ rơi sai vào nhóm `#` và không bấm tới được. Xử lý riêng bằng cặp upper/lower truyền từ Go.

**Test:** `internal/repositories/metadataFacet_test.go` — 8 ca gồm `Đ`/`đ` cùng nhóm, `#` không lẫn `Đ`, alpha khớp cả tên chữ thường, search+alpha kết hợp, và cache key phải khác nhau theo filter (nếu không 2 filter dùng chung 1 entry → trả sai dữ liệu).
**2 ca đầu tôi viết kỳ vọng SAI, code đúng:** `Ong` bắt đầu bằng `O` ASCII nên thuộc nhóm `O` chứ không phải `#`; và `đỏ nam` không khớp `%o%` vì `ỏ` khác `o` (JS `includes()` cũng vậy). Đã sửa test theo thực tế, không sửa code.

**Dọn kèm:** xoá `metadataInitial()` ở FE (giờ SQL định nghĩa nhóm — code chết), `filterMetadataItems` → `sortMetadataItems` (chỉ còn sort), và bỏ dòng đếm `x / x` gây hiểu nhầm là "đã lọc bớt" (giờ hiện `50+` khi còn trang).

**Cũng sửa:** `LibraryWorkspace` trước đây query **cả 6 facet** ngay khi mở thư viện, dù người dùng chỉ xem 1. Giờ chỉ query facet đang mở.

---

## B. Bảo mật

### 🔴 B1. `GET /books/:id/user-state` không check quyền đọc book — [x]
`internal/services/featureService.go:405-435` — không gọi `PolicyAllowsBook`/`CanReadBook`, trong khi **mọi sibling** (`SetBookmark`, `RecordBookShare`, `UpsertBookReview`) đều có.
**Hệ quả:** user đăng nhập bất kỳ đọc được rating/social/download/read stats của book trong library họ không có quyền.
**Đã sửa:** thêm `PolicyAllowsBook(ctx, "read", ...)` **ở service** (không phải controller như sibling) để caller sau này không bypass được; controller truyền `claims` và đổi sang `apperrors.HandleError` (E1 luôn).
**Test:** `TestGetBookUserStateRequiresBookAccess`. Trước khi sửa test **panic** ngay tại `repo.GetBookmark` — đúng bằng chứng là nó chạm repository mà chưa qua bất kỳ check nào.

### 🔴 B2. `POST /trackers/map` + `/trackers/sync` ghi mapping toàn cục — [x]
- `internal/services/trackerService.go:317-323` upsert thẳng, không check book.
- `db/schema/59_user_trackers.sql:20` — `UNIQUE(book_id, provider)`, **không theo user**.

**Hệ quả:** ai có `tracker.sync` cũng ghi đè mapping AniList của book trong library họ không đọc được.
**Đã sửa:** `trackerService` không có bookRepo/permission nên check đặt ở controller qua `featureService.PolicyAllowsBook` (đã sẵn dependency). Thêm helper `bookReadable` **fail-closed** khi `featureService == nil` — check quyền mà panic hoặc pass ngầm thì tệ hơn là từ chối. Áp cho **cả hai** handler (`SyncProgress` cũng ghi mapping qua `GetOrMapBookTrackerID`).

### 🔴 B3. `PATCH /users/:id/restore` thiếu guard admin/root — [x]
- `internal/services/userService.go:438` `RestoreUser` **không nhận `claims`**.
- Đối chiếu `DeleteUser` (`:389,406-409`): chặn non-root xoá admin.

**Hệ quả:** người có `user.manage` khôi phục được admin đã xoá mềm → leo thang quyền.
**Đã sửa:** `RestoreUser` nhận `claims`, guard đối xứng với `DeleteUser` (chỉ owner mới restore được admin).
**Test:** `TestRestoreUserCannotResurrectAdminUnlessOwner` — non-owner bị chặn, owner vẫn làm được. Verify guard load-bearing: tạm bỏ guard → test fail đúng dòng đó.

### 🟠 B4. Calibre import đọc được thư mục bất kỳ trên host — [x]
- `internal/controllers/calibreController.go:37-43` chỉ `filepath.Clean` + chặn `.`/`/`.
- `internal/services/calibreSyncService.go:41` dùng chính path người dùng nhập **làm base** cho `SafeJoin` → containment vô nghĩa.

**Hệ quả:** `{"path":"/etc"}` → đọc bất kỳ `metadata.db` nào trên máy.
**Đã sửa:** thêm `resolveCalibreDir` neo mọi path vào root cấu hình được (`CALIBRE_IMPORT_DIR`, mặc định `DATA_DIR/calibre`). Path tuyệt đối chỉ chấp nhận nếu đã nằm trong root. Bỏ `filepath.Clean` thừa ở controller. Cập nhật hint i18n **cả 5 locale** + `docs/en/configuration.md` + `.env.example` — nếu không thì tính năng trông như bị hỏng.
**Test:** `TestResolveCalibreDirContainsCallerPath` (relative trong root OK, absolute trong root OK, `/etc` + `../..` + escape ra ngoài đều bị chặn).

### 🟠 B5. Không có CSRF ở đâu cả — [ ] *(quyết định, chưa sửa)*
Auth bằng cookie + CORS `AllowCredentials: true`. `SameSite=Lax` là phòng tuyến duy nhất — chính comment `internal/controllers/authController.go:35` thừa nhận. `Secure` chỉ bật khi `c.Scheme()=="https"`, phụ thuộc `TRUST_PROXY` cấu hình đúng.

**Chưa sửa trong lượt này** vì đổi sẽ ảnh hưởng mọi client (kể cả app đọc sách ngoài). Điều kiện để phải làm: còn nhận cookie auth cho request non-GET thì cần CSRF token hoặc origin check.

---

## C. Hiệu năng

### 🔴 C1. Mutex global chặn mọi lượt đọc của mọi user — [x]
`internal/services/featureService.go:161-162` — `activityMu.Lock()` giữ **suốt cả transaction**, phạm vi toàn process: user A cuộn trang chặn user B.

**Điều tra ra nguyên nhân sâu hơn dự đoán ban đầu.** Bỏ mutex ra thì **không** mất lượt đếm như tôi tưởng — nó fail thẳng bằng `database is locked (517)` = `SQLITE_BUSY_SNAPSHOT`. Dựng thí nghiệm riêng để tách bạch (20 goroutine, DSN production):

| Thứ tự trong transaction | Lỗi |
|---|---|
| READ rồi WRITE (deferred, như code cũ) | **11/20** |
| WRITE trước | **0/20**, count chính xác |

Lý do: transaction `deferred` mở ở chế độ read-only rồi mới nâng cấp lên write ở câu lệnh ghi đầu tiên. Nếu giữa lúc đó có writer khác commit thì snapshot đã cũ → nâng cấp fail với 517, và **`busy_timeout` KHÔNG retry được** lỗi này (khác hẳn `SQLITE_BUSY` thường).

Nghĩa là mutex không chỉ bảo vệ counter — nó đang **che một bug transaction ordering**.

**Đếm phạm vi thật (không suy diễn):** trong 25 chỗ `BeginTx`, chỉ **3 chỗ** read trước khi write, tức chỉ 3 chỗ dính lỗi này:

| Chỗ | Lệnh DB đầu tiên | Ghi chú |
|---|---|---|
| `featureService.go:161` | `GetReadingProgress` | chỗ có mutex che |
| `authService.go:421` (`Logout`) | `GetByID` | **verify bằng test: fail thật** khi bỏ `_txlock` |
| `calibreSyncService.go:119` | `GetAuthorByName` | import chạy 1 transaction/sách nên dễ gặp khi có ghi song song |

22 chỗ còn lại write ngay lệnh đầu (`CreateBook`, `UpdatePassword`, `DeleteFile`...) → vốn đã an toàn.

**Đã sửa 2 tầng:**
1. `pkg/database/db.go` — thêm `_txlock=immediate` vào DSN: lấy write lock ngay tại `BEGIN`, biến race thành hàng đợi mà `busy_timeout` xử lý được. Chặn cả lớp bug, kể cả code viết sau này. Với 22 chỗ write-first thì gần như không đổi thời điểm lấy lock (chúng vốn lấy ngay ở lệnh đầu), nên chi phí thực tế chỉ nằm ở 3 transaction read-first — đổi lại chúng hết fail 517.
2. `db/query/user_features.sql` — `opened_count`/`qualified_read_count` chuyển thành **delta cộng trong SQL** (`reading_progress.opened_count + excluded.opened_count`), y hệt `UpsertBookReadStats` đã làm. Service truyền delta, không tính absolute nữa.
3. Xoá `activityMu` + import `sync`.

**Test:** `TestRecordReadingActivityCountsEveryConcurrentOpen` — 20 goroutine cùng ghi 1 book, `opened_count` phải đúng 20. Harness dùng `database.NewSQLiteDB()` chứ không `sql.Open` trần, vì chính pragma production (busy_timeout/WAL) mới quyết định hành vi này. Verify: bỏ mutex mà chưa sửa → fail 517; sau khi sửa → pass `-race -count=5`.
Thêm `TestConcurrentLogoutsDoNotHitBusySnapshot` cho `authService.Logout` (12 thiết bị logout đồng thời) — verify bỏ `_txlock=immediate` là fail ngay, tức chỗ này dính bug thật chứ không phải suy diễn.

**Ghi chú lỗi phụ phát hiện kèm:** `authService.Logout` bọc lỗi thành `apperrors.New(ErrInternalError, "Failed to revoke session")`, **nuốt mất** nguyên nhân 517 — nên bug này trước đây không thể debug từ log. Thuộc nhóm E1.

### 🟠 C2. Mỗi 2 giây cuộn trang wipe cache toàn cục — [x]
- BE `internal/repositories/featureRepository.go:413` — `UpsertBookReadStats` gọi `DelByPattern("book:search*")`.
- `pkg/cache/ramcache.go:131-144` — `DelByPattern` là **quét O(n) toàn bộ cache**.

**Đo được:** benchmark `DelByPattern` → 0.057ms @1k entry, 0.72ms @10k, **5.7ms @50k**, tuyến tính. Mỗi lượt ghi progress (2 giây/lần khi cuộn, mỗi lần lật trang) đều trả giá này.

**Điều chỉnh đánh giá ban đầu (tôi đã nói quá phần FE):** `invalidateQueries` của React Query **chỉ refetch query đang active**. Trong route reader chỉ có `['highlights', chapter_id]` mounted — `["books"]`/`["library"]` **không** refetch. Nên không có "refetch cả thư viện mỗi 2 giây" như tôi viết lúc đầu. Chi phí thật nằm ở BE.

Phần FE vẫn có vấn đề nhưng nhẹ hơn: đánh stale toàn bộ `["books"]` mỗi lần lật trang → **ném sạch cache list**, nên quay về thư viện là fetch lạnh mỗi lần.

**Đã sửa:**
- BE: `book:search*` chỉ chứa **danh sách ID**, và `book_read_stats` là bảng khác. Filter `nav=hot` là predicate boolean (`total_open_count > 0`) → sách chỉ **vào** danh sách hot ở lần mở **đầu tiên**, sau đó không đổi thành viên. Nên chỉ invalidate khi đúng lần chuyển đó (đọc row trước khi ghi để biết). Lượt mở thứ 2 trở đi: 0 lần quét cache.
- FE: gom thành `invalidateProgressQueries()` chỉ đánh `["reading"]`, `["trackerReadingProgress"]`, `["bookUserState", book_id]` (thêm scope theo book). `["books"]`/`["library"]` chuyển sang chỉ invalidate **một lần khi unmount** reader.
- Thêm `constants.CacheKeyBookSearchPattern` + `CacheKeyLibraryStats` thay literal (E3 một phần).

### 🟠 C3. `webhookRepository` singleflight chết hoàn toàn — [x]
- `internal/repositories/webhookRepository.go:30` giữ `sf singleflight.Group` **by value**.
- `:33` constructor không init.
- `:38` `WithTx` dựng struct mới **bỏ luôn** group.

Mọi repo khác giữ `*singleflight.Group` và thread qua. Group zero-value → dedup không tác dụng. Cũng là copylocks hazard cho `go vet`.
**Đã sửa:** đổi sang con trỏ, init trong constructor, thread qua `WithTx`. Thêm luôn `tx == nil` guard mà repo này thiếu (sibling đều có).

### 🟠 C4. Thiếu `inTx` → đọc cache cũ bên trong transaction — [x]
`internal/repositories/featureRepository.go:78-88` `WithTx` không set cờ `inTx`, và `GetReadingProgress` đọc cache không guard trong khi `RecordReadingActivity` gọi nó **qua** `txRepo`.

**Phát hiện khi viết test:** lần đầu test **pass** dù chưa sửa. Lý do: đường ghi gọi `Del` không điều kiện, nên write trong transaction đã evict key → read sau đó miss cache. Tức bug chỉ bị che **khi** mỗi read có write cùng key đi trước trong cùng transaction. Transaction read-first (đúng kiểu `RecordReadingActivity` và `authService.Logout`) không có eviction đó. Đổi test sang ghi bằng raw `tx.ExecContext` để dựng đúng hình dạng read-first → **fail thật** (`opened_count = 1`, muốn 2).

**Đã sửa:** thêm `inTx` cho `featureRepository`, `highlightRepository`, `jobRepository`, `jobScheduleRepository`, `libraryRepository`, `trackerRepository`, `webhookRepository`; gate toàn bộ 17 cached read của featureRepository + read của 6 repo kia. Sweep lại cả `internal/repositories/` còn phát hiện `roleRepository` (4 chỗ) và `settingsRepository` (3 chỗ) — hai repo này **có** `inTx` nhưng chỉ gate một phần, audit ban đầu bỏ sót. Giờ 0 cached read nào ungated.
Chỉ gate **read**, không gate invalidation — `Del` vẫn chạy vô điều kiện (pessimistic, an toàn khi rollback).
Thêm `tx == nil` guard cho `trackerRepository.WithTx`.

### 🟠 C5. `queryInChunks` chỉ dùng 5/13 chỗ — [x]
**Đã sửa:** bọc thêm `GetLibrariesByIDs`, `GetHighlightsByIDs`, `GetJobsByIDs`, `GetJobSchedulesByIDs`, `GetCollectionsByIDs`, `GetAuthorsByIDs`, `GetRolesByIDs`, `GetUserTrackersByIDs`, `GetBookTrackerMappingsByIDs`, `GetAppSettingsByKeys`, `GetPermissionsByKeys`, `GetRolePermissionsByIDs` (×2) → 18 call site dùng chunk.

**Đo được số thật:** giới hạn driver là **32767** bound params (`too many SQL variables` ở 32768, OK ở 32766), không phải 8000. `sqliteMaxSliceArgs = 8000` là biên an toàn dưới ngưỡng đó, không phải ngưỡng lỗi.
**Test:** `TestGetHighlightsByIDsBeyondBindParameterLimit`, 33000 id. Phải chỉnh 2 lần mới đúng: bản đầu dùng 8500 id (dưới ngưỡng thật → pass dù chưa sửa); bản hai gọi `GetByChapter` (cold cache đi `GetHighlightsByChapter` chỉ 2 param, không chạm IN-clause). Bản cuối gọi thẳng `GetHighlightsByIDs` → bỏ chunk là fail đúng `too many SQL variables`.

### 🟡 C6. `CountJobs` vẫn scan toàn bảng — [x] *(đo xong, quyết định KHÔNG sửa)*
`db/query/operations.sql:1-4` dùng OR-guard. Đã đo tại **500k row** bằng `EXPLAIN QUERY PLAN` + timing thật:

| Query | Plan | Thời gian |
|---|---|---|
| `CountJobs` OR-guard, có filter status | `SCAN jobs` | 60ms |
| `CountJobs` dạng `narg IS NULL` | **vẫn `SCAN jobs`** | 45ms |
| `CountJobs` OR-guard, không filter | `SCAN jobs` | 21ms |
| `COUNT(*) WHERE status = ?` (không guard) | `SEARCH ... USING COVERING INDEX idx_jobs_status` | 7ms |
| `ListFilteredJobIDs` (LIMIT 50) | `SCAN jobs USING INDEX idx_jobs_created` | **0.13ms** |

**Kết luận: không sửa.** Lý do bằng số, không phải phỏng đoán:
1. Đổi sang `narg` **không** giúp planner dùng index — vẫn `SCAN`, chỉ nhanh hơn 25% (60→45ms). Không đáng đổi contract (DTO đang dùng chuỗi rỗng, đổi sang NULL phải sửa cả controller + service).
2. Đường list — chạy thường xuyên hơn nhiều — **đã tối ưu rồi**: 0.13ms nhờ `idx_jobs_created` dừng sớm ở LIMIT.
3. `CountJobs` **có cache** (`job:count:*`, TTL 10 phút, có singleflight) nên 60ms chỉ trả 1 lần mỗi 10 phút cho mỗi tổ hợp filter, không phải mỗi request.
4. Muốn xuống 7ms thì phải viết query riêng cho từng tổ hợp filter (4 biến thể) — thêm code, thêm chỗ sai, đổi lấy 53ms mỗi 10 phút. Không đáng.

Ghi lại để lần sau không ai "tối ưu" lại rồi phát hiện vô ích.

### 🔴 C7. `location_cfi` / `location_type` chưa từng được ghi xuống DB — [x] *(phát hiện khi làm K6)*
`internal/repositories/featureRepository.go:380` dựng `sqlc.UpsertReadingProgressParams` mà **không set `LocationCfi` và `LocationType`**, dù cột tồn tại, `db/query/user_features.sql:81-82` có ghi, và `featureService` truyền đúng giá trị xuống. Zero value của `sql.NullString` là invalid → cột luôn `NULL`.

Bug từ `9c11dfb` — chính commit thêm cột location tracking. Ảnh hưởng **mọi reader, không riêng Kobo**: mở lại sách chỉ về đúng chương, không về đúng vị trí trong chương. Không ai báo vì mất-vị-trí trông giống "app cuộn về đầu chương" hơn là bug.

**Đã sửa** ở repo — một chỗ, mọi caller (web reader, Kobo, sync API) hết luôn. Không tìm ra bằng đọc code: tìm ra vì test Kobo PUT state đọc lại `Location` và nhận `nil`.

---

## D. Lỗi im lặng

### 🔴 D1. Highlight fail không báo gì — [x]
`web/src/hooks/useHighlights.ts:58-78` — cả create/update/delete chỉ `console.error`, trả `null`. Người dùng bôi highlight, mất, không biết gì.
**Đã sửa:** thêm `toast.error` cho cả 3. Hook không phải component nên dùng `i18n.t` (import default từ `@/i18n`) thay vì `useTranslation`.

### 🟠 D2. Ghi progress fail im lặng — [x]
`web/src/pages/reader/ReaderWorkspace.tsx:512,545` `.catch(console.debug)`.
**Đã sửa:** gom thành `reportProgressFailure` — **cảnh báo 1 lần / session** bằng ref, không toast mỗi lần. Vì write này chạy mỗi lần lật trang và mỗi 2s khi cuộn, toast mỗi lần sẽ thành spam; nhưng nuốt hết thì người đọc mất chỗ đang đọc mà không hay.

### 🟠 D3–D5. Mutation thiếu `onError` — [x]
- `pages/user/UserProfile.tsx` — `handleSave` thêm `onError` + toast; `handleChangePassword` thêm `onError` đổ vào `passwordError` **có sẵn** (sai mật khẩu hiện tại là ca phổ biến, nên hiện cạnh field chứ không phải toast).
- `hooks/useLibraryQueries.ts` — thêm `onError` mặc định vào `useCreateSmartCollectionMutation` + `useDeleteSmartCollectionMutation`. Sửa **ở hook** chứ không ở call site: hook dùng chung, sửa 1 chỗ là mọi consumer được, và caller vẫn override được nếu cần.
- `hooks/useReadingStats.ts` — `useUpsertReadingGoalMutation` thêm `onError`.
- `useReadingStats` phần sync session: **cố ý không toast** — counter không bị trừ khi fail nên vòng sau tự gửi lại; đã thêm comment giải thích thay vì để trống.

**i18n:** thêm 9 key × 5 locale (`reader.highlight_*`, `reader.progress_sync_failed`, `library.smart_collection_*_failed`, `analytics.goal_save_failed`, `user.profile_update_failed`, `user.password_change_failed`).

**⚠️ Phát hiện kèm — key trùng trong file locale.** Lần đầu tôi định thêm key bằng script python (sai, vi phạm rule sửa tay), và `json.load` **gộp mất key trùng** → đổi ngầm giá trị nào thắng. Đã `git checkout` revert và làm lại bằng Edit từng file. Bản thân chuyện key trùng là bug riêng → đã sửa ở E4 (2 key/file, không phải 62 như tôi đếm sai lúc đầu).

---

## E. Tuân thủ rule (diff to nhưng cơ học)

### 🟠 E1. Controller trả status thô thay vì `apperrors.HandleError` — [~] *(phần nguy hiểm đã xong)*

**Đã sửa — middleware (ảnh hưởng MỌI 403 toàn app):** `internal/middlewares/roleMiddleware.go` trả `fiber.ErrForbidden` trần → body là chuỗi `Forbidden`, **không phải JSON**. Mà 403 là lỗi API sinh ra nhiều nhất, tức response FE gặp thường xuyên nhất lại là cái nó không parse được. Đổi hết 17 chỗ sang `apperrors`. Tiện tay sửa nil-deref tiềm ẩn ở `BookLibraryAttr` (`book.LibraryID` khi `book == nil` mà `err == nil`).
**Test:** `TestPermissionDeniedUsesStandardEnvelope` — verify trước khi sửa fail đúng lý do: `403 body is not JSON at all: invalid character 'F'` (chữ `Forbidden`).

**Đã sửa — `featureController` (13 chỗ nuốt lỗi service thành 400 cứng):** mất hết `ErrNotFound`/`ErrForbidden`. Phải sửa 2 tầng vì service trả `fmt.Errorf` trần thì `HandleError` sẽ ra 500:
1. `featureService`: 21 chỗ `fmt.Errorf` → `apperrors.New(ErrBadRequest, ...)` (đều là lỗi validation).
2. `featureController`: 13 chỗ → `apperrors.HandleError`. Còn 0 chỗ flattening, `HandleError` từ 19 → 32.

**Đã sửa — `libraryController.go:120-124`:** map **mọi** lỗi upload thành 404 "Library not found" — đĩa đầy hay lỗi quyền cũng báo là không tìm thấy library. Đổi sang `HandleError`, và `libraryService.UploadFiles` bọc `sql.ErrNoRows` → `ErrNotFound` để 404 thật vẫn ra 404.

**Còn lại (chưa làm):** ~180 chỗ `Status(fiber.Status...)` khác, chủ yếu là nhóm "uid/claims missing" trả 401 và validation trả 400 — **nhất quán về hình dạng** (đều dùng `CommonResponse`) nên không phải bug, chỉ là lệch pattern. Ưu tiên thấp, có thể dọn dần.

### 🟡 E2. `metadataController.go:30-32` — [x]
Dùng `c.Query` + `strconv.Atoi` thay vì `ValidateQueryDto`, chỉ chặn cận dưới (`limit=100000` qua được controller; service clamp 100 nên không thủng, nhưng sai pattern).
**Đã sửa:** 6 handler giống nhau y hệt → gom thành `listFacet(c, fetch)` dùng `request.PaginationDto` (đã có sẵn `Cursor` + `Limit` `max=100`). Bỏ `strconv`. File 113 → 67 dòng, hết cả lỗi gofmt.

### 🟡 E3. Cache key literal — [x]
**Con số thật nhỏ hơn audit nói:** "~140 literal" đếm cả **tham số của `BuildKey`** (`"book"`, `"id"`, `"user"`…) — thứ đó là cách dùng **đúng**, không vi phạm gì. Full-key literal thật: **23 chuỗi riêng biệt** qua ~120 call site.

**Đã sửa:** thêm 21 constant vào `pkg/constants/cache_keys.go`, thay hết literal ở 10 repository. Còn **0** full-key literal.
Dọn luôn chỗ audit chỉ ra: `roleRepository` và `userRepository` invalidate `user:search*` bằng 2 cách khác nhau (constant vs literal) — giờ cả hai dùng `constants.CacheKeyUserSearch`.
**Test:** `pkg/constants/cache_keys_test.go` pin **giá trị chuỗi** của cả 23 constant đúng như trước khi thay. Đây là loại lỗi không có test thì không ai biết: gõ sai 1 ký tự thì key đó **không invalidate nữa**, không có gì fail, cache chỉ stale mãi.

### 🟡 E4. i18n + role literal + key trùng — [x]
- **Key trùng — [x].** Con số thật **2 key/file**, không phải 62 như tôi ghi lúc đầu (đếm sai: gộp cả key ở **section khác nhau**, thứ đó bình thường). Hai key thật: `admin.actions`, `admin.delete_selected`, trùng trong **cùng** object.
  Bug thật: `pages/admin/Books.tsx:196` xin "Delete selected" (xoá **sách**) nhưng parser lấy bản cuối nên nhận "Delete Selected Files" (chữ trang **duplicates**). Đã tách `admin.delete_selected_books`, xoá bản chết, sửa call site. 5 locale còn 0 trùng.
- **Role literal — [x].** `role.name === "USER"` → `role.auto_assign`, đúng cờ BE dùng để chọn role mặc định (`db/schema/90_seed_roles.sql`: USER có `auto_assign = 1`). Match theo tên là dựng lại semantic đã có, và vỡ ngay khi admin đổi tên role.
- **Doc lỗi thời — [x].** `CLAUDE.md` + `AGENTS.md` nhắc `isModOrAdminUser` (không tồn tại) → `isAdminUser`, `isBannedUser`, `hasPermission`.
- **i18n 3 trang — [x].** `RegisterPage.tsx` (11 key mới), `Reviews.tsx` (7 key), `Roles.tsx` (33 key, 649 dòng, trước đó **0** i18n). Tất cả × 5 locale, tái dùng key sẵn có (`common.cancel/delete/load_more/user`, `admin.operations.refresh`, `admin.save_changes`, `auth.email/password/back_to_library`). Câu có markup lồng bên trong dùng `<Trans>` chứ không nối chuỗi.

### 🟡 E5. `stores/bookAdminStore.ts` gọi thẳng 18 service từ Zustand — [~] *(một phần, cùng A2)*
600 dòng, bỏ qua React Query hoàn toàn, và **chính là** code gây ra A2 (phân trang chết).

**Đã làm cùng A2:** phần đọc danh sách sách chuyển hẳn sang React Query (`useBooksQuery`); store còn 530 dòng, không còn giữ bản sao dữ liệu server nào cho danh sách.

**Còn lại — chưa làm:** các mutation library/upload/metadata vẫn gọi service trực tiếp rồi `invalidateQueries` thủ công. Chuyển nốt sang `useMutation` cần thêm hook create/rename/delete library (`useLibraryQueries.ts` hiện chỉ có query, chưa có mutation nào) → là một job riêng, không nhét vào diff A2 được vì sẽ trộn hai thay đổi không review chung được. Upload có progress callback nên hợp với `useMutation` hơn là bị ép vào; cân nhắc lúc làm.

### 🟡 E6. Code chết / trùng — [x]
- **Trùng — [x].** `DELETE /admin/reviews/:bookId/:userId` khai báo 2 lần (`adminService.deleteReview` + `featureService.adminDeleteBookReview`), cùng URL cùng logic. Xoá bản ở `featureService`, `ReviewSection.tsx` chuyển sang `adminService.deleteReview`.
- **Context vứt đi — [x].** `jobController.go:57` `_, cancel := context.WithTimeout(...)` rồi không dùng. `ListTasks()` đọc map in-memory, không nhận context → bỏ hẳn.
- **`LibraryController.DownloadLibraryZip` — [ ] KHÔNG xoá.** Không có route, nhưng `StreamLibraryZip` **vừa được hardening trong chính đợt uncommitted này** (phân trang cursor 100, propagate lỗi copy/close). Tức là **tính năng chưa xong**, không phải code chết. Xoá là mất công đó; tự ý mở route thì thêm endpoint tải cả library mà không ai yêu cầu. Để nguyên, ghi lại quyết định.

---

## F. Comic archive budget

Đã verify: **CBZ và TAR đúng** — enforce cả entry count và uncompressed bytes, trên cả đường list và đường get asset.

### 🔴 F1. CB7 (7-Zip) không có budget nào cả — [x]
`pkg/bookparser/comic/comic.go:358-395` — `listSevenZipImages` và `getSevenZipAsset` **không gọi `archiveBudgetAdd`**, khác hẳn 3 format kia.
**Hệ quả:** file .cb7 nén bom (hoặc 1 triệu entry) đi thẳng qua.
**Đã sửa:** thêm `sevenZipArchiveBudget` dùng `file.UncompressedSize`, gọi ở **cả** hai đường.

### 🟠 F2. CBR chỉ đếm entry, không đếm byte — [x]
`comic.go:259,288` gọi `archiveBudgetAdd(&entries, &total, 0)` — truyền **0** nên `MaxArchiveUncompressedBytes` không bao giờ chạm tới.
**Đã sửa:** thêm `rarEntrySize` trả `header.UnPackedSize`, và **báo lỗi khi `UnKnownSize`** — coi size không biết là 0 thì lại tắt budget cho cả archive.

**Test:** `TestArchiveBudgetRejectsBombs` (cbz theo declared size, cb7 theo entry count — fixture 7z 2GB tốn 5s nên dùng 20k entry, 0.03s) + `TestArchiveBudgetAccumulatesEntrySizes` (drive `archiveBudgetAdd` trực tiếp, pin đúng lỗi "truyền 0"). Verify fail trước khi sửa: cb7 nhận archive ở **cả** ListImages và GetAsset. Verify không hồi quy: file `Dragon Ball v1.cbr` thật (260MB, 187 trang) vẫn parse được.

---

## G. OPDS / Kobo / KOReader

FE **không gọi** 3 nhóm route này (chỉ hiện URL để copy vào app đọc sách) → chưa từng được test qua UI.

### 🟠 G1. OPDS cắt cụt ở 50, không có link phân trang — [x]
`internal/controllers/opdsController.go:57,69,97,122,147` hardcode `50`, không nhận `limit`/`cursor`, feed không phát link `next`.
**Hệ quả:** thư viện >50 quyển, app đọc sách chỉ thấy 50 quyển đầu mỗi author/series/tag và **không có cách nào biết là bị cắt** — phần còn lại không tới được qua OPDS.

**Đã sửa:** thêm `OPDSPageQuery{Cursor, Limit}` cho cả 7 feed; controller có `opdsPageQuery()` đọc `?cursor`/`?limit` (clamp `MaxPaginationLimit`, default `OPDSDefaultPageSize = 50`); feed phát `rel="next"` qua helper `appendNextLink`.

**Điểm tôi cố tình KHÔNG sao chép từ code có sẵn:** `bookController.go:81` build `next_cursor` từ `filtered` — tức **sau** lọc quyền. Đó là bug: 20 sách về, 5 bị lọc → cursor trỏ sách thứ 15, **bỏ mất 5 sách** ở trang sau. OPDS build cursor từ hàng cuối **DB trả về** (`nextCursor(books, limit)`). Ghi lại vì `bookController` vẫn còn lỗi này.

**Sửa kèm:** `visibleBooks(ctx, 200, ...)` ở 3 catalog feed — `SearchBooks` clamp `>100` về **20**, nên trang authors/series/tags thực tế dựng từ 20 sách và số "N books" mỗi tác giả **cũng sai**. Giờ dùng đúng limit request + có next link. Cũng `url.PathEscape` tên trong URL (tên chứa `?`/`#`/`&` trước đây sinh link méo).

**Test:** `internal/services/opdsPaging_test.go` — `TestOPDSFeedPagesThroughEveryBook` (7 sách, limit 3, phải đi hết 7, không trùng entry) + `TestOPDSLastPageHasNoNextLink` (trang cuối không có next, nếu không client lặp vô hạn). Verify: chặn `appendNextLink` → fail đúng `paged through 3 books, want all 7`.

### 🟠 G2. `GET /kobo/v1/user/profile` trả identity giả cứng — [ ] **KHÔNG sửa, revert**
`internal/services/koboService.go:53-62` trả `novelhub-kobo-user` / `user@novelhub.local`, bỏ qua người gọi.

**Tôi đã sửa rồi REVERT.** Lần đầu tôi đổi sang `UserKey: claims.UId`, thêm `UserID`, xoá `UserToken`/`UserEmail` — thuần suy đoán, chưa tra spec. Đó là **đổi wire format mà thiết bị vật lý đang đọc**, dựa trên cảm giác "identity giả trông sai".

**Tra ra (calibre-web `cps/kobo.py`):** `/v1/user/profile` **không được implement local** — nó route vào `redirect_or_proxy_request()`, forward về Kobo store, và trả **`{}`** khi tắt proxy. Tức **không có payload đúng nào để copy**; cả hình dạng cũ lẫn hình dạng mới của tôi đều là bịa. Field tôi thêm (`UserID`) và field tôi xoá (`UserToken`) đều không có cơ sở.

Không có spec, không có thiết bị để test → **để nguyên**, chỉ thêm comment ghi rõ đây là stub chưa verify. Cái identity placeholder là chuyện nhỏ so với G5 dưới đây.

### 🔴 G5. Tích hợp Kobo có khả năng cao **chưa từng chạy được** — [x] *(làm xong ở nhóm K)*

**Nguồn tham chiếu:** [calibre-web `cps/kobo.py`](https://github.com/janeczku/calibre-web/blob/master/cps/kobo.py), [`kobo_auth.py`](https://github.com/janeczku/calibre-web/blob/master/cps/kobo_auth.py), [`SyncToken.py`](https://github.com/janeczku/calibre-web/blob/master/cps/services/SyncToken.py), [wiki Kobo Integration](https://github.com/janeczku/calibre-web/wiki/Kobo-Integration). calibre-web là implementation **đã được kiểm chứng trên thiết bị thật** bởi rất nhiều người dùng — không có máy Kobo thì đây là bằng chứng tốt nhất, và làm theo nó đáng tin hơn hẳn tự suy luận.

#### Khoảng cách hiện tại

| | Kobo thật cần | NovelHub |
|---|---|---|
| Resource ở `/v1/initialization` | **~147 key** (147 top-level trong `NATIVE_KOBO_RESOURCES()`) | **4 key** |
| Đường sync | `/v1/library/sync` | `/kobo/v1/library/sync` |
| Cover | `/<uuid>/<w>/<h>/<greyscale>/image.jpg` (+ biến thể có `Quality`) | **không có** |
| Auth | `/v1/auth/device`, `/v1/auth/refresh` (POST) | **không có** |
| Metadata | `/v1/library/<uuid>/metadata` | **không có** |
| Reading state | `/v1/library/<uuid>/state` (GET/PUT) | `/kobo/v1/library/state` (sai path) |
| Tags/collections | 5 endpoint | **không có** |
| SyncToken | header `x-kobo-synctoken`, base64(JSON), 5 mốc thời gian | **hằng số `"sync-token-1"`** |

#### Những gì học được, cụ thể

**1. Auth: thiết bị KHÔNG gửi credential.** Token nằm **trong URL path**: `https://host/kobo/<auth_token>/v1/library/sync`. `kobo_auth.py` sinh `hexlify(urandom(16))` (32 hex, `expiration = datetime.max`), user tự dán URL đó vào `api_endpoint` của máy. Sở hữu URL **chính là** credential — docstring nói thẳng về header `x-kobo-userkey`/Bearer thật: *"We pretty much ignore all of the above."*
→ **Ảnh hưởng lớn tới NovelHub:** hiện ta dùng `JwtAccess` + `RequirePermission`. Thiết bị Kobo **không gửi được `Authorization: Bearer`**, nên route hiện tại gần như chắc chắn trả 401 cho mọi request từ máy thật. Cần đổi sang token-trong-path như calibre-web.

**2. `/v1/auth/device` là dummy có chủ đích.** Trả 5 field: `AccessToken`, `RefreshToken` (mỗi cái `base64(urandom(24))`), `TokenType: "Bearer"`, `TrackingId: uuid4()`, `UserKey` (echo lại từ request body). Không verify gì sau đó — chỉ để máy đi tiếp bước login. Nên implement **theo đúng shape này**, không tự nghĩ.

**3. SyncToken — trái tim của incremental sync.** `VERSION = "1-1-0"`, `MIN_VERSION = "1-0-0"`, header `x-kobo-synctoken`, wire format = `base64(json)` của `{"version":..., "data":{...}}`. `data` có 6 field string: `raw_kobo_store_token`, `books_last_modified`, `books_last_created`, `archive_last_modified`, `reading_state_last_modified`, `tags_last_modified` — timestamp là **epoch seconds**. Nếu token chứa dấu `.` thì đó là token của Kobo store thật (`[b64].[b64]`), giữ nguyên không parse.
Chi tiết dễ sai: `books_last_created` **chỉ** được ghi vào token khi `not cont_sync`; 3 mốc kia luôn ghi.

**4. Continuation.** `SYNC_ITEM_LIMIT = 100`. Còn trang thì set header `x-kobo-sync: continue`. NovelHub hiện hardcode 100 không cursor, không header này.

**5. Book identity = UUID, dùng lại ở RẤT nhiều field.** `book_uuid = str(book.uuid)` gán cho cả `Id`, `CrossRevisionId`, `RevisionId` trong entitlement; và `CoverImageId`, `CrossRevisionId`, `EntitlementId`, `RevisionId`, `WorkId` trong metadata.
→ **Thuận lợi:** `books.id` của NovelHub đã là UUIDv7 string, khớp yêu cầu.

**6. Shape sync item.** Bọc `"NewEntitlement"` khi `ts_created > books_last_created`, ngược lại `"ChangedEntitlement"`. Mỗi cái chứa `BookEntitlement` + `BookMetadata`; `ReadingState` chỉ gắn kèm khi state mới hơn mốc token. Ngoài ra có `ChangedReadingState`, `NewTag`/`ChangedTag`/`DeletedTag`.
`BookEntitlement`: `Accessibility: "Full"`, `ActivePeriod{From}`, `Created`, `CrossRevisionId`, `Id`, `IsRemoved`, `IsHiddenFromArchive: false`, `IsLocked: false`, `LastModified`, `OriginCategory: "Imported"`, `RevisionId`, `Status: "Active"`. Timestamp format **`%Y-%m-%dT%H:%M:%SZ`**.
`BookMetadata`: `Categories` (1 GUID), `CoverImageId`, `CrossRevisionId`, `CurrentDisplayPrice{CurrencyCode:"USD",TotalAmount:0}`, `CurrentLoveDisplayPrice`, `Description`, `DownloadUrls[]{Format,Size,Url,Platform:"Generic"}`, `EntitlementId`, `ExternalIds:[]`, `Genre`, `IsEligibleForKoboLove`, `IsInternetArchive`, `IsPreOrder`, `IsSocialEnabled`, `Language`, `PhoneticPronunciations`, `PublicationDate`, `Publisher{Imprint,Name}`, `RevisionId`, `Title`, `WorkId`, `ContributorRoles`+`Contributors`, và `Series{Name,Number,NumberFloat,Id}` (Id = `uuid3(NAMESPACE_DNS, name)`).
NovelHub hiện chỉ trả `Title` + `Description` → thiếu gần hết, kể cả `DownloadUrls` (máy không biết tải ở đâu).

**7. Covers phụ thuộc initialization.** Máy **dẫn xuất URL cover từ template** trong resource map. calibre-web rewrite đúng 4 key: `image_host`, `image_url_template`, `image_url_quality_template`, `library_sync`. Template giữ placeholder `{ImageId}`/`{width}`/`{height}`/`{Quality}` (phải `unquote` sau `url_for`). Không trả 2 key image → **cover không bao giờ tải được**. Route cover chọn resolution theo `height > 1000` / `> 500`.
Cũng có bug hạ tầng đã biết: phải dùng `config_external_port` vì máy Kobo tạo HTTP request sai port — wiki gọi đây là *"a bug in HTTP-requests the Kobo reader creates, and can't be avoided"*.

**8. Format.** `KOBO_FORMATS = {"KEPUB": ["KEPUB"], "EPUB": ["EPUB3", "EPUB"]}`. Nếu chỉ có EPUB và có kepubify thì convert trước khi sync. NovelHub đã có `pkg/kepub` — dùng được.

#### Tại sao CHƯA code

Đây là **8 thay đổi liên quan nhau** trên một wire format tôi không test được, trong đó có **đổi cơ chế auth** (bỏ JWT sang token-trong-path) — tức đổi cả bề mặt bảo mật. Sửa một nửa thì vẫn không chạy, mà lại làm hỏng phần đang có. Và tôi vừa mắc đúng lỗi "đoán cho hợp lý" ở G2.

**Đề xuất:** làm thành một task riêng, theo sát calibre-web từng endpoint, kèm test dựng request giống thiết bị (fixture từ shape đã ghi ở trên) — không có máy thật thì test bằng contract là mức bảo đảm cao nhất đạt được. Trước khi làm xong, `KoboSyncCard.tsx` **không nên** hứa "Sync books, progress, and covers natively".

→ **Đã tách thành nhóm K bên dưới.**

---

## K. Kobo sync — làm lại theo calibre-web

Task riêng cho G5. Nguyên tắc: **mọi wire format copy từ calibre-web**, không tự nghĩ. Chỗ nào calibre-web cũng không có payload local (vd `/v1/user/profile`) thì ghi rõ là stub.

Mức bảo đảm cao nhất đạt được khi không có thiết bị: **contract test** — dựng request đúng như máy gửi, assert đúng shape đã đọc từ source.

### 🔴 K1. Auth token-in-path — [x]
Thiết bị Kobo có **đúng một** setting cấu hình được (`api_endpoint` trong `Kobo eReader.conf`) và **không gửi header `Authorization`** → `JwtAccess` chặn mọi request từ máy thật. calibre-web nhúng token random vào URL người dùng tự dán.

**Đã làm:**
- `db/schema/98_kobo_auth.sql` — `kobo_auth_tokens` (token PK, user_id, created_at, last_used_at) + `kobo_synced_books` (user_id, book_id, synced_at) để phân biệt `NewEntitlement` vs `ChangedEntitlement`.
- `db/query/kobo.sql` + `make sqlc` — 9 query, không SQL thô trong Go.
- `internal/repositories/koboRepository.go` — cache-by-ID + singleflight; `ResolveToken` là hot path (chạy mỗi request của máy).
- `internal/middlewares/koboAuthMiddleware.go` — resolve token → dựng `JWTClaims` như đường JWT, chặn user bị ban / bị xoá. Lỗi cố tình mơ hồ ("Invalid Kobo token") cho cả 2 nhánh: token là secret, phân biệt "không có" với "user đã xoá" là cho người ta dò.
- `internal/services/koboAuthService.go` — `hexlify(urandom(16))` giống calibre-web (128 bit hex, an toàn trong URL path). `RegenerateToken` xoá `kobo_synced_books` để máy thay thế nhận lại toàn bộ sách như mới.
- `internal/routes/koboRoutes.go` — mount `/kobo/:kobo_token`, đường `/v1/...` đúng như máy dẫn xuất từ resource map.

**Bề mặt bảo mật, ghi thẳng vì khác phần còn lại của API:** sở hữu URL **chính là** credential, không có yếu tố thứ hai, không hết hạn. Chỉ cấp được các route Kobo, chỉ cho user nó map tới. Thu hồi bằng regenerate (thay row) hoặc xoá.

**Verify:** `token_version` bump (`ChangePassword` / `AdminResetPassword` / `ChangeRoleUser`) **không** ảnh hưởng token Kobo — 2 credential độc lập. Đúng ý muốn: đổi mật khẩu không nên unpair máy đọc.

### 🔴 K3. SyncToken encode/decode — [x]
`pkg/kobo/synctoken.go` + `synctoken_test.go` (12 test). Mirror `cps/services/SyncToken.py`: `VERSION = "1-1-0"`, `MIN_VERSION = "1-0-0"`, header `x-kobo-synctoken`, wire = `base64(json)` của `{"version":…,"data":{6 field}}`, timestamp là **epoch seconds dạng string**.

Chi tiết dễ sai, đã cover bằng test: token chứa dấu `.` là token Kobo store thật (`[b64].[b64]`) → **giữ nguyên, không parse**; base64 thiếu padding vẫn phải decode được; epoch dạng `"1700000000.0"` (Python float) vẫn phải parse; version quá cũ → fallback token rỗng chứ không lỗi.

### 🔴 K2. `/v1/initialization` resource map — [x]
Máy **dẫn xuất mọi URL khác** từ map này. calibre-web trả ~147 key (`NATIVE_KOBO_RESOURCES()`), rewrite đúng 4: `image_host`, `image_url_template`, `image_url_quality_template`, `library_sync`. Thiếu 2 key image → **cover không bao giờ tải được**. NovelHub trước đây trả 4 key tổng cộng.

**Đã làm:** `pkg/kobo/kobo_resources.json` — **embed** 147 key trích thẳng từ calibre-web (`go:embed`), không gõ lại: 147 URL gõ tay là mời lỗi chính tả, mà một URL sai làm chết một tính năng của máy trong im lặng. `pkg/kobo/resources.go` rewrite 4 key về endpoint có token, giữ nguyên placeholder `{ImageId}`/`{width}`/`{height}`/`{Quality}` (máy tự thay, ta không được thay — calibre-web phải `unquote()` sau `url_for` đúng vì lý do này; dựng string trực tiếp thì bỏ được bước đó). Header `x-kobo-apitoken: e30=`.

**Bug test bắt được:** decode file vào `map[string]string` **im lặng mất cả 147 key** — 2 entry (`blackstone_header`, `free_books_page`) là object lồng nhau chứ không phải URL. Đã pin bằng test riêng.

### 🔴 K4. Sync response entitlements — [x]
`pkg/kobo/metadata.go` + `state.go`. `BookEntitlement` 12 field, `BookMetadata` ~22 field kể cả `DownloadUrls[]`. Timestamp `%Y-%m-%dT%H:%M:%SZ` (`FormatTimestamp`, zero-time → now chứ không phải năm 1 — calibre-web fallback đúng chỗ này). `SYNC_ITEM_LIMIT = 100`, còn trang thì set `x-kobo-sync: continue`.

`Series.Id` = `uuid3(NAMESPACE_DNS, name)` qua `uuid.NewMD5` (uuid3 chính là MD5). Yêu cầu duy nhất là **tính tất định** — máy nhóm sách theo id này. Giá trị mong đợi trong test lấy từ Python `uuid` thật, không đoán (lần đầu tôi viết bừa một GUID và nó sai).

**Trade-off ghi rõ:** NovelHub không có cột read-status tri-state như calibre-web, nên `StatusFor` **dẫn xuất** từ progress. Mất được "đánh dấu đã đọc xong ở 40%" — đổi lại không thêm cột chỉ Kobo mới ghi. Status máy gửi lên vẫn được ack cho tròn vòng, chỉ không lưu riêng.

**Bỏ sách không tải được:** sách không có `DownloadUrls` thì **skip khỏi sync**. Hiện lên rồi mở lỗi tệ hơn là không hiện.

### 🔴 K5. Handler còn lại — [x]
`AuthDevice` (dummy 5 field: `AccessToken`/`RefreshToken` = base64(24 byte random), `TokenType: Bearer`, `TrackingId` uuid4, `UserKey` echo lại — không verify gì sau đó, credential thật là token trong path). `GetBookMetadata` (trả **array** kể cả 1 sách). `GetReadingState`/`PutReadingState` map sang `reading_progress` qua `RecordReadingActivity` để counter và cache invalidation giống hệt phiên đọc trên web. `GetCoverImage` cả 2 biến thể URL.

**Lệch có ý thức:** cover **bỏ qua** segment width/height/quality/greyscale — NovelHub lưu 1 cover/sách, resize on-the-fly cần image pipeline không đáng cho tích hợp này. calibre-web chọn giữa 3 thumbnail dựng trước. Máy nhận ảnh lớn hơn thì tự scale.

`accessibleBook` là cổng duy nhất mọi endpoint đi qua (tồn tại + `CanReadBook` + `kobo.sync` theo library), để endpoint mới không quên 1 trong 3. Sách không có quyền trả **404 giống sách không tồn tại** — không cho dò.

### 🟠 K6. Contract test — [x]
`cmd/api/kobo_contract_test.go` — **17 test, dựng request đúng như thiết bị**: token trong path (không header `Authorization`), cursor trong header `x-kobo-synctoken`, body PascalCase. sqlite thật trong `t.TempDir()` + `database.ApplySchema`, không mock.

Cover: auth qua path token / từ chối token lạ / token của user đã xoá / 147 key resource + placeholder / auth dummy shape + random khác nhau mỗi lần / `NewEntitlement` đủ field / sync incremental (lần 2 trả 0) / body rỗng là `[]` không phải `null` / metadata array / 404 sách lạ / state cho sách chưa đọc / PUT state **đọc lại được** / body lỗi → 400 / cả 2 route cover / thiếu `kobo.sync` → 403 / user bị ban → 403 / `last_used_at` được cập nhật.

**3 bug test bắt được (đều fail trước, pass sau):**

1. **`LocationCfi`/`LocationType` chưa từng được ghi xuống DB** — `featureRepository.go:380` dựng `UpsertReadingProgressParams` mà **không set 2 field này**, dù cột có và SQL có ghi. Bug từ commit `9c11dfb` (commit thêm cột). Ảnh hưởng **mọi reader, không riêng Kobo**: mở lại sách chỉ về đúng chương, không về đúng vị trí trong chương. Sửa ở repo — một chỗ, mọi caller.
2. **Check ban của tôi là code chết** — `koboAuthMiddleware` check `role.IsBanned`, nhưng `GetUserRoles` (`db/query/users.sql:129`) **không select `is_banned`** nên field luôn `false`. Repo này ban **theo tên role `BANNED`** (`jwtMiddleware.go:120`, `authService.go:124`). Đã đổi sang `slices.Contains(claims.Roles, RoleTypeBanned)`.
3. **Test permission tự lừa chính nó** — `permissionCache.Reload()` chạy **một lần lúc startup**, nên sửa DB sau khi dựng app là vô hình. Fixture đổi sang seed **trước** `SetupServer`, qua hook.

### 🟡 K7. FE Kobo card — [x]
Trước đây `KoboSyncCard.tsx` hiện `${window.location.origin}/kobo` — **không có token, không thể ghép nối được**, và hứa "Sync books, progress, and covers natively" khi chưa chạy.

**Đã làm:** `internal/routes/koboRoutes.go` thêm `KoboSetupRoutes` mount dưới `/api/v1/kobo` với JWT thường (tách hẳn khỏi route token-in-path: trộn vào nhau thì URL thiết bị bị lộ cũng tự regenerate được). `GET /setup` (tạo token khi xem lần đầu — regenerate mỗi lần mở card sẽ âm thầm unpair máy đang chạy), `POST /setup/regenerate`, `DELETE /setup`.

FE: `services/koboService.ts` + `hooks/useKoboQueries.ts` + `types/kobo.ts` (đúng rule: không `api.get` trong component, type ở `types/`, mutation có `onError` + toast). Card giờ: URL thật từ server, badge Paired/Not set up, nút regenerate/revoke có confirm, cảnh báo **URL là credential**, cảnh báo **localhost máy Kobo không tới được** (`is_local_address` từ BE — đây là cách setup này chết trong im lặng nhiều nhất, calibre-web cũng cảnh báo đúng chỗ này). i18n 23 key × 5 locale, đã verify parity.

### 🔴 K8. Sync kẹt ở 100 quyển — [x] *(phát hiện khi refactor)*
`koboService.go` `GetSyncList` fetch `constants.MaxPaginationLimit` = 100 quyển **không cursor**, rồi skip quyển đã sync trong Go — mà `kobo.SyncItemLimit` **cũng = 100**.

Hệ quả: sync xong 100 quyển đầu thì mọi lần sync sau `items` luôn rỗng và `remaining = false` → **không bao giờ set `x-kobo-sync: continue`** → sách thứ 101 trở đi **không bao giờ tới được thiết bị**, và thiết bị không có cách nào biết là bị thiếu. Thư viện >100 quyển sync sai trong im lặng.

**Đã sửa:** đi thư viện bằng vòng lặp cursor `(created_at, id)` — `SearchBooks` đã nhận `cursor *time.Time, cursorID string`, không cần SQL mới. Lặp cho tới khi đủ `SyncItemLimit` item hoặc hết sách; trang ngắn (`< MaxPaginationLimit`) nghĩa là hết thư viện. `remaining` chỉ set khi trang đã đủ mà còn sách chưa xét — phân biệt "còn nữa" với "hết rồi", set sai thì thiết bị hỏi vô hạn.

**Test:** `TestKoboSyncPagesBeyondFirstPage` seed 150 quyển, sync tới khi hết header continue, assert đủ 150. **Fail trước khi sửa: "synced 100 of 150 books"**. Vòng lặp trong test có chặn 5 round nên bug "luôn báo continue" cũng fail chứ không treo CI.

### 🟠 K9. Refactor tuân thủ rule kiến trúc — [x]
Code Kobo lượt đầu chạy được nhưng vi phạm AGENTS.md ở 3 mặt. Tôi bám calibre-web mà quên đối chiếu rule của repo.

**1. Output không phải DTO.** Service trả `map[string]any` (`GetInitialization`, `GetUserProfile`, `AuthDevice`), controller trả `fiber.Map` (`setupPayload`). Type định nghĩa ở service (`KoboSyncInput`, `KoboSyncResult`) thay vì `internal/dtos`, naming lệch (`*Input`/`*Result` thay vì `*Dto`/`*Response`). Tệ nhất: `internal/dtos/request/kobo.go` tôi tạo **hoàn toàn không được import** — shape thật sống ở `pkg/kobo` bằng anonymous struct, tức **2 định nghĩa cho cùng 1 payload**.

**Đã sửa:** `request.PutKoboStateDto` (+ `KoboReadingStateDto`/`KoboBookmarkDto`/`KoboStatisticsDto`/`KoboStatusInfoDto`/`KoboLocationDto`), `request.KoboSyncDto`; `response.KoboSyncResponse`/`KoboSyncItemResponse`/`KoboEntitlementResponse`/`KoboInitResponse`/`KoboUserProfileResponse`/`KoboAuthDeviceResponse`/`KoboSetupResponse`. Xoá `kobo.PutStateRequest`.

`NewEntitlement`/`ChangedEntitlement` trước là `map[string]any` với **key động**, giờ là 2 field `omitempty` — device phân biệt bằng key nào có mặt, nên không dùng discriminator.

**Giữ wire type ở `pkg/kobo`** theo tiền lệ `pkg/opds` (giữ `Feed`/`Entry`/`Link`, `opdsService` trả `*opds.Feed` thẳng). Chuyển hết sang `internal/dtos` sẽ buộc `pkg/` import `internal/` — hiện chỉ `pkg/apperrors` làm vậy.

**Ngoại lệ có lý do:** `KoboInitResponse.Resources` vẫn là `map[string]any` — đó **đúng là** 147 key động của Kobo, 2 trong đó là object lồng. Struct hoá 147 field là chép lại data thành code.

**2. Hàm tách vô nghĩa / trùng lặp — xoá 10 cái:**

| Xoá | Thay bằng |
|---|---|
| `koboUserID`, `getSyncUserID` | `getUserIdFromLocals` ở `helper.go` — đã có sẵn, 26 call site, còn validate `ParseID`. Ba hàm cho một lookup |
| `koboEndpointURL`, `setupPayload` (base-URL block) | `getBaseURL` ở `opdsController.go` — copy lần 2 và 3 của cùng logic. Thêm `TrimSpace` vào bản gốc (bản Kobo có, bản OPDS thiếu) |
| `KoboAuthService.EndpointURL` | gộp vào `EnsureSetup`/`RegenerateSetup` — ghép `base + "/kobo/" + token` là copy thứ 4 |
| `isNotFound`, `maxTime` | viết thẳng vào chỗ dùng |
| `ResourceString` | **0 call site production** (7 chỗ chỉ trong test) → xoá, test tự có helper |
| `RandomTrackingID` | `uuid.NewString()` |
| `SeriesID` | inline vào `NewBookMetadata` — 1 call site |
| `ResultSuccess` (+ `*map[string]string`) | `kobo.PutStateSubResult` có tên; hàm sinh ra chỉ để che một kiểu khó chịu **tôi tự tạo** |
| `padBase64` | inline vào `ParseSyncToken` — 4 dòng, 1 call site |
| param `info` của `readingState()` | không đọc trong thân hàm |

`GetInitialization`/`GetUserProfile` bỏ luôn `ctx` và `error` (luôn nil) — thuần assembly, không có DB/FS. Theo `opdsService.GetOpenSearchDescription`.

**Giữ lại, có lý do:** `RandomToken` (24 byte + base64 + err), `compareVersion` (20 dòng, có test riêng), `accessibleBook`/`canSync`/`bookInfo` (nhiều caller, `accessibleBook` là cổng bảo mật duy nhất — gộp vào caller là mời quên check), constructor 1 dòng (idiom Go), repo method 1 dòng (implement interface).

**Test:** parse `PutKoboStateDto` chuyển sang `internal/dtos/request/kobo_test.go` (3 test, thêm case "0 và absent không được lẫn"), thêm `TestKoboSetupEndpointShape` pin snake_case key mà `web/src/types/kobo.ts` đọc. FE **không phải đổi** — JSON tag giữ nguyên.

### ⚠️ Nhóm K — mức bảo đảm thật, nói thẳng

**Chưa test trên máy Kobo thật. Không có máy.** 17 contract test chứng minh **byte trên dây khớp shape calibre-web** và các cổng auth/permission hoạt động. Nó **không** chứng minh máy thật sync được. Đây là trần bảo đảm đạt được trong điều kiện hiện tại, không phải "đã chạy".

Còn thiếu so với calibre-web, **có ý thức**:
- **Tags/collections (5 endpoint)** — chưa làm. Máy vẫn sync sách bình thường, chỉ không có shelf.
- **`DELETE /v1/library/<uuid>`** (archive từ máy) — chưa làm. Xoá trên máy không phản ánh về server.
- **`ChangedReadingState`** dạng item độc lập — chỉ gắn `ReadingState` kèm entitlement. Sách đã sync rồi mà chỉ đổi progress thì lần sync sau không đẩy được state; đợi có máy thật xác nhận đây có phải vấn đề thực tế trước khi thêm.
- **Kobo store proxy** (`config_kobo_proxy`) — không làm. Đây là tính năng gộp kết quả từ store thật, không cần cho self-hosted.
- **Sync scan cả trang thư viện rồi filter trong Go** — có comment `ponytail:` ghi rõ ceiling và đường nâng cấp (cursor theo timestamp trong sync token).
- **`/v1/user/profile`** vẫn là stub — calibre-web cũng không implement local (forward về store, trả `{}` khi tắt proxy), nên **không có payload đúng nào để copy**. Trả identity thật từ claims thay vì `novelhub-kobo-user` hardcode, nhưng key set là suy đoán và ghi rõ trong comment. Không có gì trong luồng sync đọc nó.

### 🟡 G3. `visibleBooks` alias in-place — [x]
`internal/services/opdsService.go:118` `visible := readable[:0]` ghi lên chính backing array đang iterate. Hiện an toàn nhưng là bẫy — entity đến từ cache repository và **được share với caller khác**.
**Đã sửa:** `filterVisible` cấp slice mới bằng `make`, dùng chung cho cả 7 feed (đoạn lọc này trước đây copy-paste 5 lần).

### 🟡 G4. Kobo bypass `RequestBodyLimit` — [x]
Route Kobo mount trên root app, không qua `/api` group → `POST /kobo/v1/library/state` (decode vào `map[string]any`) chỉ bị chặn bởi BodyLimit global 65 MiB.
**Đã sửa:** attach `RequestBodyLimit` trực tiếp vào group `/kobo/v1`. **Không** dời route sang `/api`: thiết bị đã ghép cặp dùng URL `${origin}/kobo`, dời là vỡ hết reader đang chạy.

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
- **`pages/admin/Users.tsx:199` restore** — có `refetchUsers()` + toast. Lệch style (thiếu hook) chứ không mất cache → hạ xuống E.
- **`GET /library/stats` public** — chỉ trả 3 con số tổng (`db/query/user_features.sql:1-6`). Ghi nhận, không sửa.

---

## Deferral có ý thức

- **B5 CSRF** — xem mục B5.
- **`OptionalJwtAccess` hạ cấp thành guest im lặng** — token hết hạn trên ~20 route read trả 200 với ít dữ liệu thay vì 401, nên interceptor auto-refresh của FE (`web/src/config/api.ts`) không bao giờ kích hoạt. Đúng thiết kế (guest browsing), nhưng gây UX khó hiểu khi token hết hạn giữa chừng. Cần quyết định riêng, không gộp vào lượt này.
- **Type input/output định nghĩa ở service thay vì `internal/dtos`** — `OPDSPageQuery`, `MetadataFacetQuery`, `TrackerSearchResult`, `UpdateCoverInput`, `BookmarkedBooksPage`, `PermissionContext`. Cùng lỗi vừa sửa ở K9 nhưng có **trước** lượt này, và `MetadataFacetQuery` còn là bản copy 1-1 của `request.MetadataFacetDto` **đã tồn tại** — controller copy 4 field sang rồi vứt. Sửa kèm vào diff refactor Kobo sẽ làm diff không review được. Quyết riêng.
- **`errors.Is(err, sql.ErrNoRows)` lặp 19 chỗ ở 10 file service** — ứng viên rõ cho `apperrors.IsNotFound(err)`: một guard, mọi caller đi qua. Đã chọn inline trong Kobo ở lượt này để giữ phạm vi; gom lại là task riêng.
