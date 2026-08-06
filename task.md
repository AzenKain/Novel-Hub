# NovelHub — Audit findings & fix backlog

Toàn bộ số liệu dưới đây là **đo được**, không suy đoán.
**Cập nhật 2026-08-06:** 12 finding đã sửa (A1–A5, B1+Komga, B2, C1, C2, C3, C5) + FTS rowid.
`go build ./... && go vet ./... && go test ./... -count=1` xanh — 33 package ok, 0 fail.
Máy đo: AMD Ryzen 5 5600H, `modernc.org/sqlite`, RAM 15 GiB → cache production = RAM/16 = **940 MiB**.

Thứ tự ưu tiên: **đúng đắn → hiệu năng → tài nguyên**.

---

## Trạng thái Komga cache

Code xong: cả 5 read path đã có singleflight + cache-by-IDs đúng chuẩn.
**Nhưng không có một dòng invalidation nào.** Đo với series mới + tập mới của series cũ, so ground truth cache lạnh:

| | phục vụ | thực tế |
|---|---|---|
| `CountSeries` | 1 | 2 |
| `ListSeries` | 1 | 2 |
| `ListSeriesBooks` | 1 | 2 |

Mihon không thấy truyện mới trong 15 phút (`NormalCacheDuration`). Gộp fix vào **B1** vì cùng chỗ.

**✅ ĐÃ SỬA.** `CacheKeyKomgaSeriesPattern` + `CacheKeyKomgaBookSeriesPattern` (`pkg/constants/cache_keys.go`)
nối vào 9 site invalidate. Probe giờ `served=2 actual=2` cả 3 hàng.
`komga:pages:*` **cố tình không sweep** — key theo mod_time, dựng lại tốn 38ms/CBR.

---

## A. ĐÚNG ĐẮN / BẢO MẬT

### A1. ✅ ĐÃ SỬA — Rò số liệu chéo user — `metadata_count` không có library scope

- `internal/repositories/bookMetadataRepository.go:835` (đọc), `:861` (ghi)
- `internal/services/metadataService.go:52` — `ReadableLibraryIDs(ctx, claims)` là ranh giới quyền, key vứt đi
- `internal/routes/metadataRoutes.go:12` — `OptionalJwtAccess`, **truy cập được khi chưa đăng nhập**

```go
keys[i] = cache.BuildKey("metadata_count", entityType, "id", id)   // thiếu scope
```

Giá trị cache là `book_count`, mà `db/query/metadata.sql:41` tính nó **theo scope người gọi**
(`WHERE b.library_id IN (SELECT value FROM json_each(...))`). Key danh sách facet *có* scope,
key entity bên dưới *không* — hai tầng mâu thuẫn, last-writer-wins toàn cục.

Kịch bản đã chạy được:
```
1. guest đọc: book_count=1   (đúng — chỉ lib-open đọc được)
2. admin đọc: book_count=4   → ghi đè key dùng chung metadata_count:author:id:au-1
3. guest đọc lại: book_count=4   ← lộ số sách trong thư viện guest không có quyền
```

Cửa sổ 15 phút. Ảnh hưởng cả 6 facet (`:615, 655, 695, 735, 775, 815`).

**Fix:** đưa scope vào key entity, y như key list đang làm —
`cache.BuildKey("metadata_count", entityType, "id", id, strings.Join(f.LibraryIDs, ","))`.
Cần luồn filter vào `getMetadataCountByIDs`/`cacheMetadataCountEntities` (private, 6 call site).
Rẻ hơn: bỏ hẳn tầng entity cho nhóm này, cache nguyên kết quả đã scope dưới key list.

**Đã sửa:** `MetadataFacetFilter.scopeKey()` (xxhash của scope) gấp vào key entity →
`metadata_count:author:id:au-1:<hash>`. Guard: `auditCache_test.go` — guest đọc lại vẫn ra 1.

### A2. ✅ ĐÃ SỬA — `filepath.Match` không bao giờ khớp key chứa `/`

- `pkg/cache/ramcache.go:145`

`*` không vượt qua `/`, mà **mọi** pattern trong `pkg/constants/cache_keys.go` đều là `prefix*`.
Hai họ key kẹt vĩnh viễn, chỉ hết hạn bằng TTL:

**(a) `book_file:path:<absolute path>`** — ghi ở `bookFileRecordRepository.go:225, 247, 274`.
Chỉ `BulkDeleteBooks` (`bookCatalogRepository.go:571`) và `DeleteLibrary` (`libraryRepository.go:242`)
quét, cả hai đều truyền `"book_file*"` mà không liệt kê path.
(`DeleteBook:421` và `DeleteFile:370` làm đúng — `Del` chính xác từng path.)

**(b) `metadata:<facet>:<cursor>:<limit>:<search>:...`** — `Search` vào key thô
(`bookMetadataRepository.go:30-33`), mà `MetadataFacetDto.Search` chỉ validate `max=200`,
không giới hạn ký tự (`internal/dtos/request/common.go:18`). Một query `?search=sci-fi/fantasy`
là đủ đúc ra key mà **19 sweep site** không xoá nổi.

Đo:
```
key = "book_file:path:/nas/library/Dune/dune.epub"   pattern "book_file*"  -> Match=false
                                                      pattern "book_file:*" -> Match=false
key = "metadata:authors::20:sci-fi/fantasy::lib-1"   pattern "metadata:*"  -> Match=false
key = "user:search:6acf8593ad1d8917"                 pattern "user:search*" -> Match=true  (QueryKey hash, sạch)
```

Hệ quả end-to-end: sau `BulkDeleteBooks`, row `book_files` đã biến khỏi DB nhưng
`GetBookFileByPath` vẫn trả `id="f-1"` suốt 15 phút → download/read lấy file đã xoá, rồi 404.

**Fix:** đổi `filepath.Match` → `strings.HasPrefix(key, strings.TrimSuffix(pattern, "*"))`
trong `DelByPattern`. Một dòng, cả 12 pattern hiện tại đều là `prefix*` nên phủ hết.
Sửa ở `DelByPattern`, **không** sửa ở call site.

**Đã sửa** đúng như trên, 1 dòng trong `ramcache.go`.

### A3. ✅ ĐÃ SỬA (bằng cách XOÁ key) — Lệch key raw-vs-resolved trên `book_file:path`

