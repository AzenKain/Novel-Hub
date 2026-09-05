# Cấu hình

NovelHub chia cấu hình thành hai phần. Biến môi trường phụ trách những gì phải
biết trước khi mở database — hoặc những gì bảo vệ database nên không thể nằm bên
trong nó. Mọi thứ còn lại đặt trong giao diện admin và có hiệu lực ngay không cần
khởi động lại.

Chỉ có bốn biến thật sự quan trọng. Số còn lại đều có giá trị mặc định dùng được
và phần lớn tự điều chỉnh theo cấu hình máy.

---

## Bắt buộc

Copy `.env.example` thành `.env` rồi điền ba secret. Server sẽ không phát hành
token nếu thiếu chúng.

```bash
cp .env.example .env
openssl rand -hex 32   # run three times, one value each
```

| Biến | Mục đích |
| --- | --- |
| `JWT_SECRET` | Ký access token |
| `JWT_REFRESH_SECRET` | Ký refresh token |
| `DB_ENCRYPTION_KEY` | Mã hóa token của bên thứ ba (AniList, MAL) và mật khẩu SMTP lưu trong database |

Mỗi biến dùng một giá trị random khác nhau.

**Đổi về sau.** Đổi một trong hai JWT secret sẽ đăng xuất toàn bộ người dùng —
không mất dữ liệu. Đổi `DB_ENCRYPTION_KEY` khiến các token tracker và mật khẩu
SMTP đã mã hóa vĩnh viễn không đọc được nữa; người dùng phải kết nối lại những
tài khoản đó và admin phải nhập lại mật khẩu SMTP. Hãy backup database trước khi
chạm vào nó.

Chúng vẫn là biến môi trường vì chúng ký và mã hóa những gì nằm *trong* database.
Lưu chính chúng trong database sẽ thành vòng lặp luẩn quẩn.

---

## Reverse proxy

```bash
TRUST_PROXY=false
```

Biến này quyết định NovelHub có tin hai header do proxy gửi tới hay không:

- `X-Forwarded-For` — client thật sự là ai, dùng cho rate limit
- `X-Forwarded-Proto` — request ban đầu có phải HTTPS không, quyết định cookie
  đăng nhập có được gắn cờ `Secure` hay không

