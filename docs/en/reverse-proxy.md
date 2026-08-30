# Reverse Proxy

NovelHub serves plain HTTP. To reach it over HTTPS you put a reverse proxy in
front — nginx, Caddy, Traefik or Cloudflare — which terminates TLS and forwards
the request.

The proxy hides the real client from NovelHub. Two headers carry that
information back, and NovelHub ignores them by default because any client can
forge them. `TRUST_PROXY` says who is allowed to set them.

There are two halves. Both are required.

| Half | Where |
| --- | --- |
| NovelHub trusts the proxy | `TRUST_PROXY` in `.env` |
| The proxy sends the headers | Your proxy's configuration |

Miss either one and you get the same symptoms: everyone shares one rate-limit
bucket, and the login cookie never gets `Secure`.

---

## Step 1 — Trust the proxy

In `.env`:

```bash
TRUST_PROXY=true
```

`true` covers proxies on loopback, private or link-local addresses, which is
almost every self-hosted setup: nginx or Caddy on the same machine, or another
container on the same Docker network.

If the proxy reaches NovelHub from a **public** address — Cloudflare, or a proxy
on a separate host — list the addresses instead:

```bash
TRUST_PROXY=173.245.48.0/20,103.21.244.0/22,103.22.200.0/22
```

Cloudflare publishes its current ranges at <https://www.cloudflare.com/ips/>.
List all of them, comma-separated.

Restart NovelHub after changing this.

---

## Step 2 — Send the headers

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

Caddy sends all three headers and obtains a certificate automatically. Nothing
else is needed.

### Traefik (docker-compose labels)

```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.novelhub.rule=Host(`library.example.com`)"
  - "traefik.http.routers.novelhub.entrypoints=websecure"
  - "traefik.http.routers.novelhub.tls.certresolver=letsencrypt"
  - "traefik.http.services.novelhub.loadbalancer.server.port=3434"
```

Traefik sends the forwarded headers by default.

### Cloudflare

Cloudflare sends `X-Forwarded-For` and `X-Forwarded-Proto` on its own, but
connects from public addresses, so `TRUST_PROXY=true` will **not** cover it. List
the published ranges as shown in step 1.

Set the SSL/TLS mode to **Full** or **Full (strict)**. In Flexible mode
Cloudflare talks to your origin over plain HTTP while telling browsers the site
is HTTPS, which breaks cookies in confusing ways.

---

## Step 3 — Verify

Sign in over HTTPS, then open **DevTools → Application → Cookies**. The
`access_token` row must show `Secure`.

If it does not, work backwards:

1. Is `TRUST_PROXY` set, and did you restart NovelHub?
2. Does its value cover the address the proxy connects from? Check NovelHub's
   logs for the source IP of an incoming request — if it is public, `true` is not
   enough and you need an explicit list.
3. Is the proxy sending `X-Forwarded-Proto`? Confirm from the proxy side:

```bash
curl -sI https://library.example.com/api/v1/health
```

---

## Restricting access to the proxy

With a proxy on the same host, nothing else needs to reach NovelHub. Bind it to
loopback so the port is not exposed on your LAN.

Native install, in `.env`:

```bash
SERVER_HOST=127.0.0.1
```

Docker, in `docker-compose.yml`:

```yaml
ports:
  - "127.0.0.1:3434:3434"
```

---

## Notes

**Serving under a subpath** (`example.com/novelhub`) is not supported. The
frontend is built with absolute asset paths. Use a subdomain instead.

**WebSockets** are not used, so no upgrade headers are required.

**Large uploads** need the proxy's body limit raised. nginx defaults to 1 MB and
will reject book uploads with `413` until you set `client_max_body_size`. Caddy
and Traefik have no such default.
