# 配置

NovelHub 把配置分成两部分。环境变量负责那些在数据库打开之前就必须知道的东西 ——
或者那些用来保护数据库、因此不能存在数据库里的东西。其余全部在管理后台里设置,
改完无需重启即可生效。

真正重要的只有四个变量。其余都有可用的默认值,而且大多会根据机器自动调优。

---

## 必填

把 `.env.example` 复制为 `.env`,填入三个密钥。没有它们,服务器不会签发令牌。

```bash
cp .env.example .env
openssl rand -hex 32   # run three times, one value each
```

| 变量 | 用途 |
|---|---|
| `JWT_SECRET` | 为访问令牌签名 |
| `JWT_REFRESH_SECRET` | 为刷新令牌签名 |
| `DB_ENCRYPTION_KEY` | 加密存放在数据库里的第三方令牌(AniList、MAL) |

每个都要用不同的随机值。

**之后再修改。** 修改任意一个 JWT 密钥会让所有人被登出 —— 不会丢失数据。修改
`DB_ENCRYPTION_KEY` 会让已加密的追踪器令牌永久无法解读;用户必须重新关联那些账号。
动手之前先备份数据库。

它们之所以保留为环境变量,是因为它们签名和加密的正是数据库*里面*的内容。把它们存进
数据库会形成循环依赖。

---

## 反向代理

```bash
TRUST_PROXY=false
```

它决定 NovelHub 是否相信代理发来的这两个请求头:

- `X-Forwarded-For` —— 客户端到底是谁,用于限流
- `X-Forwarded-Proto` —— 原始请求是否为 HTTPS,它决定登录 Cookie 是否带上
  `Secure` 标记

| 取值 | 适用场景 | 信任范围 |
|---|---|---|
| `false` | 浏览器直连 NovelHub | 什么都不信任 |
| `true` | nginx/Caddy 与 NovelHub 在同一台主机,或是同一 Docker 网络内的另一个容器 | 回环、私有和链路本地地址:`127.0.0.0/8`、`::1`、`10.0.0.0/8`、`172.16.0.0/12`、`192.168.0.0/16`、`fc00::/7`、`169.254.0.0/16` |
| `1.2.3.0/24,5.6.7.8` | 代理位于公网地址上 —— 最常见的是 Cloudflare | 仅信任列出的 IP 和 CIDR |

前面没有真正的代理时,不要设成 `true`。否则任何客户端都能自己伪造这些头:每个请求
都能拿到一个全新的限流计数桶(登录限流器彻底失效),还能伪造 `https` 声明,让在
纯 HTTP 上下发的 Cookie 带上 `Secure`,而浏览器会静默丢弃这样的 Cookie。

设好它只完成了一半 —— 你的代理还得真的把这些头发出来。各代理的具体配置以及如何验证,
参见[反向代理](reverse-proxy.md)。

`TRUST_PROXY` 只在启动时读取一次,而且不能做成管理后台设置项:它决定登录 Cookie 是否
带 `Secure`,所以一旦数据库里存了错误的值,你就得先登录才能修好那个正在阻止你登录的
东西。

---

## 可选

下面所有项在 `.env.example` 里都是注释掉的。只取消注释你确实需要覆盖的那些。

### 网络

| 变量 | 默认值 | 说明 |
|---|---|---|
| `SERVER_HOST` | `127.0.0.1` | Docker 会设为 `0.0.0.0`;不要在那里覆盖它,否则映射出去的端口无法访问 |
| `SERVER_PORT` | `3434` | |
| `SERVER_URL` | — | OPDS 目录链接使用的绝对基础 URL。仅当自动探测到的主机不正确时才需要,例如位于会重写路径的代理之后 |

### 存储

| 变量 | 默认值 | 说明 |
|---|---|---|
| `DATA_DIR` | `./data` | 下面所有内容的根目录 |
| `SQLITE_DB_PATH` | `$DATA_DIR/novelhub.db` | |

`DATA_DIR` 包含:

```
data/
├── novelhub.db      SQLite database
├── books/           imported books and covers
├── inbox/           drop files here for automatic import
├── uploads/         in-progress chunked uploads
├── logs/            rotating application logs
└── backups/         database backups
```

备份了 `DATA_DIR`,就等于备份了整套安装。

### 性能

这些全部会自动调优。只有当你想刻意限制资源占用时才去设置。

