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
| --- | --- |
| `JWT_SECRET` | 为访问令牌签名 |
| `JWT_REFRESH_SECRET` | 为刷新令牌签名 |
| `DB_ENCRYPTION_KEY` | 加密存放在数据库里的第三方令牌(AniList、MAL)和 SMTP 密码 |

每个都要用不同的随机值。

**之后再修改。** 修改任意一个 JWT 密钥会让所有人被登出 —— 不会丢失数据。修改
`DB_ENCRYPTION_KEY` 会让已加密的追踪器令牌和已保存的 SMTP 密码永久无法解读;用户必须
重新关联那些账号,管理员也必须重新输入 SMTP 密码。
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
| --- | --- | --- |
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
| --- | --- | --- |
| `SERVER_HOST` | `127.0.0.1` | Docker 会设为 `0.0.0.0`;不要在那里覆盖它,否则映射出去的端口无法访问 |
| `SERVER_PORT` | `3434` | |

### 存储

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `DATA_DIR` | `./data` | 下面所有内容的根目录 |
| `SQLITE_DB_PATH` | `$DATA_DIR/novelhub.db` | |
| `CALIBRE_IMPORT_DIR` | `$DATA_DIR/calibre` | 只有位于该根目录下的目录才能导入。如果你的 Calibre 库在别处，请把它指向那里。 |

`DATA_DIR` 包含:

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

备份了 `DATA_DIR`,就等于备份了整套安装。

### 性能

这些全部会自动调优。只有当你想刻意限制资源占用时才去设置。

| 变量 | 默认值 |
| --- | --- |
| `SQLITE_CACHE_SIZE_KB` | 按系统内存计算(64 MB–512 MB) |
| `SQLITE_MMAP_SIZE_BYTES` | 按系统内存计算(256 MB–2 GB) |
| `SQLITE_MAX_OPEN_CONNS` | CPU 核数 × 2,限制在 4–16 之间 |
| `SQLITE_MAX_IDLE_CONNS` | 与最大打开连接数相同 |
| `CACHE_MAX_COST_BYTES` | 按系统内存计算 |
| `ASSET_CACHE_MAX_COST_BYTES` | 系统内存 ÷ 32,限制在 32 MB–512 MB —— 漫画页与封面,以原始字节存放在独立预算中,因此不会挤掉书目记录 |
| `JOB_WORKERS` | `1` —— 后台任务并发数 |
| `GOGC` | `200` —— Go GC 目标值;调低是用 CPU 换内存 |
| `FIBER_CONCURRENCY` | Fiber 默认值 |
| `FIBER_READ_BUFFER_SIZE` | Fiber 默认值 |
| `FIBER_WRITE_BUFFER_SIZE` | Fiber 默认值 |

### 日志

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `LOG_MAX_SIZE_MB` | `10` | 当前日志文件达到该大小时轮转 |
| `LOG_MAX_FILES` | `5` | 保留的轮转文件数 |
| `DISABLE_REQUEST_LOG` | `true` | 关闭逐请求日志以提升吞吐 |
| `DISABLE_STARTUP_MESSAGE` | `false` | |

这些保留为环境变量,是因为日志在数据库打开之前就已启动 —— 数据库出问题时必须还能被
记录下来。

### 行为

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `TOKEN_VERSION_CACHE` | `true` | 多个实例共用一个数据库时设为 `false`,那种情况下内存缓存会变成过期的脏数据 |
| `DISABLE_RESPONSE_COMPRESSION` | `false` | |
| `ENABLE_PREFORK` | `false` | 多进程 worker。会禁用令牌版本缓存 |
| `RESTORE_AUTO_RESTART` | `false` | 暂存数据库恢复后退出进程,让 Docker 或 systemd 重启并应用它。Docker 中设为 `true` |

---

## 在管理后台中配置

这些不是环境变量。首次启动时通过安装向导设置,之后在 **管理后台 → 设置** 中修改。
改动立即生效。

| 区域 | 涵盖内容 |
| --- | --- |
| 站点 | 标题、描述、Logo、favicon、侧边栏项目、首页板块 |
| 服务器 URL | OPDS 目录和 Kobo 同步链接使用的绝对基础 URL。留空则按每个请求自动探测;仅当探测到的主机不正确时才需设置,例如位于代理之后 |
| 访问 | 注册开关、是否必须登录、访客访问模式、按书库的访客可见性 |
| 权限 | 按角色控制全部 39 项权限 —— 阅读、个人功能、书库内容、集成、管理 |
| 邮件 (SMTP) | 主机、端口、用户名、密码、发件地址、TLS 模式、最大附件大小 (MB，默认 50MB)、是否允许连接私有网络,以及连接测试。邮箱验证与密码重置的开关也在这里 |
| 阅读器功能 | 书内深度搜索、自定义字体上传、封面上显示哪些互动统计、自定义用户 CSS |
| OAuth / SSO | Google、GitHub、Discord 和自定义 OIDC (OpenID Connect) 客户端凭据配置、重定向和发行者 URL |
| 追踪器 | 启用/禁用与 AniList、MyAnimeList 和 Hardcover.app 的阅读进度同步与认证 |
| 上传限制 | 分片大小、分片数量、并发会话数、总大小、会话 TTL、封面与站点素材大小 |
| 限流 | 每个时间窗内的登录与 OPDS 尝试次数,以及时间窗长度 |