- Read ghi key theo path **đã resolve**: `cacheBookFileEntities` (`bookFileRecordRepository.go:225`),
  gọi từ 2 read path sống (`:111`, `:181`) — `models.BookFileEntity.FromSqlc` chạy `localfs.ResolveBookFilePath`
- Mọi `Del` nhắm path **thô** từ cột DB: `:23, :47, :297, :370` + `bookCatalogRepository.go:421`

Lệch nhau đúng lúc file vắng mặt trên đĩa — chính ca mà `ResolveBookFilePath` sinh ra để xử lý.
**Fix của A2 KHÔNG che được cái này** (đây là lệch cách dựng key, không phải lỗi glob).

**Đã sửa bằng cách xoá key, không phải căn lại.** `GetBookFileByPath` **không có call site nào**
trong repo (chỉ định nghĩa sqlc + interface + impl) → `book_file:path:*` là 100% chỉ-ghi.
Xoá khỏi `cacheBookFileEntities`, xoá 2 `Set`, xoá 5 `Del`; path giờ chỉ còn làm khoá singleflight.
Diff nhỏ hơn việc luồn resolved path qua 9 site, và **đóng luôn C3**.
Guard: `TestProbeBookFilePathIsNotCached` — khẳng định *không* key nào (thô hay resolved) được ghi.

### A4. ✅ ĐÃ SỬA — `CreateChapter` nuốt lỗi rồi commit `ready`

- `internal/services/bookService.go:343` — `_ = txRepo.CreateChapter(ctx, chapter)`
- `:347` set `book.Status = "ready"`, rồi commit

Sách mất chương không phân biệt được với sách đủ. `chapters` **không có unique constraint**
(`db/schema/20_books.sql:48`), nên chạy lại `extract_metadata` **nhân đôi toàn bộ chương** chứ không vá.
Khôi phục phải sửa DB bằng tay.

**Đã sửa:** `return err` để tx rollback và job retry.

### A5. ✅ ĐÃ SỬA — Rò `http.Response.Body` khi non-200

- `internal/services/settingsService.go:700-704`

```go
resp, err := client.Do(req)
if err != nil || resp.StatusCode != http.StatusOK {   // thoát TRƯỚC defer
    return "", apperrors.New(...)
}
defer resp.Body.Close()                                // không bao giờ tới khi non-200
```

`err == nil` nhưng status 301/403/404/500 → body không đóng, socket không về idle pool.
Admin dán URL logo/favicon bị 404 → tích luỹ.
Bonus: `||` short-circuit đang gánh việc chống nil-deref; đảo vế là panic.

**Fix:** check `err` trước → return, rồi `defer resp.Body.Close()`, rồi mới check status.

**Đã sửa** theo đúng thứ tự đó.

5 site HTTP anh em đều đúng (`bookService_cover.go:86`, `webhookService.go:265`,
`trackerService.go:199,294`, `deviceService.go:252`).

---

### A6. ✅ ĐÃ SỬA — `sort.Strings` trên slice alias của caller phá keyset pagination

**Không nằm trong bản audit gốc — lộ ra khi một guard mới fail.**

`internal/repositories/`: `bookCatalogRepository.go:477`, `jobRepository.go:116`,
`libraryRepository.go:159`, `roleRepository.go:87`, `userRepository.go:340`.

Các reader by-IDs sort danh sách id để có **key singleflight ổn định**. Khi **mọi id đều miss cache**,
`missingIDs` là **alias** của slice caller truyền vào, nên `sort.Strings(missingIDs)` **ghi đè trang của caller**.
`orderBooks(ids, …)` sau đó tôn trọng đúng thứ tự đã bị sort → `SearchBooks` giao `created_at DESC`
và nhận lại **id tăng dần**.

Triệu chứng: cursor chỉ tiến **1 sách/trang** thay vì 100 → `StreamLibraryZip` sinh 43.425 entry
đáng lẽ 714. Trúng cả nhánh in-tx (luôn miss cache).

**Đã sửa:** sort **bản copy**, 3 dòng mỗi site:
```go
sortedIDs := append([]string(nil), missingIDs...)
sort.Strings(sortedIDs)
sfgKey := "books:ids:" + strings.Join(sortedIDs, ",")
```
`queryInChunks(missingIDs, …)` bên dưới **không đổi** — nó cần thứ tự gốc.

Guard: `internal/repositories/byIDsOrder_test.go` — hỏi `[b-4,b-2,b-0,b-3,b-1]`, khẳng định slice
caller **không bị mutate** và thứ tự kết quả khớp thứ tự hỏi. **Đã verify guard FAIL khi revert fix**,
rồi restore — guard chỉ là guard khi đã thấy nó đỏ.

---

## B. HIỆU NĂNG

### B1. ✅ ĐÃ SỬA — Sweep cache chạy BÊN TRONG write transaction — nặng nhất

- `internal/services/bookService.go:315-357` (tx mở `:315`, commit `:357`)
- `internal/repositories/chapterRepository.go:31` — 1 sweep mỗi chương
- `internal/repositories/bookMetadataRepository.go:325-341` và các hàm Link/Clear anh em

`DelByPattern` quét **toàn bộ** cache (`Range()`, không index key). Nó bọc bởi `if r.c != nil`
mà **không có `!r.inTx`** — trong khi `userRepository.go:520` đã có sẵn guard đó kèm comment
giải thích đúng ca này.

`ExtractMetadata` mở tx `_txlock=immediate` rồi bên trong: `ParseSpine` (`:332`, giải nén cả archive)
+ `CreateChapter` mỗi spine item (`:343`) + ~8 sweep cho metadata link.
Chi phí = **O(chapters × cache_entries)** — bậc hai theo hai biến độc lập, mà biến khuếch đại
(kích thước cache) phản ánh lưu lượng duyệt web toàn server, chẳng liên quan gì tới cuốn sách đang import.

Đo với 1 cuốn EPUB 600 chương (chỉ 0,2 MB trên đĩa):

| cache entries | ExtractMetadata | write-lock stall dài nhất |
|---|---|---|
| 0 (lạnh) | 123ms | 131ms |
| 20.000 | 1,61s | 1,633s |
| 250.000 | **42,95s** | 10,016s → **4/45 ghi đồng thời FAIL** |

`ParseSpine` một mình chỉ ~110ms; phần còn lại là quét cache trong tx.