| 变量 | 默认值 |
|---|---|
| `SQLITE_CACHE_SIZE_KB` | 按系统内存计算(64 MB–512 MB) |
| `SQLITE_MMAP_SIZE_BYTES` | 按系统内存计算(256 MB–2 GB) |
| `SQLITE_MAX_OPEN_CONNS` | CPU 核数 × 2,限制在 4–16 之间 |
| `SQLITE_MAX_IDLE_CONNS` | 与最大打开连接数相同 |
| `CACHE_MAX_COST_BYTES` | 按系统内存计算 |
| `JOB_WORKERS` | `1` —— 后台任务并发数 |
| `GOGC` | `200` —— Go GC 目标值;调低是用 CPU 换内存 |
| `FIBER_CONCURRENCY` | Fiber 默认值 |
| `FIBER_READ_BUFFER_SIZE` | Fiber 默认值 |
| `FIBER_WRITE_BUFFER_SIZE` | Fiber 默认值 |

### 日志

| 变量 | 默认值 | 说明 |
|---|---|---|
| `LOG_MAX_SIZE_MB` | `10` | 当前日志文件达到该大小时轮转 |
| `LOG_MAX_FILES` | `5` | 保留的轮转文件数 |
| `DISABLE_REQUEST_LOG` | `true` | 关闭逐请求日志以提升吞吐 |
| `DISABLE_STARTUP_MESSAGE` | `false` | |

这些保留为环境变量,是因为日志在数据库打开之前就已启动 —— 数据库出问题时必须还能被
记录下来。

### 行为

| 变量 | 默认值 | 说明 |
|---|---|---|
| `TOKEN_VERSION_CACHE` | `true` | 多个实例共用一个数据库时设为 `false`,那种情况下内存缓存会变成过期的脏数据 |
| `DISABLE_RESPONSE_COMPRESSION` | `false` | |
| `ENABLE_PREFORK` | `false` | 多进程 worker。会禁用令牌版本缓存 |
| `RESTORE_AUTO_RESTART` | `false` | 暂存数据库恢复后退出进程,让 Docker 或 systemd 重启并应用它。Docker 中设为 `true` |

---

## 在管理后台中配置

这些不是环境变量。首次启动时通过安装向导设置,之后在 **管理后台 → 设置** 中修改。
改动立即生效。

| 区域 | 涵盖内容 |
|---|---|
| 站点 | 标题、描述、Logo、favicon、侧边栏项目、首页板块 |
| 访问 | 注册开关、访客访问模式、按书库的访客可见性 |
| 权限 | 按角色控制阅读、下载、书签、收藏集、书评、分享 |
| 上传限制 | 分片大小、分片数量、并发会话数、总大小、会话 TTL、封面与站点素材大小 |
| 限流 | 每个时间窗内的登录与 OPDS 尝试次数,以及时间窗长度 |

### 限流

NovelHub 只对两件事限流,而且两者共用同一对设置:**登录**(`/api/v1/auth/*`)和
**OPDS**(`/opds/*`)。

两者都会执行 bcrypt 密码校验,每次大约消耗 50–100 ms 的 CPU —— 差不多是同一个请求里
其他所有工作总和的 600 倍。这才是值得保护的资源。OPDS 也算在内,是因为它使用
HTTP Basic 认证,不携带会话,所以*每个*请求都要跑一次 bcrypt。

默认值:每 60 秒 5 次尝试,按客户端 IP 计数。

对 OPDS 只统计失败的尝试。阅读器应用带着正确凭据轮询目录属于正常流量,永远不会被
限流。

这里刻意没有设置通用的 API 限流。一个漫画章节每页都渲染为一个图片请求,所以打开一卷
200 页的书正常就会发出 200 个请求 —— 通用限流只会限住读者,而不是攻击者。

---

## 认证 Cookie

无需配置;两个属性都按每个请求推导得出。

**`Secure`** 会在请求通过 HTTPS 到达时设置 —— 无论是直连,还是经由受信任代理的
`X-Forwarded-Proto`(见上文 `TRUST_PROXY`)。在纯 HTTP 上则会省略,因为浏览器会在
不安全连接上静默丢弃带 `Secure` 的 Cookie,而表现出来就像是密码错误。

**`Domain`** 从不设置,因此 Cookie 的作用域始终限定在下发它的那台主机上。

**`SameSite`** 为 `Lax`,并且应当保持这个值。NovelHub 既没有 CSRF token,也不做来源
(Origin)校验,所以 `SameSite` 是它唯一的 CSRF 防线。把它放宽成 `None` 会移除这层
保护,并且没有任何东西来替代它。

---

## 验证

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

在代理后面时,用 HTTPS 登录,然后查看 **DevTools → Application → Cookies**。
`access_token` 那一行必须显示 `Secure`。如果没有,说明代理没有发送
`X-Forwarded-Proto`,或者 `TRUST_PROXY` 没有覆盖它的地址。