### 限流

NovelHub 只对两件事限流,而且两者共用同一对设置:**登录**(`/api/v1/auth/*`)和
**OPDS**(`/api/opds/*`)。

两者都会执行 bcrypt 密码校验,每次大约消耗 50–100 ms 的 CPU —— 差不多是同一个请求里
其他所有工作总和的 600 倍。这才是值得保护的资源。OPDS 也算在内,是因为它使用
HTTP Basic 认证,不携带会话,所以*每个*请求都要跑一次 bcrypt。

默认值:每 60 秒 5 次尝试,按客户端 IP 计数。

对 OPDS 只统计失败的尝试。阅读器应用带着正确凭据轮询目录属于正常流量,永远不会被
限流。

这里刻意没有设置通用的 API 限流。一个漫画章节每页都渲染为一个图片请求,所以打开一卷
200 页的书正常就会发出 200 个请求 —— 通用限流只会限住读者,而不是攻击者。

---

### OPDS 1.2 & 2.0 服务器

NovelHub 内置了完整的 OPDS 1.2 (Atom XML) 和 OPDS 2.0 (JSON) 书目目录服务器:

- **OPDS 1.2 目录**: `/api/opds/v1` (Atom XML 格式,完美兼容 KOReader、Moon+ Reader、Calibre、PocketBook、Aldiko)。包含导航 Feed (`/recent`、`/authors`、`/series`、`/tags`)、OpenSearch XML (`/api/opds/v1/opensearch.xml`) 及全文搜索 (`/api/opds/v1/search?q={searchTerms}`)。
- **OPDS 2.0 目录**: `/api/opds/v2/catalog` (JSON 格式 `application/opds+json`,兼容 Thorium 等现代阅读器)。包含根导航链接、作品元数据、封面图片链接及获取/下载链接。
- **身份验证**: 支持 HTTP Basic Auth(使用用户邮箱和密码)以及在管理设置中按书库配置的访客访问策略。

### PWA 与离线阅读

NovelHub 是一套完整的渐进式 Web 应用 (PWA),支持原生应用安装:

- **离线引擎**: 用户可将整本书籍、章节及嵌入图片直接保存至浏览器的 IndexedDB 存储中,在无网络连接下实现 100% 离线阅读。
- **Service Worker 与更新**: 由 `vite-plugin-pwa` 和 `workbox` 驱动,具备自动更新通知横幅及存储空间用量监控。
- **权限控制**: 离线保存书籍功能通过 `book.offline` 权限按角色精细化控制。

### OAuth / SSO (单点登录)

在 **管理 → 设置 → OAuth** 下配置第三方认证服务。

- **支持的服务商**: Google、GitHub、Discord、OIDC (OpenID Connect)。
- **设置**: 输入 Client ID、Client Secret、重定向 URI（必须与 `/api/v1/auth/oauth/:provider/callback` 一致）以及发行者 URL（用于 OIDC）。
- **行为**: 经由第三方登录的账户，如果系统允许注册，则会自动创建新账号，若邮箱已存在且已验证，则会自动映射到该账户。

### 播客 (Podcasts)

在 **管理 → 设置 → Podcasts**（或播客页面）下订阅和管理 RSS 播客源。

- 输入绝对 RSS 订阅源 URL 进行订阅。
- 系统具备 Jekyll/Liquid 模板检测机制，自动拦截未编译的模板源，避免订阅出错。
- 原生支持下载和解析高达 250MB 的巨型 RSS XML 播客源。
- 可以设置定时调度自动刷新，或手动触发后台任务下载最新单集音频文件为图书。

### 儿童模式 & 年龄分级 (Kids Mode)

根据内容分级限制未成年人接触的内容。

- **内容分级**: G、PG、R、R18。
- **儿童模式**: 在 **个人资料 → 儿童模式** 中设定 6 位数字 PIN 码启用。启用后，所有评级超出限制的书籍在书架和搜索结果中将被自动隐藏。解除该模式需输入正确的 PIN 码。

### VBook 联动

允许直接在 Android 端 VBook 阅读器应用上浏览和阅读你的 NovelHub 图书馆。

- **设置**: 在个人资料页面复制 VBook 专属的 plugin JSON 链接，或直接下载 `plugin.zip`。
- **注册点**: 系统通过 `/api/v1/vbook/plugin.json` 暴露插件注册元数据，并通过 `/api/v1/vbook/plugin.zip` 提供 VBook 插件包的下载。

### 阅读列表与 `.cbl` 导入

收藏集回答的是"这本书属于哪一组"。阅读列表回答的是"接下来该读哪一本": 每个条目都
带有明确的位置，因此顺序由你决定，而不是文件入库的顺序。

