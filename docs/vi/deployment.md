# Triển khai

Có hai cách chạy NovelHub: Docker, hoặc binary native. Cả hai đều cần ba secret
giống nhau từ [Cấu hình](configuration.md).

Frontend được biên dịch thẳng vào binary, nên chỉ có một tiến trình, một port và một
thư mục cần backup. Không web server, không host frontend riêng.

---

## Docker

```bash
cp .env.example .env
openssl rand -hex 32   # three times, one per secret
$EDITOR .env
docker compose up -d
```

Mở `http://<host>:3434`. Wizard cài đặt chạy ở lần khởi động đầu tiên và tạo tài
khoản quản trị gốc.

File compose đặt `SERVER_HOST`, `SERVER_PORT` và `DATA_DIR` cho container. Đừng sửa
chúng — riêng `SERVER_HOST` phải giữ là `0.0.0.0`, nếu không port đã publish sẽ
không truy cập được từ host.

### Đằng sau reverse proxy

Không cần thêm gì — file compose đã mặc định `TRUST_PROXY=true`, vì gần như mọi bản
triển khai bằng compose đều nằm sau proxy. Chỉ cần cấu hình proxy chuyển tiếp
`X-Forwarded-For` và `X-Forwarded-Proto` — xem [Reverse Proxy](reverse-proxy.md).

**Publish port thẳng ra internet mà không có proxy? Hãy đặt `TRUST_PROXY=false`
trong `.env`.** Request đi qua một port đã publish sẽ đến từ Docker bridge
(`172.17.0.1`), là một địa chỉ *private*, nên `true` tin cậy mọi khách truy cập trực
tiếp đúng như tin cậy một proxy thật. Những khách đó khi ấy có thể tự đặt
`X-Forwarded-For` để có một rate-limit bucket mới cho mỗi request — vô hiệu hoá hoàn
toàn limiter đăng nhập — và giả mạo `X-Forwarded-Proto: https` để cookie đăng nhập
được gắn `Secure` trên HTTP thuần, thứ mà trình duyệt âm thầm loại bỏ và biểu hiện ra
thành "sai mật khẩu".

Nếu proxy chạy trên cùng máy, hãy publish vào loopback để không thứ gì khác chạm được
tới container:

```yaml
ports:
  - "127.0.0.1:3434:3434"
```

### Dữ liệu

Mọi thứ nằm trong volume `novelhub_data`, mount tại `/data`:

```
/data
├── novelhub.db      SQLite database
├── books/           imported books and covers
├── calibre/         Calibre libraries available for import
├── inbox/           drop files here for automatic import
├── uploads/         in-progress chunked uploads
├── public/          uploaded site logo and favicon
├── logs/            rotating application logs
└── backups/         database backups
```

Để dùng một thư mục trên host thay cho named volume:

```yaml
volumes:
  - /srv/novelhub:/data
```

Container chạy bằng root và sẽ tự tạo nội dung bên trong thư mục.

### Cập nhật

```bash
docker compose pull
docker compose up -d
```

Schema mới được áp dụng lúc khởi động. Hãy backup trước — xem bên dưới.

### Sức khoẻ container

Image có sẵn healthcheck, gọi `/api/v1/health` mỗi 30 giây sau 20 giây chờ khởi động,
nên `docker compose ps` báo đúng trạng thái thật của container chứ không chỉ "running":

```bash
docker compose ps          # cột STATUS hiện healthy / unhealthy
curl http://127.0.0.1:3434/api/v1/health
```

### Log

```bash
docker compose logs -f
```

Cũng được ghi vào `/data/logs/novelhub.log`, rotate ở mốc 10 MB và giữ 5 file.

---

## Native

