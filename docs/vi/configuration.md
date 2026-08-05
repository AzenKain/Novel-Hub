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
|---|---|
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
|---|---|---|
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
|---|---|---|
| `SERVER_HOST` | `127.0.0.1` | Docker đặt `0.0.0.0`; đừng ghi đè ở đó nếu không port đã publish sẽ không truy cập được |
| `SERVER_PORT` | `3434` | |
| `SERVER_URL` | — | Base URL tuyệt đối dùng trong các link catalog OPDS. Chỉ cần khi host tự nhận diện bị sai, ví dụ khi nằm sau proxy có rewrite path |

### Lưu trữ

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `DATA_DIR` | `./data` | Thư mục gốc cho mọi thứ bên dưới |
| `SQLITE_DB_PATH` | `$DATA_DIR/novelhub.db` | |

`DATA_DIR` chứa:

```
data/
├── novelhub.db      SQLite database
├── books/           imported books and covers
├── inbox/           drop files here for automatic import
├── uploads/         in-progress chunked uploads
├── logs/            rotating application logs
└── backups/         database backups
```

Backup `DATA_DIR` là backup toàn bộ bản cài đặt.

### Hiệu năng

Tất cả đều tự điều chỉnh. Chỉ đặt khi bạn muốn chủ động giới hạn tài nguyên.

| Biến | Mặc định |
|---|---|
| `SQLITE_CACHE_SIZE_KB` | Tính theo RAM hệ thống (64 MB–512 MB) |
| `SQLITE_MMAP_SIZE_BYTES` | Tính theo RAM hệ thống (256 MB–2 GB) |
| `SQLITE_MAX_OPEN_CONNS` | Số CPU × 2, giới hạn trong khoảng 4–16 |
| `SQLITE_MAX_IDLE_CONNS` | Bằng max open |
| `CACHE_MAX_COST_BYTES` | Tính theo RAM hệ thống |
| `JOB_WORKERS` | `1` — mức song song của job nền |
| `GOGC` | `200` — mục tiêu GC của Go; giảm xuống là đánh đổi CPU để lấy bộ nhớ |
| `FIBER_CONCURRENCY` | Mặc định của Fiber |
| `FIBER_READ_BUFFER_SIZE` | Mặc định của Fiber |
| `FIBER_WRITE_BUFFER_SIZE` | Mặc định của Fiber |

### Log

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `LOG_MAX_SIZE_MB` | `10` | Kích thước để log đang hoạt động được rotate |
| `LOG_MAX_FILES` | `5` | Số file đã rotate được giữ lại |
| `DISABLE_REQUEST_LOG` | `true` | Tắt log theo từng request để tăng throughput |
| `DISABLE_STARTUP_MESSAGE` | `false` | |

Chúng vẫn là biến môi trường vì hệ thống log khởi động trước khi database được mở
— lỗi database thì phải ghi log được.

### Hành vi

| Biến | Mặc định | Ghi chú |
|---|---|---|
| `TOKEN_VERSION_CACHE` | `true` | Đặt `false` khi nhiều instance dùng chung một database, lúc đó cache trong RAM sẽ bị cũ |
| `DISABLE_RESPONSE_COMPRESSION` | `false` | |
| `ENABLE_PREFORK` | `false` | Worker đa tiến trình. Vô hiệu hóa cache token version |
| `RESTORE_AUTO_RESTART` | `false` | Thoát sau khi chuẩn bị (stage) bản restore database để Docker hoặc systemd khởi động lại và áp dụng nó. Docker đặt `true` |

---

## Cấu hình trong giao diện admin

Không phải biến môi trường. Được thiết lập trong wizard cài đặt lần đầu, sau đó
sửa tại **Admin → Settings**. Thay đổi có hiệu lực ngay.

| Khu vực | Bao gồm |
|---|---|
| Site | Tiêu đề, mô tả, logo, favicon, mục sidebar, các section trang chủ |
| Access | Bật/tắt đăng ký, chế độ truy cập cho khách, mức hiển thị cho khách theo từng library |
| Permissions | Quyền theo từng role: đọc, tải, bookmark, collection, đánh giá, chia sẻ |
| Upload limits | Kích thước chunk, số chunk, số session đồng thời, tổng dung lượng, TTL của session, kích thước ảnh bìa và asset của site |
| Rate limits | Số lần thử đăng nhập và OPDS trong mỗi cửa sổ, và độ dài cửa sổ |

### Rate limit

NovelHub chỉ giới hạn đúng hai thứ, cả hai dùng chung một cặp setting:
**đăng nhập** (`/api/v1/auth/*`) và **OPDS** (`/opds/*`).

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

## Cookie xác thực

Không có gì để cấu hình; cả hai thuộc tính đều được suy ra theo từng request.

**`Secure`** được đặt mỗi khi request đến qua HTTPS — trực tiếp, hoặc qua
`X-Forwarded-Proto` của một proxy đáng tin (xem `TRUST_PROXY` ở trên). Trên HTTP
thuần nó bị bỏ đi, vì trình duyệt âm thầm loại bỏ cookie `Secure` trên kết nối
không an toàn, và biểu hiện ra ngoài giống như sai mật khẩu.

**`Domain`** không bao giờ được đặt, nên cookie chỉ giới hạn trong host đã phục vụ nó.

**`SameSite`** là `Lax` và nên giữ nguyên như vậy. NovelHub không có CSRF token hay
kiểm tra origin, nên `SameSite` là lớp phòng vệ CSRF duy nhất. Nới xuống `None` là
bỏ đi lớp bảo vệ đó và không thay thế bằng bất cứ thứ gì.

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