Vượt `busy_timeout=10000` thì writer khác không còn chờ mà **lỗi thẳng
`SQLITE_BUSY: database is locked (5)`** — 500 cho user, không phải chỉ chậm.
Một `UPDATE books SET title=...` 1 dòng mất **1,377s** để commit khi có import đang parse.

Chi phí sweep vs `Del` chính xác: **4.047× đến 59.469×**, và `Del` **phẳng** theo kích thước cache
(2–5µs bất kể bao nhiêu entry).

Quy mô hệ thống: **146 `DelByPattern` trong `internal/repositories/`, chỉ 20 có guard `!r.inTx`**.
Nặng nhất: `bookMetadataRepository.go` (26), `bookCatalogRepository.go` (22), `bookFileRecordRepository.go` (21).

**Đã sửa — chọn chokepoint thay vì 125 guard tay.** `pkg/cache/deferred.go`: `WithTx` trao cho
repo tx-scoped một `DeferredCache` gom + dedup invalidation, `FlushCache(ctx)` sau `Commit` phát một lần
(8 site). `ParseSpine` chuyển ra trước `BeginTx`. Cách này còn đóng cửa sổ reader cache lại
giá trị tiền-commit — thứ mà `!r.inTx` rải rác không làm được.

Đo lại:

| cache entries | trước | sau | stall dài nhất trước | sau |
|---|---|---|---|---|
| 0 | 123ms | **90ms** | 131ms | 79ms |
| 20.000 | 1,61s | **62ms** | 1,633s | **0s** |
| 250.000 | 42,95s | **237ms** | 10,014s + 4 SQLITE_BUSY | **79ms**, 0 lỗi |

Chi phí **không còn phụ thuộc kích thước cache**. Hai assertion trong `auditTx_test.go` đã đảo
thành chặn trên (`> 500ms` là hồi quy) vì bản gốc viết để *chứng minh* bug tồn tại.

### B2. ✅ ĐÃ SỬA — Thiếu index FK con — `DELETE FROM books` quét bảng, **185×**

| Schema | Cột | EQP |
|---|---|---|
| `56_highlights.sql:4` | `highlights.book_id` | `SCAN ... COVERING INDEX idx_highlights_user_book` |
| `57_reading_sessions.sql:4` | `reading_sessions.book_id` | `SCAN ... sqlite_autoindex_reading_sessions_2` |
| `98_kobo_auth.sql:29` | `kobo_synced_books.book_id` | `SCAN ... sqlite_autoindex_kobo_synced_books_1` |
| `59_user_trackers.sql:21` | `book_tracker_mappings.book_id` | `SCAN ... idx_btm_user_book_provider` |
| `55_reading_activity.sql:4` | `reading_progress.file_id` | `SCAN reading_progress` (không index) |

Xoá 200 sách **không có row con nào**, 200 user:

| row con | hiện tại | +4 index | |
|---|---|---|---|
| 10k | 1,438s (7,19ms/sách) | 34,4ms | 41,8× |
| 40k | 5,650s (28,2ms/sách) | 30,5ms | **185,5×** |
| scaling | **3,93×** (tuyến tính) | 0,89× (phẳng) | |

Đây là chi phí thuần "đi tìm con". Đường đi: `bookService.go:809`, `bookService_bulk.go:73`,
`maintenanceService.go:205`, calibre sync.

⚠️ Cảnh báo phương pháp: `ANALYZE` trong seed bật skip-scan làm bug **biến mất**.
Production **không chạy ANALYZE** (đã verify: vắng mặt trong `db/` và mọi file Go non-test),
nên không được cứu. `reading_progress.file_id` tuyến tính kể cả có ANALYZE.

**Đã sửa:** 5 index (`idx_highlights_book`, `idx_reading_sessions_book`, `idx_btm_book`,
`idx_reading_progress_file`, `idx_kobo_synced_books_book`).
`TestAuditIndexFKChildCoverage` giờ thấy SEEK ở **mọi** cột FK, allowlist rỗng
(sau khi C5 drop `reading_history` — entry non-SEEK cuối cùng).

### B0. ✅ ĐÃ SỬA — Cột `UNINDEXED` của FTS5 là quét toàn bảng → quét thư viện O(n²)

Phát hiện ngoài lượt audit này (trong lúc chạy scale test 100k, test timeout 40 phút **lúc seed**).
Cả 3 bảng FTS5 dùng cột `UNINDEXED` làm khoá tra cứu; FTS5 **không có index trên cột thường** —
`EXPLAIN QUERY PLAN` ra `SCAN fts_metadata VIRTUAL TABLE INDEX 0` cho mọi `WHERE book_id = ?`.

| Bảng | Thao tác nóng | 8k dòng | 32k |
|---|---|---|---|
| `fts_metadata` | trigger UPDATE, 4 lần/sách | 3,48ms | **12,8ms** |
| `fts_chapters` | DELETE mỗi lần reindex | 2,91ms | **11,6ms** |
| `fts_users` | DELETE+INSERT mỗi update user | — | **12,5ms** @20k |

**Đã sửa — căn `rowid` bảng FTS trùng rowid bảng gốc, ghi theo rowid.** Giữ nguyên cột nên
**không query SELECT nào phải sửa** (kể cả `snippet()` và `fts_metadata MATCH` table-level).
12 trigger ở `45_metadata_fts.sql`, 3 trigger ở `10_auth.sql` (`t_users_fts_au` đổi DELETE+INSERT
→ UPDATE thuần), `search.sql` căn theo `chapters.rowid`.

| | trước | sau | |
|---|---|---|---|
| `fts_metadata` UPDATE @32k | 10,6ms | **11µs** | 881× |
| `fts_chapters` xoá 1 sách @32k | 14,6ms | **484µs** | 30× |
| `fts_users` update @20k | 12,5ms | **198µs** | 63× |

5 rủi ro đã kiểm, tất cả an toàn: rowid tái sử dụng (trigger xoá FTS cùng lúc hàng gốc),
`trusted_schema(OFF)`, `VACUUM` (không tồn tại trong repo — thứ duy nhất đánh số lại rowid),
backup/restore (copy trang + `io.Copy`, rowid sống nguyên), `WITHOUT ROWID` (không bảng nào dùng).

Guard: `internal/repositories/ftsRowidSync_test.go` — apply schema thật lên DB trắng, khẳng định
mismatch/orphan/missing = 0 sau mỗi giai đoạn, + chi phí không tăng khi số dòng ×4.

