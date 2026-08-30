# 反向代理

NovelHub 只提供纯 HTTP 服务。要通过 HTTPS 访问它,你需要在前面放一个反向代理 ——
nginx、Caddy、Traefik 或 Cloudflare —— 由它终结 TLS 并转发请求。

代理会把真实客户端信息挡在 NovelHub 之外。有两个请求头负责把这些信息带回来,而
NovelHub 默认忽略它们,因为任何客户端都能伪造。`TRUST_PROXY` 规定了谁才有权设置它们。

这里有两半工作,两者都是必需的。

| 一半 | 位置 |
| --- | --- |
| NovelHub 信任该代理 | `.env` 中的 `TRUST_PROXY` |
| 代理发送这些请求头 | 你的代理自身配置 |

漏掉任意一半,症状都一样:所有人共用同一个限流计数桶,而且登录 Cookie 永远拿不到
`Secure`。

---

## 第 1 步 —— 信任代理

在 `.env` 中:

```bash
TRUST_PROXY=true
```

`true` 覆盖位于回环、私有或链路本地地址上的代理,几乎所有自托管环境都属于这一类:
同一台机器上的 nginx 或 Caddy,或是同一 Docker 网络内的另一个容器。

如果代理是从**公网**地址访问 NovelHub 的 —— Cloudflare,或者位于另一台主机上的代理
—— 那就改为逐个列出地址:

```bash
TRUST_PROXY=173.245.48.0/20,103.21.244.0/22,103.22.200.0/22
```

Cloudflare 在 <https://www.cloudflare.com/ips/> 公布其当前的地址段。把它们全部列上,
用逗号分隔。

修改此项后需要重启 NovelHub。

---

## 第 2 步 —— 发送请求头

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

Caddy 会自动发送这三个请求头并自动申请证书。不需要任何额外配置。

### Traefik(docker-compose 标签)

```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.novelhub.rule=Host(`library.example.com`)"
  - "traefik.http.routers.novelhub.entrypoints=websecure"
  - "traefik.http.routers.novelhub.tls.certresolver=letsencrypt"
  - "traefik.http.services.novelhub.loadbalancer.server.port=3434"
```

Traefik 默认就会发送这些转发请求头。

### Cloudflare

Cloudflare 会自行发送 `X-Forwarded-For` 和 `X-Forwarded-Proto`,但它是从公网地址接入的,
所以 `TRUST_PROXY=true` **不会**覆盖它。请按第 1 步所示列出其公布的地址段。

把 SSL/TLS 模式设为 **Full** 或 **Full (strict)**。在 Flexible 模式下,Cloudflare 用纯
HTTP 与你的源站通信,却告诉浏览器该站点是 HTTPS,这会以各种令人费解的方式破坏 Cookie。

---

## 第 3 步 —— 验证

用 HTTPS 登录,然后打开 **DevTools → Application → Cookies**。`access_token` 那一行
必须显示 `Secure`。

如果没有,就倒着排查:

1. `TRUST_PROXY` 设了吗?设完重启 NovelHub 了吗?
2. 它的取值覆盖了代理接入时所用的地址吗?查看 NovelHub 的日志,找出某个进入请求的
   源 IP —— 如果是公网地址,`true` 就不够用,你需要显式列出地址。
3. 代理有在发送 `X-Forwarded-Proto` 吗?从代理这一侧确认:

```bash
curl -sI https://library.example.com/api/v1/health
```

---

## 限制仅代理可访问

代理与 NovelHub 在同一台主机时,别的东西都不需要访问 NovelHub。把它绑定到回环地址,
这样端口就不会暴露在你的局域网上。

原生安装,在 `.env` 中:

```bash
SERVER_HOST=127.0.0.1
```

Docker,在 `docker-compose.yml` 中:

```yaml
ports:
  - "127.0.0.1:3434:3434"
```

---

## 备注

**部署在子路径下**(`example.com/novelhub`)不受支持。前端是以绝对资源路径构建的。
请改用子域名。

**WebSocket** 未被使用,所以不需要任何 upgrade 请求头。

**大文件上传** 需要提高代理的请求体大小上限。nginx 默认为 1 MB,在你设置
`client_max_body_size` 之前,它会以 `413` 拒绝上传书籍。Caddy 和 Traefik 没有这类默认限制。
