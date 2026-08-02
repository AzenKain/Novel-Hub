# Deployment

Two ways to run NovelHub: Docker, or a native binary. Both need the same three
secrets from [Configuration](configuration.md).

The frontend is compiled into the binary, so there is one process, one port and
one directory to back up. No web server, no separate frontend host.

---

## Docker

```bash
cp .env.example .env
openssl rand -hex 32   # three times, one per secret
$EDITOR .env
docker compose up -d
```

Open `http://<host>:3434`. The setup wizard runs on first launch and creates the
root administrator.

The compose file sets `SERVER_HOST`, `SERVER_PORT` and `DATA_DIR` for the
container. Leave those alone — `SERVER_HOST` in particular must stay `0.0.0.0`
or the published port cannot be reached from the host.

### Behind a reverse proxy

Add to `.env`:

```bash
TRUST_PROXY=true
```

Then configure the proxy to forward `X-Forwarded-For` and `X-Forwarded-Proto` —
see [Reverse Proxy](reverse-proxy.md).

`TRUST_PROXY` is not enabled by default in Docker, even though many Docker
deployments have a proxy. Requests through a published port arrive from the
Docker bridge (`172.17.0.1`), which is a *private* address — so `true` would
equally trust every direct visitor on a plain `docker compose up`. Those visitors
could then set `X-Forwarded-For` themselves to get a fresh rate-limit bucket per
request, and forge `X-Forwarded-Proto: https` so the login cookie gets `Secure`
over plain HTTP, which browsers silently drop.

If the proxy runs on the same host, publish to loopback so nothing else can
reach the container:

```yaml
ports:
  - "127.0.0.1:3434:3434"
```

### Data

Everything lives in the `novelhub_data` volume, mounted at `/data`:

```
/data
├── novelhub.db      SQLite database
├── books/           imported books and covers
├── inbox/           drop files here for automatic import
├── uploads/         in-progress chunked uploads
├── logs/            rotating application logs
└── backups/         database backups
```

To use a host directory instead of a named volume:

```yaml
volumes:
  - /srv/novelhub:/data
```

The container runs as root and will create the directory contents itself.

### Updating

```bash
docker compose pull
docker compose up -d
```

Schema migrations apply automatically at startup. Back up first — see below.

### Logs

```bash
docker compose logs -f
```

Also written to `/data/logs/novelhub.log`, rotating at 10 MB with 5 files kept.

---

## Native

Requires Go 1.26+ and [Bun](https://bun.sh).

```bash
git clone https://github.com/AzenKain/Novel-Hub.git
cd Novel-Hub
cp .env.example .env
openssl rand -hex 32   # three times
$EDITOR .env

make run
```

`make run` builds the frontend and starts the server. For a standalone binary:

```bash
make build
./novelhub
```

The binary needs `db/schema/` beside it — it applies schema files at startup.

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

`Restart=always` also enables `RESTORE_AUTO_RESTART=true`, which lets a database
restore complete without manual intervention.

---

## Backups

**Admin → Operations → Backups** creates consistent SQLite snapshots while the
server is running, either database-only or database plus book files. Schedule
them under **Operations → Schedules**.

Restores are staged and validated first, then applied on the next start. With
`RESTORE_AUTO_RESTART=true` (the Docker default) NovelHub exits so the supervisor
restarts it automatically. Otherwise restart it yourself once the admin UI says
the restore is ready.

Backing up externally: stop the server and copy `DATA_DIR`. Copying `novelhub.db`
while the server is running can capture a torn write — use the admin backup
instead, which handles this correctly.

---

## Importing books

Three routes:

| Route | How |
|---|---|
| Upload | **Admin → Books → Upload**, chunked so large files survive a flaky connection |
| Inbox | Drop files into `data/inbox/<libraryID>/`, then run **Operations → Jobs → Scan inbox**. Nested folders are scanned up to 5 levels deep; imported files are removed and empty directories cleaned up |
| Calibre | **Admin → Library → Import from Calibre**, pointed at a folder containing `metadata.db` |

Inbox scanning waits 10 seconds after a file stops changing before importing, so
a partial copy is never picked up.

---

## E-reader clients

| Protocol | Endpoint | Auth |
|---|---|---|
| OPDS 1.2 | `/opds/v1` | HTTP Basic — your NovelHub email and password |
| OPDS 2.0 | `/opds/v2/catalog` | HTTP Basic |
| Kobo | `/kobo/v1` | Bearer token |

Works with KOReader, Calibre, Moon+ Reader, Thorium and other OPDS clients.

OPDS is rate-limited on *failed* authentication only, so normal polling is never
throttled. If catalog links point at the wrong host — behind a path-rewriting
proxy, for instance — set `SERVER_URL` to the correct absolute base URL.

---

## Troubleshooting

**Cannot reach the server in Docker.** `SERVER_HOST` must be `0.0.0.0` inside the
container. If `.env` sets `127.0.0.1`, remove it; the compose file already sets
the right value.

**Login cookie has no `Secure` flag over HTTPS.** `TRUST_PROXY` is unset, does not
cover the proxy's address, or the proxy is not sending `X-Forwarded-Proto`. See
[Reverse Proxy](reverse-proxy.md#step-3--verify).

**Everyone shares one rate-limit bucket.** Same cause. Without `TRUST_PROXY`,
every request appears to come from the proxy.

**`413` on upload.** The proxy's body limit, not NovelHub's. nginx defaults to
1 MB; set `client_max_body_size 0`.

**Database missing after restart.** `SQLITE_DB_PATH` or `DATA_DIR` pointed outside
the mounted volume. In Docker both are set correctly by the compose file — check
whether `.env` overrides them.

**Locked out after one wrong password.** Fixed in current versions. Older builds
cached an empty password hash after a failed attempt, rejecting the correct
password until the cache expired. Update.