Xác nhận cuối bằng chính test đã phát hiện bug: `TestKomgaCatalogueUnderConcurrentLoad`
(`NOVELHUB_SCALE_TEST=1`) trước **timeout 40 phút lúc seed**, giờ seed
**10.000 series × 10 sách = 100.000 sách trong 24,5s**, cả test 27,8s:

| readers | pages | per_page | throughput | alloc |
|---|---|---|---|---|
| 1 | 20 | 2,747ms | 364 pg/s | 2 MB |
| 8 | 160 | 1,086ms | 921 pg/s | 10 MB |
| 16 | 320 | 1,443ms | 693 pg/s | 15 MB |
| 50 | 1000 | 2,625ms | 381 pg/s | 34 MB |

### B3. 🟠 Phân trang sâu là O(offset) — **rewrite query, KHÔNG phải index**

- `db/query/books.sql:16-18`

Trang 1: 88µs. Trang 400: 27,3ms, tăng tuyến tính. Phân rã 3 nghi phạm:

| biến thể | 10k | 40k |
|---|---|---|
| hiện tại (`datetime()` + `?IS NULL OR`) | 7,08ms | 28,4ms |
| bỏ `datetime()` | 84,9µs | 17,4ms |
| bỏ cả hai | 141µs | **120µs** |
| row-value `(created_at,id) < (?,?)` + index | 71µs | **57,6µs** |
| **hiện tại + đúng index đó** | 6,30ms | **26,5ms** |

Index **không cứu được**. Thủ phạm: `datetime(created_at)` là hàm trên cột, và disjunction
`sqlc.narg` `IS NULL OR` chặn seek. Không phân rã thì đã kê nhầm thuốc.

Ảnh hưởng `ListBookIDs`, `SearchBookIDs`, `GetBookIDsInCollection` (69,5ms/trang @40k, 4,65×)
và ~15 cursor query khác.

### B4. 🟠 6 facet query scale theo tổng số sách dù page cố định

4,19×–6,17× khi bảng lớn lên. Bản rewrite EXISTS **nhanh 11,4× ở 40k, row giống hệt byte-by-byte**.
Tệ hơn: **một** `AddBookTag` xoá sạch trang facet, nên trong lúc scan/sync mọi facet view
đều trả giá lạnh (~481ms/trang khi ngoại suy 200k sách).

### B5. ✅ ĐÃ SỬA — `GetDuplicateFileDetails` không có LIMIT

Vi phạm duy nhất luật max-100 trong CLAUDE.md. 40k row / 272ms.

**Đã sửa:** LIMIT đặt **trong subquery `HAVING COUNT(*) > 1`**, không phải ngoài cùng.
Giới hạn theo *nhóm hash*, không theo *row*: LIMIT ở ngoài sẽ cắt giữa một nhóm và
`if len(files) < 2 { continue }` (`bookService.go`) im lặng bỏ luôn nhóm đó.
Chữ ký `(ctx, limit int64)` luồn qua interface → repo → service, gọi với `constants.MaxPaginationLimit`.

### B6. ✅ ĐÃ SỬA — 3 read có cache thiếu singleflight

`bookFileRecordRepository.go:310 GetDuplicateFiles`, `:341 ListAllFiles`, `:406 CountFilesForBook`.
Mọi read có cache khác trong cùng file (`:71, :239, :266`) đều có `sfg.Do`.
Vi phạm luật bắt buộc trong CLAUDE.md. Không sai kết quả, chỉ query trùng.

**Đã sửa:** cả 3 bọc `sfg.Do` theo đúng khuôn `GetBookFileById:256`. `grep -c "sfg.Do"` 3 → 6.

### B7. ✅ ĐÃ SỬA (nửa zip) / ⏭️ CỐ Ý BỎ (nửa calibre)

**Con số 8,4× trong bản báo cáo này là sai — đo lại chỉ 1,4×.** Sửa phép đo, không sửa con số:
thêm warmup + 10 lần chạy, rồi phân rã raw SQL / sqlc / mapper.

| tầng | 200 row |
|---|---|
| raw SQL | ~1ms |
| sqlc | ~2ms |
| **`FromSqlc`** | **28,46ms** |

28 trong 31ms nằm ở **mapper**, không phải SQL. Truy tiếp: `localfs.ResolveBookFilePath` gọi
`filepath.Abs` → `os.Getwd` = **141µs/lần** trên máy này (mount ngoài/FUSE), nhưng **chỉ trên nhánh
`os.Stat` miss** → là artifact của test không có file thật, không phải chi phí production.

Đo lại **có file thật trên đĩa**: `12,24ms → 1,36ms` mỗi trang 100 sách = **9,0×**. Đó là số thật.

**Đã sửa:** `StreamLibraryZip` gọi `GetFilesByBookIDs` 1 lần/trang rồi group bằng map.

**Nửa calibre cố ý bỏ:** 11 name lookup/sách **phẳng theo kích thước bảng**
(14,7µs @200 tag → 12,9µs @2000) và chỉ chiếm **4,8–24,2%** chi phí mỗi sách, cạnh sha256.
Batch lại là tối ưu 5% → không đáng diff.

## C. TÀI NGUYÊN

### C1. ✅ ĐÃ SỬA — Queue "bounded" thực ra unbounded

- `pkg/worker/queue.go:54` — `pond.NewPool(workers)` không option
- `queue.go:103-110` — nhánh `default:`

`pond.NewPool` mặc định `queueSize = DefaultQueueSize = Unbounded = math.MaxInt`
(verify tại `pond@v2.7.1/pool.go:16-18, 575`). Channel 5000 slot **không phải backpressure, chỉ là gờ giảm tốc**:
đầy rồi thì mọi job sau đó đi thẳng vào buffer vô hạn. **Không gì từ chối job.**

Concurrency vẫn chặn ở `JOB_WORKERS=1` → đây là **unbounded memory/queue depth**, không phải rò goroutine.

Đường kích hoạt: `POST /api/v1/libraries/:id/upload` enqueue **2 job mỗi file**
(`libraryService.go:219-228`), **không giới hạn số file** (`libraryController.go:113` lấy nguyên
`form.File["files"]`; chỉ tổng body size bị chặn). Mỗi job còn ghi 1 row DB.

