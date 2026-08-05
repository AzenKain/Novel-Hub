# 部署

运行 NovelHub 有两种方式:Docker,或原生二进制文件。两者都需要
[配置](configuration.md)中提到的那三个密钥。

前端被编译进了二进制文件,所以只有一个进程、一个端口、一个需要备份的目录。不需要
web 服务器,也不需要单独的前端主机。

---

## Docker

```bash
cp .env.example .env
openssl rand -hex 32   # three times, one per secret
$EDITOR .env
docker compose up -d
```

打开 `http://<host>:3434`。首次启动会运行安装向导并创建根管理员。

compose 文件已为容器设置了 `SERVER_HOST`、`SERVER_PORT` 和 `DATA_DIR`。别去改动它们
—— 尤其是 `SERVER_HOST` 必须保持为 `0.0.0.0`,否则从宿主机无法访问映射出去的端口。

### 在反向代理之后

无需添加任何配置 —— compose 文件已经把 `TRUST_PROXY` 默认设为 `true`,因为几乎所有
用 compose 的部署都位于代理之后。你只需配置代理转发 `X-Forwarded-For` 和
`X-Forwarded-Proto` —— 参见[反向代理](reverse-proxy.md)。

**如果没有代理、直接把端口暴露到公网,请在 `.env` 中设置 `TRUST_PROXY=false`。**
通过映射端口进来的请求来自 Docker 网桥(`172.17.0.1`),那是一个*私有*地址,所以
`true` 会像信任真正的代理一样信任每一个直连访客。这些访客就能自己设置
`X-Forwarded-For`,让每个请求都拿到一个全新的限流计数桶 —— 登录限流被完全绕过 ——
还能伪造 `X-Forwarded-Proto: https`,使登录 Cookie 在纯 HTTP 上也带上 `Secure`,
而浏览器会静默丢弃它,表现出来就是"密码错误"。

如果代理运行在同一台主机上,就把端口只映射到回环地址,这样别的东西都无法访问容器:

```yaml
ports:
  - "127.0.0.1:3434:3434"
```

### 数据

所有内容都存放在 `novelhub_data` 卷中,挂载于 `/data`:

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

如果想用宿主机目录代替命名卷:

```yaml
volumes:
  - /srv/novelhub:/data
```

容器以 root 运行,会自行创建目录内容。

### 更新

```bash
docker compose pull
docker compose up -d
```

新的结构定义会在启动时应用。先做备份 —— 见下文。

### 健康检查

镜像内置了健康检查:在 20 秒的启动宽限期之后,每 30 秒探测一次 `/api/v1/health`,
因此 `docker compose ps` 报告的是容器的真实状态,而不只是 "running":

```bash
docker compose ps          # STATUS 列会显示 healthy / unhealthy
curl http://127.0.0.1:3434/api/v1/health
```

### 日志

```bash
docker compose logs -f
```

同时也会写入 `/data/logs/novelhub.log`,达到 10 MB 时轮转,保留 5 个文件。

---

## 原生部署

需要 Go 1.26+ 和 [Bun](https://bun.sh)。

```bash
git clone https://github.com/AzenKain/Novel-Hub.git
cd Novel-Hub
cp .env.example .env
openssl rand -hex 32   # three times
$EDITOR .env

make run
```

`make run` 会构建前端并启动服务器。如果需要独立的二进制文件:

```bash
make build
./novelhub
```

二进制文件是自包含的：结构定义和 Web 界面都已嵌入其中，无需在旁边放置任何文件。它会在启动时创建数据库并应用结构定义。

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

有了 `Restart=always`,也就可以启用 `RESTORE_AUTO_RESTART=true`,让数据库恢复无需人工
干预即可完成。

---

## 备份

**管理后台 → 运维 → 备份** 会在服务器运行期间创建一致的 SQLite 快照,可以只备份数据库,
也可以连书籍文件一起备份。在 **运维 → 计划任务** 中安排定时备份。

恢复操作会先暂存并校验,然后在下次启动时应用。当 `RESTORE_AUTO_RESTART=true`
(Docker 的默认值)时,NovelHub 会退出,由进程管理器自动把它重启起来。否则就等管理
后台提示恢复已就绪后,自行重启。

从外部做备份:停掉服务器,然后复制 `DATA_DIR`。在服务器运行时复制 `novelhub.db` 可能
拷到一个写入中途被撕裂的文件 —— 请改用管理后台的备份功能,它会正确处理这种情况。

---

## 导入书籍

三种途径:

| 途径 | 做法 |
|---|---|
| 上传 | **管理后台 → 书籍 → 上传**,分片上传,让大文件也能扛住不稳定的网络 |
| Inbox | 把文件放进 `data/inbox/<libraryID>/`,然后运行 **运维 → 任务 → 扫描 inbox**。嵌套文件夹最多扫描 5 层;导入完成的文件会被删除,空目录会被清理 |
| Calibre | **管理后台 → 书库 → 从 Calibre 导入**,指向一个包含 `metadata.db` 的文件夹 |

Inbox 扫描会在文件停止变化后再等 10 秒才导入,因此绝不会捡到一个复制了一半的文件。

---

## 电子阅读器客户端

| 协议 | 端点 | 认证 |
|---|---|---|
| OPDS 1.2 | `/api/opds/v1` | HTTP Basic —— 你的 NovelHub 邮箱和密码 |
| OPDS 2.0 | `/api/opds/v2/catalog` | HTTP Basic |
| Kobo | `/kobo/<token>/v1/…` | 路径中的令牌 —— Kobo 不会发送 Authorization 头 |

可配合 KOReader、Calibre、Moon+ Reader、Thorium 以及其他 OPDS 客户端使用。

Kobo 端点不需要手动输入:打开 **个人资料 → Kobo 同步**,复制生成的 URL,其中包含每位
用户各自的密钥令牌。请把它当作密码看待 —— 拿到它的人就能访问你的整个书库。

OPDS 仅对认证*失败*的请求限流,因此正常轮询永远不会被限流。如果目录链接指向了错误的
主机 —— 例如位于会重写路径的代理之后 —— 请在 **管理 → 设置** 中把 **服务器 URL** 设为
正确的绝对基础 URL。它会立即生效,无需重启。

---

## 故障排查

**在 Docker 中无法访问服务器。** 容器内的 `SERVER_HOST` 必须是 `0.0.0.0`。如果 `.env`
里设成了 `127.0.0.1`,把它删掉;compose 文件已经设了正确的值。

**HTTPS 下登录 Cookie 没有 `Secure` 标记。** `TRUST_PROXY` 没设置、没覆盖代理的地址,
或者代理没有发送 `X-Forwarded-Proto`。参见
[反向代理](reverse-proxy.md#第-3-步--验证)。

**所有人共用同一个限流计数桶。** 同样的原因。没有 `TRUST_PROXY`,每个请求看起来都来自
代理。

**上传时返回 `413`。** 这是代理的请求体大小上限,不是 NovelHub 的。nginx 默认为 1 MB;
设置 `client_max_body_size 0`。

**重启后数据库不见了。** `SQLITE_DB_PATH` 或 `DATA_DIR` 指向了挂载卷之外的位置。在
Docker 中这两者都由 compose 文件正确设置 —— 检查是不是 `.env` 覆盖了它们。

**输错一次密码就被锁在外面。** 当前版本已修复。较旧的构建在一次失败尝试后会缓存一个
空的密码哈希,导致在缓存过期前连正确的密码也被拒绝。请更新版本。