Yêu cầu Go 1.26+ và [Bun](https://bun.sh).

```bash
git clone https://github.com/AzenKain/Novel-Hub.git
cd Novel-Hub
cp .env.example .env
openssl rand -hex 32   # three times
$EDITOR .env

make run
```

`make run` build frontend và khởi động server. Để tạo binary độc lập:

```bash
make build
./novelhub
```

Binary là file duy nhất: schema và giao diện web đã nằm sẵn bên trong, không cần copy gì kèm theo. Nó tự tạo database và áp schema lúc khởi động.

### systemd

`/etc/systemd/system/novelhub.service`:

```ini
[Unit]
Description=NovelHub
After=network.target

[Service]
Type=simple
User=novelhub
WorkingDirectory=/opt/novelhub
EnvironmentFile=/opt/novelhub/.env
ExecStart=/opt/novelhub/novelhub
Restart=always
RestartSec=5

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/opt/novelhub/data

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now novelhub
sudo journalctl -u novelhub -f
```

`Restart=always` cũng là điều kiện để dùng `RESTORE_AUTO_RESTART=true`, giúp một lần
restore database hoàn tất mà không cần can thiệp thủ công.

---

## Backup

**Admin → Operations → Backups** tạo snapshot SQLite nhất quán ngay khi server đang
chạy, có thể chỉ database hoặc database kèm file sách. Lên lịch tại
**Operations → Schedules**.

Bản restore được chuẩn bị (stage) và kiểm tra trước, sau đó áp dụng ở lần khởi động
kế tiếp. Với `RESTORE_AUTO_RESTART=true` (mặc định của Docker), NovelHub tự thoát để
supervisor khởi động lại nó tự động. Nếu không, hãy tự khởi động lại khi giao diện
admin báo bản restore đã sẵn sàng.

Backup từ bên ngoài: dừng server rồi copy `DATA_DIR`. Copy `novelhub.db` khi server
đang chạy có thể chụp phải một lần ghi dở dang — hãy dùng chức năng backup trong
admin, nó xử lý việc này đúng cách.

---

## Nhập sách

Năm đường:

| Đường | Cách làm |
|---|---|
| Upload | **Admin → Books → Upload**, chia chunk để file lớn vẫn qua được kết nối không ổn định |
| Inbox | Thả file vào `data/inbox/<libraryID>/`, rồi chạy **Operations → Jobs → Scan inbox**. Thư mục lồng nhau được quét sâu tới 5 cấp; file đã nhập sẽ bị xóa và thư mục rỗng được dọn sạch |
| Calibre | **Admin → Library → Import from Calibre**, trỏ vào thư mục chứa `metadata.db` |
| Podcast | **Podcasts → Subscribe**, dán URL RSS podcast feed. Các tập tự động tải về dưới dạng file audio (.mp3, .m4a, .m4b, .flac) và được thêm vào thư viện |
| Conversion | **Admin → Books → Convert** (hoặc bulk convert). Chuyển đổi các định dạng file sách có sẵn sang định dạng khác (epub, kepub, mobi, azw, docx, fb2, cbz, txt, pdf) trực tiếp trên server |

Quá trình quét inbox chờ 10 giây sau khi một file ngừng thay đổi mới nhập, nên không
bao giờ nhặt phải bản copy dở dang.

---

## Ứng dụng đọc sách

| Giao thức / Ứng dụng | Endpoint | Xác thực |
|---|---|---|
| OPDS 1.2 | `/api/opds/v1` | HTTP Basic — email và mật khẩu NovelHub của bạn |
| OPDS 2.0 | `/api/opds/v2/catalog` | HTTP Basic |
| Kobo | `/kobo/<token>/v1/…` | Token nằm trong path — máy Kobo không gửi header Authorization |
| Mihon / Tachiyomi | `/komga/api/v1` | HTTP Basic, hoặc `X-API-Key: <email>:<mật khẩu>` |
| VBook (Android) | `/api/v1/vbook/plugin.json` | Không cần (plugin registry) |
| Magic Code (eReader) | `/api/v1/magic-code/request` | Xác thực qua token thăm dò (polling) |

Hoạt động với KOReader, Calibre, Moon+ Reader, Thorium, VBook và các client OPDS khác.

Với Mihon (trước là Tachiyomi), cài extension **Komga** gốc rồi trỏ vào
`http://<host>:3434/komga`. Không phải sửa gì phía client — NovelHub trả lời đúng
REST API Komga mà extension đó vốn đã nói, phục vụ từng trang truyện đọc thẳng từ
file CBZ/CBR. Tiến độ đọc đồng bộ hai chiều qua tracker Komga có sẵn trong Mihon.
Cần quyền `komga.sync`.

Với **VBook**, sao chép link plugin JSON hoặc quét mã QR từ trang cá nhân để cài đặt extension NovelHub trong VBook. Bạn có thể duyệt, tìm kiếm và đọc sách trực tiếp từ điện thoại.

Với **Magic Code**, các thiết bị đọc sách (e-reader) hoặc thiết bị thông minh bị hạn chế về bàn phím có thể đăng nhập không cần mật khẩu. Chọn đăng nhập bằng mã trên thiết bị (thiết bị hiển thị mã 6 chữ số), sau đó vào **Trang cá nhân → Kích hoạt thiết bị** trên trình duyệt máy tính/điện thoại đã đăng nhập, nhập mã 6 chữ số và thiết bị sẽ tự động được xác thực.

Endpoint Kobo không tự gõ tay: vào **Trang cá nhân → Kobo Sync** rồi copy URL đã
sinh, trong đó có token bí mật riêng của từng người. Hãy coi nó như mật khẩu — ai
giữ nó là truy cập được cả thư viện của bạn.

OPDS chỉ bị rate limit khi xác thực *thất bại*, nên việc poll bình thường không bao
giờ bị chặn. Nếu link catalog trỏ sai host — chẳng hạn khi nằm sau proxy có rewrite
path — hãy đặt **URL máy chủ** trong **Admin → Cài đặt** thành base URL tuyệt đối
đúng. Nó có hiệu lực ngay, không cần restart.


---

## Xử lý sự cố

**Không truy cập được server trong Docker.** `SERVER_HOST` phải là `0.0.0.0` bên
trong container. Nếu `.env` đặt `127.0.0.1`, hãy xóa dòng đó; file compose đã đặt
giá trị đúng.

**Cookie đăng nhập không có cờ `Secure` dù chạy HTTPS.** `TRUST_PROXY` chưa được
đặt, không bao phủ địa chỉ của proxy, hoặc proxy không gửi `X-Forwarded-Proto`. Xem
[Reverse Proxy](reverse-proxy.md#bước-3--kiểm-tra).

**Mọi người dùng chung một rate-limit bucket.** Cùng nguyên nhân. Không có
`TRUST_PROXY`, mọi request trông như đều đến từ proxy.

**Lỗi `413` khi upload.** Do giới hạn body của proxy, không phải của NovelHub. nginx
mặc định 1 MB; hãy đặt `client_max_body_size 0`.

**Mất database sau khi khởi động lại.** `SQLITE_DB_PATH` hoặc `DATA_DIR` trỏ ra ngoài
volume đã mount. Trong Docker cả hai đều được file compose đặt đúng — hãy kiểm tra
xem `.env` có ghi đè chúng không.

**Bị khóa sau một lần sai mật khẩu.** Đã sửa ở các phiên bản hiện tại. Các bản build
cũ cache một password hash rỗng sau một lần thử thất bại, khiến mật khẩu đúng cũng bị
từ chối cho tới khi cache hết hạn. Hãy cập nhật.