**Đã sửa:** nhánh `default:` không submit vào pool nữa mà trả `worker.ErrQueueFull` (+ `lifecycle.Failed`),
nên channel 5000 slot thành bound thật. `UploadFiles` cap `maxUploadFiles = 200`.

### C2. ✅ ĐÃ SỬA — Vòng lặp khuếch đại webhook `job.failed`

- `internal/services/jobService.go:187-199` ↔ `internal/services/webhookService.go:170-204`

`Failed` dispatch `job.failed` cho **mọi** job hỏng, **không loại trừ `job.Type == "webhook.dispatch"`**
(verify: `job_type` truyền làm data nhưng không branch).

Chu trình: endpoint chết → `webhook.dispatch` fail (3 attempt) → `Failed` → `DispatchEvent("job.failed")`
→ enqueue 1 `webhook.dispatch` mới **mỗi webhook đăng ký** → fail y hệt → quay lại bước 1.

`job.failed` chọn được trong admin UI (`web/src/components/admin/settings/WebhookModal.tsx:61`),
kể cả webhook đăng ký `"*"`. Tự nuôi khi endpoint còn chết, nhân N×/thế hệ.
Mỗi thế hệ ghi 1 row jobs + 3 attempt HTTP timeout 10s. Chĩa thẳng vào C1.

Giảm nhẹ: `JOB_WORKERS=1` tuần tự hoá, backoff ~1,5s/job, `prune_finished_jobs` dọn lịch sử.
Nhưng **không gì cắt vòng lặp**.

**Đã sửa** đúng như vậy ở `jobService.go:195`.

### C3. ✅ ĐÃ SỬA — 49% cache `book_file` là rác chỉ-ghi

Hệ quả của A2 + A3. ~32 MiB ở 100k sách, chỉ TTL dọn được.
**Đã đóng bởi A3** — key không còn được ghi.

### B8. ✅ ĐÃ SỬA — `ListFormatsWithCount` — đã đo xong, chuyển từ NGHI NGỜ sang sửa

Báo cáo cũ ghi `SCAN b`. **Đo lại: sai.** Plan thật là nested loop —
`SEARCH b USING idx_books_library_created` rồi `SEARCH bf USING idx_book_files_book (book_id=?)`
một lần **mỗi sách**, cộng 2 temp b-tree.

| | 10k | 40k |
|---|---|---|
| hiện tại | 28,0ms | 134,4ms |

4,8× khi bảng 4× — scale theo tổng số sách dù LIMIT cố định (cùng bệnh B4).

Phân rã 4 biến thể @40k:

| biến thể | thời gian |
|---|---|
| hiện tại | 133,8ms |
| `COUNT(*)` bỏ DISTINCT | 75,2ms |
| **`EXISTS` cho `book_files` lái** | **71,1ms** |
| hiện tại + index `(format, book_id)` | 135,3ms |
| `EXISTS` + index đó | 83,4ms |

**Index không cứu được** (thêm vào còn chậm hơn). `COUNT(*)` nhanh tương đương EXISTS nhưng
**không được dùng**: không có `UNIQUE(book_id, format)` nên một sách có 2 file cùng format
sẽ bị đếm hai lần — DISTINCT là đúng đắn, không phải thừa.

**Đã sửa:** `JOIN books` → `WHERE EXISTS (...)`, để `book_files` lái vòng lặp.
Plan mới dùng covering index + BLOOM FILTER. **1,76× trên query đầy đủ filter** (135,3ms → 76,7ms).
`sqlc` param struct **không đổi** → 0 dòng Go phải sửa.

Verify: 6/6 case (no filter / search / alpha / cursor / limit / library khác) **row giống hệt**;
`metadataFacetShape_test.go` (đã có case `formats`) xanh.

### C4. ✅ ĐÃ SỬA — 19 index thừa — vệ sinh, KHÔNG phải hiệu năng

10 trùng hệt + 9 thừa tiền tố trên tổng 124. Chi phí ghi **+6,2/5,9/6,8/6,5%** (4 lần chạy).
Nhưng bỏ 13 cái đo được **−8%, tức không lợi**.

Đáng chú ý: `idx_reading_progress_user_time` ≡ `idx_reading_progress_user_updated`;
`idx_bookmarks_user_time` ≡ `idx_bookmarks_user_created`; `role_permissions` có cả hai cặp nhân đôi;
`idx_series_name`/`idx_publishers_name`/`idx_languages_name` (`25_metadata.sql:44-46`)
trùng UNIQUE autoindex — đo được là **chết hẳn**, mọi probe đều chọn `sqlite_autoindex_*`.

⚠️ Cảnh báo phương pháp: lần đo đầu ra **−6%** — bất khả thi khi *thêm* index.
Nguyên nhân là **thứ tự nhánh benchmark** (cùng một nhánh: 2,13s vs 3,29s chỉ do vị trí).
Phải chạy trung bình cả hai chiều mới ra số đúng.

**Đã bỏ 10 định nghĩa index:**
- `25_metadata.sql`: `idx_series_name`, `idx_publishers_name`, `idx_languages_name`
  (bị UNIQUE autoindex che; đo lại sau B4 vẫn trong noise → chết hẳn)
- `70_performance_indexes.sql`: `idx_reading_progress_user_updated`, `idx_bookmarks_user_created`
  (trùng hệt định nghĩa **khác tên** → b-tree thật thứ hai trên mọi lần ghi),
  `idx_book_files_book`, `idx_chapters_book_index`, `idx_highlights_user_book`
  (trùng **cùng tên** với `30_:30`, `20_:63`, `56_:15` → `IF NOT EXISTS` biến thành no-op, vô hại),
  `idx_role_permissions_role`, `idx_role_permissions_perm`
- `65_permissions_settings.sql`: `idx_role_permissions_role_id` (đã có `UNIQUE(role_id, permission_key)`);
  **giữ** `idx_role_permissions_permission_key`

Phân biệt quan trọng: trùng **tên** không tốn gì lúc ghi; trùng **định nghĩa dưới tên khác** mới tốn.
Chỉ loại thứ hai (+ loại bị UNIQUE/tiền tố che) bị bỏ. Verify: 0 tên trùng còn lại,
`auditIndex_test.go` (bất biến FK-CASCADE-phải-seek) xanh sau khi bỏ.

### C5. ✅ ĐÃ SỬA — `reading_history` là bảng chết