- **按用户隔离**: 阅读列表与收藏集一样,只属于创建它的账号,并由同一个 `book.collection` 权限守卫。
- **重新排序**: 在 `/read-lists` 拖动条目,或使用上/下按钮。整个顺序在一次请求中保存。
- **按顺序阅读**: 打开第一个条目并带上 `?readlist=<id>`。读到最后一章末尾时,阅读器原有的"下一个"按钮不会停下,而是接着跳到列表中的下一本书。已归档(archived)的书会被跳过。阅读位置**不会**被记住 — "按顺序阅读"始终从第一个条目开始。
- **`.cbl` 导入**: 上传 ComicRack 阅读列表(最大 8 MB)。文档顺序*就是*阅读顺序,不会重新排序。条目按系列名称(不区分大小写)加期号匹配,其中 `01`、`1` 和 `1.0` 视为同一个号。由于 books 表没有年份列,`Year` 与 `Volume` 会被忽略。库中不存在的条目会连同系列名与期号一起出现在导入报告里;若有两本书的系列与期号相同,则取先找到的那一本。
- **接口**(全部位于 `/api/v1/read-lists` 之下): `GET /`、`POST /`、`POST /import`、`GET|PUT|DELETE /:id`、`GET|POST /:id/books`、`DELETE /:id/books/:bookId`、`PUT /:id/order`、`GET /:id/next`。

### Book Doctor (EPUB 修复引擎)

NovelHub 内置了针对损坏或非标准 EPUB 电子书文件的自动化深度诊断与修复引擎:

- **结构校验与错误检测**: 全面检查 ZIP 头部、`mimetype` 偏移位置、`container.xml` 结构、`content.opf` 语法、清单重复 ID/href、未登记文件、脊骨(Spine)缺失、失效的内部 XHTML 锚点、未转义的 XML 实体(如 `&nbsp;`)以及 NCX/Nav 目录缺失。
- **自动修复管线**:
  - 重建未压缩的 `mimetype` 并置于 ZIP 归档首字节偏移。
  - 清理清单重复项及指向已删除文件的孤儿清单引用。
  - 自动修复 XML 命名空间并清理失效的内部链接。
  - 自动生成标准目录(`toc.ncx` / `nav.xhtml`),并将旧式 EPUB 2.0 规范平滑升级至现代 EPUB 3.0。
- **哈希同步与缓存刷新**: 修复完成后,系统自动重新计算文件的 SHA-256 哈希并同步至数据库,同时清理对应的 RAM 内存缓存(`book:*`, `book_file:*`, `chapter:*)。
- **后台批量修复作业**: 管理员可在 **管理 → 运维 → 维护** 中配置后台定时任务(`repair_books`),一键对全库 EPUB 文件进行批量修复。

### 原生 WebDAV 服务

NovelHub 提供原生 WebDAV 服务端点,支持将图书库直接挂载为本地网络驱动器:

- **服务地址**: `http(s)://<你的主机>/webdav`
- **支持客户端**: macOS Finder("连接服务器")、Windows 资源管理器("映射网络驱动器")、Linux (Nautilus/Dolphin/davfs2) 以及各类电子书阅读器应用(KOReader, Moon+ Reader)。
- **身份认证**: 使用你的 NovelHub 账号邮箱与密码进行 HTTP Basic 认证。
- **权限隔离**: 受 `system.webdav` 权限管控,当前角色无权访问的图书库将自动从 WebDAV 目录树中剔除。

### Anki 卡片联动 (AnkiConnect)

支持在阅读时将高亮笔记、生词和精彩摘录直接同步至 Anki 记忆卡片组:

- **AnkiConnect 桥接**: 无缝连接安装有 AnkiConnect 插件的 Anki 客户端(默认: `http://127.0.0.1:8765`)。
- **自定义映射**: 在 **个人资料 → 集成设置 → Anki** 中自由配置目标卡组(Deck)、卡片模板(Basic, Cloze 等)、字段映射(正面, 背面, 来源, 书名, 作者)与标签(Tags)。

---

## 认证 Cookie

无需配置;两个属性都按每个请求推导得出。

**`Secure`** 会在请求通过 HTTPS 到达时设置 —— 无论是直连,还是经由受信任代理的
`X-Forwarded-Proto`(见上文 `TRUST_PROXY`)。在纯 HTTP 上则会省略,因为浏览器会在
不安全连接上静默丢弃带 `Secure` 的 Cookie,而表现出来就像是密码错误。

**`Domain`** 从不设置,因此 Cookie 的作用域始终限定在下发它的那台主机上。

**`SameSite`** 为 `Lax`,并且应当保持这个值。放宽成 `None` 等于让攻击者的页面也能
带上你的 Cookie。

**`csrf_token`** 是第三个 Cookie,有意允许 JavaScript 读取。前端把它复制到
`X-CSRF-Token` 请求头,服务端在每个 POST/PUT/PATCH/DELETE 上比对两者。带
`Authorization` 头的请求,以及 `/kobo/`、`/komga/`、`/api/opds/`、`/api/v1/sync/`
这几个前缀不受此限制 —— 它们逐请求认证且不发送 Cookie,没有可伪造的对象。

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
