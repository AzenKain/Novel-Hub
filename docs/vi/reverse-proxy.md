# Reverse Proxy

NovelHub phục vụ HTTP thuần. Để truy cập qua HTTPS, bạn đặt một reverse proxy đứng
trước — nginx, Caddy, Traefik hoặc Cloudflare — để nó kết thúc TLS và chuyển tiếp
request.

Proxy che mất client thật khỏi NovelHub. Hai header mang thông tin đó trở lại, và
NovelHub mặc định bỏ qua chúng vì bất kỳ client nào cũng có thể giả mạo.
`TRUST_PROXY` chỉ ra ai được phép đặt chúng.

Có hai nửa. Cả hai đều bắt buộc.

| Nửa | Ở đâu |
| --- | --- |
| NovelHub tin cậy proxy | `TRUST_PROXY` trong `.env` |
| Proxy gửi các header | Cấu hình của proxy |

Thiếu một trong hai thì triệu chứng như nhau: tất cả mọi người dùng chung một
rate-limit bucket, và cookie đăng nhập không bao giờ có `Secure`.

---

## Bước 1 — Tin cậy proxy

Trong `.env`:

```bash
TRUST_PROXY=true
```

`true` bao phủ các proxy nằm trên địa chỉ loopback, private hoặc link-local, tức là
gần như mọi thiết lập self-host: nginx hoặc Caddy trên cùng máy, hoặc một container
khác trong cùng Docker network.

Nếu proxy kết nối tới NovelHub từ một địa chỉ **public** — Cloudflare, hoặc proxy
nằm trên máy khác — hãy liệt kê địa chỉ thay vì dùng `true`:

```bash
TRUST_PROXY=173.245.48.0/20,103.21.244.0/22,103.22.200.0/22
```

Cloudflare công bố các dải hiện hành tại <https://www.cloudflare.com/ips/>. Liệt kê
toàn bộ, phân tách bằng dấu phẩy.

Khởi động lại NovelHub sau khi thay đổi giá trị này.

---

## Bước 2 — Gửi các header

### nginx

```nginx
server {
    listen 443 ssl;
    server_name library.example.com;

    ssl_certificate     /etc/letsencrypt/live/library.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/library.example.com/privkey.pem;

    # Book uploads are large; the default 1m rejects them.
    client_max_body_size 0;

    location / {
        proxy_pass http://127.0.0.1:3434;

        proxy_set_header Host              $host;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Long enough for large uploads and scans.
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
}

server {
    listen 80;
    server_name library.example.com;
    return 301 https://$host$request_uri;
}
```

### Caddy

```caddy
library.example.com {
    reverse_proxy 127.0.0.1:3434
}
```

Caddy gửi cả ba header và tự động lấy chứng chỉ. Không cần gì thêm.

### Traefik (label trong docker-compose)

```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.novelhub.rule=Host(`library.example.com`)"
  - "traefik.http.routers.novelhub.entrypoints=websecure"
  - "traefik.http.routers.novelhub.tls.certresolver=letsencrypt"
  - "traefik.http.services.novelhub.loadbalancer.server.port=3434"
```

Traefik gửi các forwarded header theo mặc định.

### Cloudflare

Cloudflare tự gửi `X-Forwarded-For` và `X-Forwarded-Proto`, nhưng kết nối từ địa chỉ
public, nên `TRUST_PROXY=true` sẽ **không** bao phủ nó. Hãy liệt kê các dải đã công
bố như ở bước 1.

Đặt chế độ SSL/TLS là **Full** hoặc **Full (strict)**. Ở chế độ Flexible, Cloudflare
nói chuyện với origin của bạn qua HTTP thuần trong khi vẫn báo với trình duyệt rằng
site là HTTPS, làm cookie lỗi theo những cách rất khó hiểu.

---

## Bước 3 — Kiểm tra

Đăng nhập qua HTTPS, rồi mở **DevTools → Application → Cookies**. Dòng
`access_token` phải hiện `Secure`.

Nếu không, hãy truy ngược lại:

1. `TRUST_PROXY` đã được đặt chưa, và bạn đã khởi động lại NovelHub chưa?
2. Giá trị của nó có bao phủ địa chỉ mà proxy kết nối tới không? Kiểm tra log của
   NovelHub để xem IP nguồn của một request đến — nếu là địa chỉ public thì `true`
   không đủ và bạn cần một danh sách tường minh.
3. Proxy có đang gửi `X-Forwarded-Proto` không? Xác nhận từ phía proxy:

```bash
curl -sI https://library.example.com/api/v1/health
```

---

## Giới hạn truy cập chỉ cho proxy

Khi proxy nằm trên cùng máy, không cần thứ gì khác chạm tới NovelHub. Hãy bind nó
vào loopback để port không bị mở ra LAN.

Cài đặt native, trong `.env`:

```bash
SERVER_HOST=127.0.0.1
```

Docker, trong `docker-compose.yml`:

```yaml
ports:
  - "127.0.0.1:3434:3434"
```

---

## Ghi chú

**Phục vụ dưới một subpath** (`example.com/novelhub`) không được hỗ trợ. Frontend
được build với đường dẫn asset tuyệt đối. Hãy dùng subdomain thay thế.

**WebSocket** không được sử dụng, nên không cần header upgrade nào.

**Upload lớn** đòi hỏi nâng giới hạn body của proxy. nginx mặc định 1 MB và sẽ từ
chối các lần upload sách với mã `413` cho đến khi bạn đặt `client_max_body_size`.
Caddy và Traefik không có mặc định như vậy.