`db/schema/50_user_features.sql:22-32`. Không sinh sqlc code, không query nào tham chiếu
(hit trong Go chỉ là chuỗi cache key). Vẫn mang `chapter_id` không index (`SCAN reading_history`)
mà mọi lần xoá chương phải trả giá. **Đã sửa:** drop bảng + `idx_reading_history_user_time`.

### C6. ✅ ĐÃ SỬA (guard) — `*:name` metadata key không bao giờ invalidate

`author:name`, `tag:name`, `series:name`, `publisher:name`, `language:name` được ghi
(`bookMetadataRepository.go:83, 137, 230, 315, 456, 532`), không có `Del` nào cho các prefix đó.
Hạ mức vì các entity này là create-or-get, không có method update/delete nào — rows coi như bất biến.
Thành bug thật ngay khi ai đó thêm tính năng đổi tên tác giả.

**Đã xử lý bằng guard, không phải `Del` phỏng đoán.** Cache **hiện tại đúng** vì rows bất biến;
thêm `Del` bây giờ là code chết. `internal/repositories/metadataNameCache_test.go` quét
`db/query/*.sql` tìm `-- name: (Update|Delete|Rename)(Author|Tag|Series|Publisher|Language)`
và fail ngay khi ai đó thêm query đổi tên mà chưa có `cache.Del`. Hôm nay: 0 kết quả → xanh.

---

## SẠCH — đã kiểm, không phải độn báo cáo

- **Data race: không có.** `-race` xanh trên `./internal/services` (191s), `./pkg/cache` (192s), `./pkg/worker`.
  Gồm cả test 12 goroutine gọi `ExtractMetadata` đồng thời.
  `permissionCache`, `settingsService.raw`, `uploadService.sessions`, `OTPStore`, `RotatingWriter`,
  `RamCache.versions`, `systemgate.Gate` đều đồng bộ đúng.
- **Rò connection/tx: không có.** Cả 26 site `BeginTx` non-test đều `defer tx.Rollback()` ngay sau check lỗi.
  30 vòng `ExtractMetadata` trả `db.Stats().InUse` về baseline (0→0), `BeginTx` sau đó lấy write lock ngay.
- **`rows.Close`/`stmt.Close`/archive reader: sạch.** 9 query tay trong `pkg/calibre/metadata.go` đều defer;
  24 site archive reader đều đóng; cả 2 handle `sql.Open` ngoài đều đóng.
  `sqlc.Prepare()` **không bao giờ được gọi** — 20 repo đều dùng `sqlc.New(db)`.
- **Request-ctx rò vào background: SẠCH, kỷ luật khác thường.** 24 controller đều detach qua
  `context.WithTimeout(context.Background(), ...)`; `pkg/worker` derive job ctx từ `q.ctx` (`queue.go:157`)
  chứ không từ người enqueue. Đúng ở **hai tầng độc lập**.
- **Transaction dài khác: không có.** Quét toàn bộ 26 khối tx tìm file I/O / network / parsing — đúng 1 hit là B1.
  Calibre sync, backup, upload, email đều giữ tx thuần DB, việc filesystem đặt sau commit.
- **Negative caching: sạch.** `GetFilesByBookId:79` cache list rỗng 10 phút, nhưng `CreateBookFile` invalidate nó.
- **Byte cache chia sẻ buffer: cơ chế, không phải bug sống.** `bytecache.go:67 GetOrLoad` trả buffer dùng chung,
  nhưng `bookService.go:765` chỉ cho `image/*` vào cache, còn `text/css` (thứ duy nhất bị biến đổi)
  đi nhánh không cache và `scopeReaderCSS` cấp phát chuỗi mới. `RamCache.MGet` trả bản sao.
- **`GetLibraryStats` PHẲNG** (1,24×) — đã nghi, đo xong thì không phải.
- **Series cover correlated subquery** chỉ chiếm 4% thời gian query — không phải vấn đề.
- **Multi-write không tx:** không có finding thật. Các ứng viên (`koboAuthService.go:55/61`, upload paths)
  hoặc idempotent độc lập, hoặc có cleanup bù trừ tường minh.

---

### A7. ✅ ĐÃ SỬA — race thật trên `q.handler`/`q.lifecycle`, chứng minh bằng `-race`

Nghi ngờ cũ nói "an toàn hôm nay vì 16 lần đăng ký đều trước `Start()`". Đã dựng probe đăng ký
handler **sau** `Start()` trong lúc worker đang chạy job:

```
WARNING: DATA RACE
Read at 0x00c000126930 by goroutine 11:
      .../pkg/worker/queue.go:143
```

`q.mu` đã tồn tại nhưng chỉ canh `stopped`. Sửa: cho nó canh luôn hai trường kia —
`RegisterHandler`/`SetLifecycle` lấy `Lock()`, thêm `lc()`/`handlerFor()` đọc dưới `RLock()`,
9 chỗ đọc `q.lifecycle` viết lại thành `if lc := q.lc(); lc != nil`. Sau sửa: `ok 1.019s` dưới `-race`.
Guard: `pkg/worker/handlerRace_test.go` (chỉ `-race` bắt được).

### B9. ✅ ĐÃ SỬA — `ListAllReviews` quét toàn bảng + sort tạm, **342×**

Chuyển từ NGHI NGỜ sang sửa vì đã đo tách được cả hai nguyên nhân.

Plan trước: `SCAN br` + `USE TEMP B-TREE FOR ORDER BY`. Không index nào có `updated_at` đứng đầu —
3 index hiện có đều bắt đầu bằng `book_id`.

| n dòng | trang đầu | trang sâu |
|---|---|---|
| gốc @8k | 21,4ms | 37,6ms |
| gốc @32k | 85,1ms | 154,2ms |
| + index `(updated_at DESC)` @32k | **248µs** | 33,2ms |
| + deferred join @32k | **307µs** | **1,85ms** |

Hai nguyên nhân tách bạch: index giết temp b-tree (trang đầu 342×), deferred join chặn việc
join `users`+`books` cho cả 32k dòng rồi vứt hết trừ 50 (trang sâu thêm 18×).

Chi phí ghi đo riêng: UPDATE review @32k **82,6µs → 102,9µs** (+20µs). Đổi 20µs ghi lấy 85ms đọc.

Chữ ký Go không đổi (`Limit`, `Offset`) → **0 dòng Go phải sửa**.
Guard `reviewsIndexGuard_test.go` canh plan, đã verify đỏ khi bỏ index.