| Giá trị | Dùng khi | Tin cậy |
| --- | --- | --- |
| `false` | Trình duyệt kết nối trực tiếp tới NovelHub | Không gì cả |
| `true` | nginx/Caddy trên cùng máy, hoặc một container khác trong cùng Docker network | Địa chỉ loopback, private và link-local: `127.0.0.0/8`, `::1`, `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `fc00::/7`, `169.254.0.0/16` |
| `1.2.3.0/24,5.6.7.8` | Proxy nằm trên địa chỉ public — thường gặp nhất là Cloudflare | Đúng những IP và CIDR được liệt kê |

Đừng đặt `true` khi không có proxy thật đứng trước. Khi đó bất kỳ client nào cũng
có thể tự gửi hai header đó: mỗi request một rate-limit bucket mới (vô hiệu hóa
hoàn toàn bộ giới hạn đăng nhập) và một khai báo `https` giả mạo khiến cookie phục
vụ qua HTTP thuần bị gắn `Secure`, thứ mà trình duyệt âm thầm loại bỏ.

Đặt biến này chỉ là một nửa công việc — proxy của bạn phải thực sự gửi các header
đó. Xem [Reverse Proxy](reverse-proxy.md) để biết cấu hình cho từng loại proxy và
cách kiểm tra.

`TRUST_PROXY` chỉ được đọc một lần lúc khởi động và không thể là một setting trong
giao diện admin: nó quyết định cookie đăng nhập có `Secure` hay không, nên một giá
trị sai nằm trong database sẽ đồng nghĩa với việc phải đăng nhập được để sửa đúng
cái đang chặn bạn đăng nhập.

---

## Tùy chọn

Mọi thứ bên dưới đều đang bị comment trong `.env.example`. Chỉ bỏ comment những gì
bạn cần ghi đè.

### Mạng

| Biến | Mặc định | Ghi chú |
| --- | --- | --- |
| `SERVER_HOST` | `127.0.0.1` | Docker đặt `0.0.0.0`; đừng ghi đè ở đó nếu không port đã publish sẽ không truy cập được |
| `SERVER_PORT` | `3434` | |

### Lưu trữ

| Biến | Mặc định | Ghi chú |
| --- | --- | --- |
| `DATA_DIR` | `./data` | Thư mục gốc cho mọi thứ bên dưới |
| `SQLITE_DB_PATH` | `$DATA_DIR/novelhub.db` | |
| `CALIBRE_IMPORT_DIR` | `$DATA_DIR/calibre` | Chỉ những thư mục nằm dưới gốc này mới import được. Trỏ nó tới thư viện Calibre nếu thư viện nằm ở chỗ khác. |

`DATA_DIR` chứa:

```
data/
├── novelhub.db      SQLite database
├── books/           imported books and covers
├── calibre/         Calibre libraries available for import
├── inbox/           drop files here for automatic import
├── uploads/         in-progress chunked uploads
├── public/          uploaded site logo and favicon
├── logs/            rotating application logs
└── backups/         database backups
```

Backup `DATA_DIR` là backup toàn bộ bản cài đặt.

### Hiệu năng

Tất cả đều tự điều chỉnh. Chỉ đặt khi bạn muốn chủ động giới hạn tài nguyên.

| Biến | Mặc định |
| --- | --- |
| `SQLITE_CACHE_SIZE_KB` | Tính theo RAM hệ thống (64 MB–512 MB) |
| `SQLITE_MMAP_SIZE_BYTES` | Tính theo RAM hệ thống (256 MB–2 GB) |
| `SQLITE_MAX_OPEN_CONNS` | Số CPU × 2, giới hạn trong khoảng 4–16 |
| `SQLITE_MAX_IDLE_CONNS` | Bằng max open |
| `CACHE_MAX_COST_BYTES` | Tính theo RAM hệ thống |
| `ASSET_CACHE_MAX_COST_BYTES` | RAM hệ thống ÷ 32, kẹp trong 32 MB–512 MB — trang truyện và ảnh bìa, giữ dạng byte thô trong ngân sách riêng nên không thể đẩy bản ghi sách ra khỏi cache |
| `JOB_WORKERS` | `1` — mức song song của job nền |
| `GOGC` | `200` — mục tiêu GC của Go; giảm xuống là đánh đổi CPU để lấy bộ nhớ |
| `FIBER_CONCURRENCY` | Mặc định của Fiber |
| `FIBER_READ_BUFFER_SIZE` | Mặc định của Fiber |
| `FIBER_WRITE_BUFFER_SIZE` | Mặc định của Fiber |

### Log

| Biến | Mặc định | Ghi chú |
| --- | --- | --- |
| `LOG_MAX_SIZE_MB` | `10` | Kích thước để log đang hoạt động được rotate |
| `LOG_MAX_FILES` | `5` | Số file đã rotate được giữ lại |
| `DISABLE_REQUEST_LOG` | `true` | Tắt log theo từng request để tăng throughput |
| `DISABLE_STARTUP_MESSAGE` | `false` | |

Chúng vẫn là biến môi trường vì hệ thống log khởi động trước khi database được mở
— lỗi database thì phải ghi log được.

### Hành vi

| Biến | Mặc định | Ghi chú |
| --- | --- | --- |
| `TOKEN_VERSION_CACHE` | `true` | Đặt `false` khi nhiều instance dùng chung một database, lúc đó cache trong RAM sẽ bị cũ |
| `DISABLE_RESPONSE_COMPRESSION` | `false` | |
| `ENABLE_PREFORK` | `false` | Worker đa tiến trình. Vô hiệu hóa cache token version |
| `RESTORE_AUTO_RESTART` | `false` | Thoát sau khi chuẩn bị (stage) bản restore database để Docker hoặc systemd khởi động lại và áp dụng nó. Docker đặt `true` |

---

## Cấu hình trong giao diện admin

Không phải biến môi trường. Được thiết lập trong wizard cài đặt lần đầu, sau đó
sửa tại **Admin → Settings**. Thay đổi có hiệu lực ngay.

| Khu vực | Bao gồm |
| --- | --- |
| Site | Tiêu đề, mô tả, logo, favicon, mục sidebar, các section trang chủ |
| Server URL | Base URL tuyệt đối dùng trong link catalog OPDS và Kobo sync. Để trống là tự nhận diện theo từng request — chỉ điền khi host tự nhận diện bị sai, ví dụ khi nằm sau proxy |
| Access | Bật/tắt đăng ký, bắt buộc đăng nhập, chế độ truy cập cho khách, mức hiển thị cho khách theo từng library |
| Permissions | Quyền theo từng role cho cả 39 permission — đọc, tính năng cá nhân, nội dung library, tích hợp, quản trị |
| Email (SMTP) | Host, port, username, mật khẩu, địa chỉ gửi, chế độ TLS, dung lượng đính kèm tối đa (MB, mặc định 50MB), cho phép gọi mạng private, kèm nút test kết nối. Cũng gồm bật/tắt xác minh email và reset mật khẩu |
| Reader features | Deep search trong sách, upload font riêng, chọn chỉ số engagement nào hiện trên bìa, custom user CSS |
| OAuth / SSO | Cấu hình đăng nhập một lần (SSO) qua Google, GitHub, Discord, và OIDC (OpenID Connect) |
| Trackers | Bật/tắt sync tiến độ đọc với AniList, MyAnimeList, và Hardcover.app |
| Upload limits | Kích thước chunk, số chunk, số session đồng thời, tổng dung lượng, TTL của session, kích thước ảnh bìa và asset của site |
| Rate limits | Số lần thử đăng nhập và OPDS trong mỗi cửa sổ, và độ dài cửa sổ |

### Rate limit

NovelHub chỉ giới hạn đúng hai thứ, cả hai dùng chung một cặp setting:
**đăng nhập** (`/api/v1/auth/*`) và **OPDS** (`/api/opds/*`).

Cả hai đều chạy xác minh mật khẩu bằng bcrypt, tốn khoảng 50–100 ms CPU mỗi lần
thử — gấp chừng 600 lần mọi thứ còn lại trong request. Đó chính là tài nguyên đáng
bảo vệ. OPDS được tính vào vì nó dùng HTTP Basic auth, không mang session, nên
bcrypt chạy trên *mọi* request.

Mặc định: 5 lần thử trong 60 giây, tính theo IP client.

Với OPDS, chỉ những lần thất bại mới được tính. Một ứng dụng đọc sách poll catalog
bằng thông tin đăng nhập hợp lệ là traffic bình thường và không bao giờ bị chặn.

Cố ý không có rate limit chung cho toàn bộ API. Một chương truyện tranh render
thành một request ảnh cho mỗi trang, nên mở một tập 200 trang sẽ tạo ra đúng 200
request hợp lệ — một giới hạn chung sẽ chặn người đọc, không phải kẻ tấn công.

---

### Server OPDS 1.2 & 2.0

NovelHub tích hợp sẵn máy chủ OPDS 1.2 (Atom XML) và OPDS 2.0 (JSON) hoàn chỉnh:

- **OPDS 1.2 Catalog**: `/api/opds/v1` (Định dạng Atom XML, tương thích hoàn hảo với KOReader, Moon+ Reader, Calibre, PocketBook, Aldiko). Bao gồm các danh mục điều hướng (`/recent`, `/authors`, `/series`, `/tags`), OpenSearch XML (`/api/opds/v1/opensearch.xml`), và tìm kiếm toàn văn (`/api/opds/v1/search?q={searchTerms}`).
- **OPDS 2.0 Catalog**: `/api/opds/v2/catalog` (Định dạng JSON `application/opds+json`, tương thích với các trình đọc hiện đại như Thorium). Bao gồm các liên kết điều hướng gốc, thông tin chi tiết tác phẩm, ảnh bìa và liên kết tải sách.
- **Xác thực**: Hỗ trợ HTTP Basic Auth (dùng email và mật khẩu tài khoản) cũng như chính sách Truy cập Khách (Guest Access) được cấu hình theo từng thư viện trong Admin Settings.

### PWA & Đọc Sách Offline

NovelHub là một Progressive Web App (PWA) hoàn chỉnh với khả năng cài đặt ứng dụng trực tiếp:

- **Bộ máy Offline**: Cho phép lưu toàn bộ cuốn sách, các chương và hình ảnh đính kèm trực tiếp vào kho lưu trữ IndexedDB của trình duyệt để đọc offline 100% không cần kết nối mạng.
- **Service Worker & Cập nhật**: Vận hành bởi `vite-plugin-pwa` và `workbox` với thông báo tự động khi có bản cập nhật mới và theo dõi dung lượng bộ nhớ khả dụng.
- **Phân quyền**: Tính năng lưu sách đọc offline được kiểm soát theo từng vai trò thông qua quyền `book.offline`.

### OAuth / SSO (Single Sign-On)

Cấu hình đăng nhập từ bên thứ ba dưới mục **Admin → Settings → OAuth**.

- **Nhà cung cấp được hỗ trợ**: Google, GitHub, Discord, OIDC (OpenID Connect).
- **Thiết lập**: Cung cấp Client ID, Client Secret, Redirect URI (khớp với `/api/v1/auth/oauth/:provider/callback`), và Issuer URL (dành cho OIDC).
- **Hành vi**: Người dùng đăng nhập qua OAuth sẽ tự động đăng ký tài khoản mới (nếu chức năng đăng ký đang bật) hoặc liên kết với tài khoản email đã xác minh có sẵn.

### Podcasts

Đăng ký và quản lý nguồn phát thanh (podcast) dưới mục **Admin → Settings → Podcasts** (hoặc trang Podcasts).

- Đăng ký bằng URL nguồn RSS tuyệt đối.
- Cơ chế kiểm tra uncompiled template phát hiện và từ chối các nguồn dạng template thô của Jekyll để tránh lỗi đăng ký.
- Hỗ trợ tải nguồn dữ liệu lớn (tối đa 250MB).
- Đặt lịch định kỳ hoặc kích hoạt thủ công tác vụ làm mới và tải các tập podcast mới.

### Kids Mode & Age Rating (Chế độ Trẻ Em)

Lọc các nội dung dành cho người lớn đối với khán giả nhỏ tuổi.

- **Phân loại độ tuổi**: G, PG, R, R18.
- **Kids Mode**: Kích hoạt bằng mã PIN trong **User Profile → Kids Mode**. Khi bật, tất cả sách vượt quá phân loại độ tuổi tối đa cho phép sẽ tự động ẩn khỏi kệ sách và tìm kiếm. Tắt chế độ này yêu cầu nhập mã PIN dạng số gồm 6 chữ số.

### Tích hợp VBook

Cho phép duyệt và đọc sách từ thư viện của bạn bằng ứng dụng VBook trên Android.

- **Thiết lập**: Sao chép URL plugin JSON hoặc tải file `plugin.zip` từ trang cá nhân của bạn.
- **Registry**: Endpoint registry `/api/v1/vbook/plugin.json` cung cấp thông tin metadata, và `/api/v1/vbook/plugin.zip` cung cấp gói cài đặt cho VBook.

### Danh sách đọc & Nhập `.cbl`

Collection trả lời "cuốn này thuộc nhóm nào". Danh sách đọc trả lời "đọc cuốn nào
tiếp theo": mỗi mục mang một vị trí tường minh, nên thứ tự là do bạn đặt chứ
không phải thứ tự file được nhập vào kho.

- **Riêng từng user**: danh sách đọc chỉ thuộc về tài khoản đã tạo ra nó, giống Collection, và gác bằng cùng quyền `book.collection`.
- **Sắp xếp lại**: kéo một mục hoặc dùng nút lên/xuống ở `/read-lists`. Toàn bộ thứ tự được lưu trong một request.
- **Đọc theo thứ tự**: mở mục đầu tiên kèm `?readlist=<id>`. Hết chương cuối, nút next có sẵn của reader sẽ dắt sang cuốn kế tiếp trong danh sách thay vì dừng lại. Sách đã lưu trữ (archived) bị bỏ qua. Vị trí đang đọc **không** được ghi nhớ — "Đọc theo thứ tự" luôn bắt đầu ở mục đầu tiên.
- **Nhập `.cbl`**: tải lên một reading list của ComicRack (tối đa 8 MB). Thứ tự trong tài liệu *chính là* thứ tự đọc; không có gì bị sắp lại. Các mục khớp theo tên bộ truyện (không phân biệt hoa thường) cộng số tập, trong đó `01`, `1` và `1.0` được coi là cùng một số. `Year` và `Volume` bị bỏ qua vì bảng sách không có cột năm. Mục nào không tìm thấy trong thư viện sẽ được trả về trong báo cáo nhập kèm bộ truyện và số tập; nếu hai cuốn trùng cả bộ truyện lẫn số tập thì lấy cuốn tìm thấy đầu tiên.
- **Endpoint** (tất cả dưới `/api/v1/read-lists`): `GET /`, `POST /`, `POST /import`, `GET|PUT|DELETE /:id`, `GET|POST /:id/books`, `DELETE /:id/books/:bookId`, `PUT /:id/order`, `GET /:id/next`.

### Bác sĩ Sách & Công cụ Sửa lỗi EPUB (Book Doctor)

NovelHub tích hợp bộ công cụ phân tích và tự động sửa chữa cấu trúc các file EPUB bị lỗi chuẩn hoặc hỏng hóc:

- **Kiểm tra & Phát hiện lỗi**: Quét và chẩn đoán toàn diện header ZIP, vị trí file `mimetype`, cấu trúc file `container.xml`, cú pháp file `content.opf`, trùng lặp ID/href trong manifest, file rác chưa khai báo manifest, thiếu spine, link nội bộ HTML bị đứt gãy, lỗi cú pháp XML (như `&nbsp;` chưa escape) và thiếu mục lục NCX/Nav.
- **Quy trình Tự động Sửa chữa**:
  - Tái tạo file `mimetype` chuẩn không nén ở vị trí byte đầu tiên của file ZIP.
  - Khử trùng lặp manifest và loại bỏ các mục manifest trỏ tới file không tồn tại.
  - Tự động sửa XML namespace và làm sạch các thẻ liên kết nội bộ bị gãy.
  - Tự động sinh mục lục chuẩn (`toc.ncx` / `nav.xhtml`) và nâng cấp chuẩn cũ EPUB 2.0 lên chuẩn hiện đại EPUB 3.0.
- **Đồng bộ Hash & Xóa Cache**: Sau khi sửa chữa thành công, hệ thống tự động tính toán lại mã băm SHA-256 của file, cập nhật vào database và xóa toàn bộ RAM cache liên quan (`book:*`, `book_file:*`, `chapter:*`).
- **Tác vụ Sửa lỗi Hàng loạt**: Quản trị viên có thể quét và tự động sửa lỗi toàn bộ thư viện sách thông qua cron job chạy ngầm (`repair_books`) trong mục **Admin → Operations → Maintenance**.

### Giả lập API Calibre Content Server

NovelHub tích hợp lớp giả lập API máy chủ Calibre Content Server chuẩn xác, cho phép các ứng dụng đọc sách trong hệ sinh thái Calibre kết nối trực tiếp:

- **Endpoint**: `/calibre/ajax/*` (duyệt danh mục, phân loại sách và tìm kiếm) và `/calibre/get/:what/:book_id` (tải ảnh bìa và tải file sách).
- **Ứng dụng hỗ trợ**: Calibre Companion, Aldiko, và các trình đọc e-reader hỗ trợ chuẩn giao tiếp JSON của Calibre Content Server.
- **Xác thực linh hoạt**: Hỗ trợ HTTP Basic Auth, Bearer token, query token (`?token=...`), cookie phiên đăng nhập và chế độ Khách (Guest mode nếu được bật trong Admin Settings). Tài khoản bị cấm (banned) sẽ nhận mã lỗi 403 Forbidden.
- **Kiểm soát truy cập**: Tự động lọc sách và quyền tải xuống theo từng thư viện của người dùng (`CanReadBook`, `CanDownloadBook`) cùng cơ chế giới hạn phân trang tối đa 100 kết quả để phòng chống tấn công DoS.

### Máy chủ WebDAV Chuẩn (RFC 4918)

NovelHub cung cấp máy chủ WebDAV chuẩn IETF RFC 4918 cho phép gắn kết thư viện sách trực tiếp như một ổ đĩa mạng trên máy tính hoặc đồng bộ với ứng dụng đọc sách di động:

- **Endpoint**: `http(s)://<host-của-bạn>/webdav`
- **Phương thức hỗ trợ**: `OPTIONS`, `PROPFIND` (Depth 0, 1, và vô hạn), `GET` (hỗ trợ phân luồng đọc phạm vi byte HTTP 206 Partial Content), và `HEAD`.
- **Ứng dụng hỗ trợ**: Moon+ Reader, KyBook 3, FBReader, Foliate, Zotero, macOS Finder (`Connect to Server`), Windows File Explorer (`Map Network Drive`), Linux (Nautilus/Dolphin/davfs2).
- **Xác thực & Mã QR Đồng bộ**: Hỗ trợ HTTP Basic Auth, Bearer token và token trên URL. Thẻ **User Profile → WebDAV Sync** tự động tạo sẵn thông tin kết nối và mã QR Code để quét nhanh trên thiết bị đọc sách di động.
- **Bảo mật & Phân quyền**: Được kiểm soát chặt chẽ thông qua quyền `system.webdav`. Các thư viện bị giới hạn theo vai trò sẽ tự động ẩn khỏi cây thư mục.

### Xuất Thẻ Ghi Nhớ Anki (.apkg & .csv)

Đồng bộ các đoạn trích dẫn, từ vựng và ghi chú khi đọc sách trực tiếp vào bộ thẻ Anki:

- **Gói Thẻ Độc Lập `.apkg`**: Xuất toàn bộ trích dẫn và ghi chú ra file gói `.apkg` (chuẩn SQLite Anki) để nhập trực tiếp vào Anki Desktop, AnkiMobile (iOS) hoặc AnkiDroid (Android) mà không cần cài đặt thêm add-on.
- **Xuất file `.csv` chuẩn Anki**: Xuất dữ liệu dưới dạng bảng CSV phân tách dấu phẩy gồm Mặt trước (câu trích), Mặt sau (ghi chú cá nhân) và Ngữ cảnh (đoạn văn xung quanh trong sách).
- **Cầu nối AnkiConnect**: Hỗ trợ đồng bộ tức thời hai chiều tới ứng dụng Anki đang mở trên máy tính thông qua add-on AnkiConnect (`http://127.0.0.1:8765`).

### Công Cụ Dọn Dẹp Metadata & Tiêu Đề Hàng Loạt (Bulk Title Cleaner)

Nằm trong mục **Admin → Books & Libraries → Bulk Title Cleaner**:

- **Làm sạch theo quy tắc**: Loại bỏ hàng loạt các thẻ trong ngoặc vuông (như `[Light Novel]`), ngoặc tròn (`(2024)`), đổi dấu gạch dưới thành khoảng trắng và chuẩn hóa khoảng trắng thừa.
- **Tách Tiêu đề & Tác giả tự động**: Tách tên file gộp thành các trường tiêu đề và tác giả riêng biệt theo ký tự phân tách (`Tên sách - Tác giả` hoặc `Tác giả - Tên sách`).
- **Biểu thức chính quy tùy chỉnh (Regex)**: Hỗ trợ tạo các quy tắc Regex tùy chỉnh kèm bảng xem trước kết quả trực tiếp trước khi ghi đè vào cơ sở dữ liệu.

### Thẻ Đọc Sách Cá Nhân & Thẻ Trích Dẫn (Reading & Quote Cards)

- **Thẻ Thành Tích Đọc Sách (Reading Cards)**: Mở từ trang **Reading Analytics & Stats** (`/analytics`). Cho phép tạo và xuất ảnh card thành tích cá nhân độ nét cao hiển thị tổng số từ đã đọc, số ngày đọc liên tục (streak), thời gian đọc và bản đồ nhiệt hàng năm với nhiều giao diện màu sắc (Aurora, Cyberpunk, Sepia, E-ink) cùng tỉ lệ khung hình tùy chọn để chia sẻ lên mạng xã hội.
- **Thẻ Trích Dẫn (Quote Cards)**: Bôi đen đoạn văn trong trình đọc và bấm vào biểu tượng Quote để tạo ảnh trích dẫn nghệ thuật kèm bìa sách, tên chương và tên tác giả.

### Chế Độ Màn Hình Mực Điện Tử (Pure 1-Bit E-Ink Mode)

Thiết kế tối ưu hóa riêng cho các thiết bị e-paper (Kindle, Kobo, Boox, Supernote) cũng như người thích đọc sách tập trung:

- **Độ tương phản cao tuyệt đối**: Bảng màu đen trắng 1-bit thuần túy (chữ `#000000` trên nền `#ffffff`), tắt toàn bộ hiệu ứng chuyển động (0ms transitions) và đổ bóng mờ để tăng tốc độ phản hồi trên màn hình e-ink.
- **Hiệu ứng Flash làm mới trang**: Tùy chọn chớp sáng trang mô phỏng thao tác xóa bóng mờ (ghosting clearance) của máy đọc sách chuyên dụng.
- **Hỗ trợ toàn diện**: Áp dụng đồng bộ cho cả Trình đọc sách đa định dạng (eBook Reader) và Trình nghe sách nói (Audiobook Player).

---

## Cookie xác thực

Không có gì để cấu hình; cả hai thuộc tính đều được suy ra theo từng request.

**`Secure`** được đặt mỗi khi request đến qua HTTPS — trực tiếp, hoặc qua
`X-Forwarded-Proto` của một proxy đáng tin (xem `TRUST_PROXY` ở trên). Trên HTTP
thuần nó bị bỏ đi, vì trình duyệt âm thầm loại bỏ cookie `Secure` trên kết nối
không an toàn, và biểu hiện ra ngoài giống như sai mật khẩu.

**`Domain`** không bao giờ được đặt, nên cookie chỉ giới hạn trong host đã phục vụ nó.

**`SameSite`** là `Lax` và nên giữ nguyên như vậy. Nới xuống `None` là trao cho
trang của kẻ tấn công khả năng gửi kèm cookie của bạn.

**`csrf_token`** là cookie thứ ba, cố ý để JavaScript đọc được. Frontend chép nó
vào header `X-CSRF-Token`, server so hai giá trị này ở mọi POST/PUT/PATCH/DELETE.
Request mang header `Authorization`, cùng các prefix `/kobo/`, `/komga/`,
`/api/opds/` và `/api/v1/sync/`, được miễn — chúng xác thực theo từng request và
không gửi cookie, nên không có gì để giả mạo.

---

## Kiểm tra

```bash
# Server is up
curl http://127.0.0.1:3434/api/v1/health

# Rate limit engages (expect 429 within the first handful)
for i in $(seq 10); do
  curl -s -o /dev/null -w "%{http_code}\n" \
    -X POST http://127.0.0.1:3434/api/v1/auth/signin \
    -H 'Content-Type: application/json' \
    -d '{"email":"nobody@example.com","password":"wrong"}'
done
```

Khi nằm sau proxy, hãy đăng nhập qua HTTPS và kiểm tra
**DevTools → Application → Cookies**. Dòng `access_token` phải hiện `Secure`. Nếu
không, proxy đang không gửi `X-Forwarded-Proto` hoặc `TRUST_PROXY` không bao phủ
địa chỉ của nó.