### ⏭️ `DownloadLibraryZip` — nghi ngờ DoS đã **đo là sai**, để nguyên

Nghi ngờ: "goroutine `context.Background()` 15 phút, kệ client disconnect".
Đo bằng server fiber thật + client ngắt giữa chừng:

```
PROBE: goroutine exited 0s after client close: write err after 2 chunks: io: read/write on closed pipe
```

fasthttp gọi `CloseWithError` lên body stream khi ghi lỗi (`http.go:432,738`), `io.PipeReader`
có method đó → mọi `pw.Write` kế tiếp trả lỗi ngay. 15 phút không bao giờ tới.

Vẫn là dead code: không route nào đăng ký (`git log -S` xác nhận **chưa bao giờ** đăng ký), FE
không gọi. Nhưng service có đủ và có test (`libraryZip_test.go`, 714 entry) → không xoá, chỉ ghi nhận.

---

## D. FE ↔ BE — LỆCH HỢP ĐỒNG

Audit 6 loại: cursor pagination từng hook, tên/kiểu field vs JSON tag, envelope,
endpoint không tồn tại, `limit > 100`, lỗi mới FE chưa xử lý.

**Sạch:** envelope (9/9 hook đọc đúng tầng), 110 path FE gọi đều resolve tới route đã đăng ký,
`limit` lớn nhất FE gửi là 100.

### D1. ✅ ĐÃ SỬA — FE tự chế cursor collections, mất `|id`, **nuốt 7/12 hàng**

`useLibraryQueries.ts:95` vứt `next_cursor` của BE rồi tự dựng lại từ `created_at` hàng cuối.
BE phát `"<RFC3339Nano>|<id>"` và parse bằng `SplitN(cursor,"|",2)`; cursor không có `|`
rơi vào nhánh chỉ set `cursorCreatedAt`, để `cursorID` rỗng.

Đo bằng đường ghi thật (`CreateCollection`, tức `DEFAULT CURRENT_TIMESTAMP` — độ phân giải giây):

```
BE-style (ts|id):  [c-11 c-10 c-09 c-08 c-07 c-06 c-05 c-04 c-03 c-02 c-01 c-00]   12/12
FE-style (ts):     [c-11 c-10 c-09 c-08 c-07]                                       5/12
```

Không lỗi, không cảnh báo — 7 collection biến mất im lặng. Sửa: dùng `CursorPaginatedResponse<T>`
+ đọc `next_cursor`, đúng pattern `useReadingHistoryQuery` ngay dưới nó đã làm.
Guard: `collectionCursorGuard_test.go`.

Bẫy đo: probe đầu insert bằng `time.Time` của Go → lưu `"2026-08-06T12:21:28Z"` trong khi
`cursorTimeArg` phát `"2026-08-06 12:21:28"`, so chuỗi `'T' > ' '` nên **cả hai** style đều
gãy — số giả. Phải seed qua repo thật mới ra đúng bức tranh.

### D2. ✅ ĐÃ SỬA — Admin Users gửi `page`, BE chỉ hiểu cursor → kẹt ở 50 user

`userService.go:604` cursor-only, `dto.Page` đọc rồi vứt, không có offset nào.
`Users.tsx:62` gửi `page: 1` cứng, hook bỏ luôn `pagination.next_cursor`. Không có UI chuyển trang.
Admin không xem được quá 50 user đầu.

Sửa: `SearchUserParams.page` → `cursor`, hook trả thêm `nextCursor`, trang thêm nút
Trước/Sau đúng pattern `AuditTab.tsx` đã dùng, đổi search/filter thì reset cursor.
Dùng lại key `common.previous`/`common.next` — **0 key locale mới**.

### D3. ✅ ĐÃ SỬA — `ErrQueueFull` thành 500, client không biết là tạm thời

`apperrors/handler.go` không có case cho nó → rơi xuống 500 kèm message thô. 500 là status
không client nào retry. Sửa một chỗ ở `HandleError` (`worker` không import `apperrors` nên không cycle)
→ **503** cho tất cả 8 call site `Enqueue`. Guard: `pkg/apperrors/queueFull_test.go`.

Kèm theo, `bookAdminStore.ts:401` catch mỗi file chỉ `console.error` → người dùng thấy
**không gì cả** khi vài file lỗi; toàn bộ lỗi thì toast tiếng Anh cứng.
Sửa: giữ lý do lỗi đầu tiên, phân biệt xong-hết / xong-một-phần, 3 key × 5 locale.

### D4. ✅ ĐÃ SỬA — `/reader/history/progress/:id` khai sai kiểu ở FE

Báo cáo audit nói `location_cfi`/`location_type` là field chết. Đúng một nửa:
`ReadingHistoryResponse` hứa 2 field mà entity không có và query không select → key
không bao giờ lên wire. Nhưng `ReaderWorkspace.tsx:441-448` **đang đọc** chúng — từ endpoint
*progress*, mà service khai nhầm kiểu trả về là `ReadingHistory`.

Endpoint đó trả `ReadingProgressResponse`, **có** 2 field. Xoá lời hứa thừa ở
`ReadingHistoryResponse` + FE type, thêm type `ReadingProgress` khớp DTO thật.
`tsc` là thứ bắt được — xoá field khỏi `ReadingHistory` làm 5 lỗi bật ra ở `ReaderWorkspace`.

### 📌 Ghi nhận, không sửa

- `GET /user/devices` nhận query `cursor` (`deviceController.go:41-48`) nhưng không trả cursor →
  chết cả hai đầu.
- `POST /libraries/:id/upload` cap 200 file: không FE caller nào (web dùng luồng chunked 1 file/lần).
- 5 chỗ `toast` text tiếng Anh cứng còn lại trong `bookAdminStore.ts` — nợ sẵn có, khác loại.

---

## NGHI NGỜ — CHƯA CHỨNG MINH, đừng động vào khi chưa đo

- `internal/services/jobScheduleService.go:149,176` — `Stop()` chỉ signal, `runDue` dùng
  `context.Background()` riêng nên shutdown không huỷ được việc đang chạy.
  `main.go:41` dừng scheduler trước queue → schedule row có thể bị Claim rồi enqueue fail.
  Tự lành qua `ReleaseClaim`; xấu nhất là bỏ 1 tick.


---

## Còn lại — thứ tự đề xuất

Đã xong: **B0–B9**, **A1–A7**, **C1–C6**, **D1–D4**.

Còn lại:

1. **B7 nửa calibre** — 11 name lookup/sách. **Đã đo là không đáng**: phẳng theo kích thước bảng,
   4,8–24,2% chi phí mỗi sách. Chỉ làm nếu profile sync thật chỉ vào đây.
2. Chưa làm từ trước: test thiết bị thật — Mihon trên điện thoại đánh vào server đang chạy.
3. Xoá `data/novelhub.db*` rồi chạy lại — index `idx_book_reviews_updated` là schema mới,
   `ApplySchema` track theo tên file nên DB đang tồn tại sẽ không nhận.

Đã dọn: `RamCache.versions`/`verMu` là dead code → **đã xoá** (kéo theo import `sync`).

Ngoài phạm vi (đã thống nhất từ trước): `bookService_bulk.go:70` bỏ lỗi,
`maintenanceService.go:134` log thay vì trả lỗi, external-content FTS, gộp 4 lần update FTS/sách thành 1.
Mục **NGHI NGỜ** ở trên vẫn chưa chứng minh — đừng động khi chưa đo.

---

## File test đo đạc — trạng thái

| File | Nội dung | Trạng thái |
|---|---|---|
| `internal/repositories/auditCache_test.go` | A1 scope guard + bảng key-vs-pattern | **Giữ** — guard, đã xoá 3 case chết theo A3 |
| `internal/services/auditTx_test.go` | B1: bảng đo + writer bị chặn + rò connection + race probe | **Giữ** — 2 assertion đã đảo thành chặn trên 500ms |
| `internal/repositories/auditIndex_test.go` | Mọi cột FK CASCADE phải seek được | **Giữ** — allowlist giờ rỗng, xanh sau khi bỏ 10 index |
| `internal/repositories/ftsRowidSync_test.go` | 3 bất biến rowid qua 7 giai đoạn + chi phí không tăng theo n | **Giữ** — guard chống hồi quy về `WHERE book_id = ?` |
| `internal/repositories/scanInvalidation_probe_test.go` | `TestProbeBookFilePathIsNotCached` | **Giữ** phần A3 |
| `internal/repositories/byIDsOrder_test.go` | A6: caller slice không bị mutate + thứ tự giữ nguyên | **Giữ** — guard đã verify FAIL khi revert |
| `internal/repositories/metadataNameCache_test.go` | C6: quét `db/query/*.sql` tìm query rename/delete metadata | **Giữ** — tripwire, hôm nay 0 kết quả |
| `internal/repositories/cursorSeek_test.go` | B3: cursor không lặp trang + trang sâu không scale theo bảng | **Giữ** — `seekSeed` đã dọn về đây từ probe |
| `internal/repositories/metadataFacetShape_test.go` | B4: row giống hệt + chi phí không scale | **Giữ** — chứa `facetTime` |
| `internal/services/libraryZip_test.go` | B7: mọi file phải có trong zip, không trùng, tên đã escape | **Giữ** — 714 entry, chính test đã lộ A6 |
| `pkg/worker/handlerRace_test.go` | A7: đăng ký handler đồng thời với worker đang chạy | **Giữ** — chỉ `-race` bắt được |
| `internal/repositories/reviewsIndexGuard_test.go` | B9: plan không được có temp b-tree, phải giữ deferred join | **Giữ** — verify đỏ khi bỏ index |
| `internal/repositories/collectionCursorGuard_test.go` | D1: cursor thiếu `\|id` phải đi không hết bảng | **Giữ** — canh cả hai chiều |
| `pkg/apperrors/queueFull_test.go` | D3: `ErrQueueFull` phải ra 503 chứ không 500 | **Giữ** |
| ~~`zzcursorseek_probe_test.go`~~ | B3 probe | **Đã xoá** — `seekSeed` chuyển sang `cursorSeek_test.go` |
| ~~`zzfacetshape_probe_test.go`~~ | B4 probe | **Đã xoá** — số đã ghi ở đây |
| ~~`zzkomgastale_probe_test.go`~~ | Komga staleness | **Đã xoá** — `served=2 actual=2` |
| ~~`zzpathleak_probe_test.go`~~ | Đo rác cache C3 | **Đã xoá** — key không còn tồn tại |
| ~~`pkg/cache/delpattern_probe_test.go`~~ | Chi phí `DelByPattern` | **Đã xoá** — số đã ghi ở đây |
| ~~`pkg/cache/zzscale_probe_test.go`~~ | Sweep ở 64k/256k/1M | **Đã xoá** — số đã ghi ở đây |
| ~~`zzb7_probe_test.go`~~, ~~`zzc4_probe_test.go`~~, ~~`localfs/zzresolve*_probe_test.go`~~ | B7/C4 phân rã tầng | **Đã xoá** — số đã ghi ở B7/C4 |

Lệnh chạy: `go test ./internal/repositories/ -run 'TestAudit|TestFTS' -v -count=1`
và `go test ./internal/services/ -run TestAudit -v -count=1 -race -timeout 15m`.

---

## Sửa test, không phải sửa code — race do chính test tạo ra

`TestJobFailedStillRecordsWithoutWebhookService` (`jobFailed_test.go`) chỉ đỏ **khi chạy cả package**,
xanh khi chạy riêng. Không đổ cho flake — đã truy tới cùng:

- `UpdateJobStatus` là **last-write-wins** (chứng minh: gọi `Failed` rồi `Completed` → status `"completed"`).
- Harness đăng ký handler `database_health_check` **trả nil** và `queue.Start()`.
- Test gọi `Trigger` (→ enqueue, worker chạy thành công → `lifecycle.Completed` ghi `"completed"`)
  rồi **ngay lập tức** tự gọi `Failed` lên đúng job đó. Hai writer, một hàng.
- Test anh em sống sót chỉ vì có `time.After(400ms)` chen giữa.

**Production không có race:** `queue.process()` gọi `Completed` **hoặc** `Failed` rồi `return`, không bao giờ cả hai;
và `Failed` **không có caller nào ngoài worker** (đã grep toàn repo).

**Đã sửa test, không sửa code:** dùng `service.Queued(ctx, job)` — tạo hàng job mà không đưa cho worker.
Test cần một hàng job tồn tại, không cần queue. `-count=5` xanh.
