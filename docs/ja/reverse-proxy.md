# リバースプロキシ

NovelHub は平文 HTTP で待ち受けます。HTTPS で到達させるには、前段にリバースプロキ
シ — nginx、Caddy、Traefik または Cloudflare — を置き、そこで TLS を終端してリクエ
ストを転送します。

プロキシは実際のクライアントを NovelHub から隠してしまいます。その情報を伝え戻す
のが 2 つのヘッダーですが、どのクライアントでも偽装できるため、NovelHub はデフォル
トでそれらを無視します。誰にそれらを設定させるかを指定するのが `TRUST_PROXY` です。

作業は 2 つに分かれます。どちらも必須です。

| 要素 | 設定場所 |
| --- | --- |
| NovelHub がプロキシを信頼する | `.env` の `TRUST_PROXY` |
| プロキシがヘッダーを送信する | 使用するプロキシの設定 |

どちらか一方でも欠けると、同じ症状が現れます。全員が 1 つのレート制限バケットを共
有し、ログインクッキーに `Secure` が付かなくなります。

---

## ステップ 1 — プロキシを信頼する

`.env` に記述します。

```bash
TRUST_PROXY=true
```

`true` はループバック、プライベート、リンクローカルアドレス上のプロキシをカバーし
ます。セルフホスト環境のほぼすべてがこれに該当します。同一マシン上の nginx や
Caddy、あるいは同じ Docker ネットワーク上の別コンテナです。

プロキシが **パブリック** アドレスから NovelHub に到達する場合 — Cloudflare や、別
ホスト上のプロキシ — は、代わりにアドレスを列挙してください。

```bash
TRUST_PROXY=173.245.48.0/20,103.21.244.0/22,103.22.200.0/22
```

Cloudflare は現在のレンジを <https://www.cloudflare.com/ips/> で公開しています。
すべてをカンマ区切りで列挙してください。

変更後は NovelHub を再起動してください。

---

## ステップ 2 — ヘッダーを送信する

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

Caddy は 3 つのヘッダーすべてを送信し、証明書も自動的に取得します。他に必要なもの
はありません。

### Traefik（docker-compose のラベル）

```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.novelhub.rule=Host(`library.example.com`)"
  - "traefik.http.routers.novelhub.entrypoints=websecure"
  - "traefik.http.routers.novelhub.tls.certresolver=letsencrypt"
  - "traefik.http.services.novelhub.loadbalancer.server.port=3434"
```

Traefik はデフォルトで forwarded ヘッダーを送信します。

### Cloudflare

Cloudflare は `X-Forwarded-For` と `X-Forwarded-Proto` を独自に送信しますが、パブリッ
クアドレスから接続してくるため、`TRUST_PROXY=true` ではカバー **できません**。ステッ
プ 1 で示したように、公開されているレンジを列挙してください。

SSL/TLS モードは **Full** または **Full (strict)** に設定してください。Flexible モー
ドでは、Cloudflare はブラウザに対してサイトが HTTPS だと伝えながら、オリジンとは平
文 HTTP で通信します。これはクッキーを分かりにくい形で壊します。

---

## ステップ 3 — 検証

HTTPS でサインインし、**DevTools → Application → Cookies** を開いてください。
`access_token` の行に `Secure` が表示されている必要があります。

表示されていない場合は、逆順に確認していきます。

1. `TRUST_PROXY` は設定されていますか。そして NovelHub を再起動しましたか。
2. その値は、プロキシが接続してくるアドレスをカバーしていますか。受信リクエストの
   送信元 IP を NovelHub のログで確認してください — それがパブリックアドレスなら
   `true` では不十分で、明示的なリストが必要です。
3. プロキシは `X-Forwarded-Proto` を送信していますか。プロキシ側から確認します。

```bash
curl -sI https://library.example.com/api/v1/health
```

---

## プロキシからのみアクセスできるようにする

プロキシが同一ホスト上にある場合、それ以外のものが NovelHub に到達する必要はあり
ません。ループバックにバインドして、ポートを LAN に公開しないようにします。

ネイティブインストールの場合、`.env` に記述します。

```bash
SERVER_HOST=127.0.0.1
```

Docker の場合、`docker-compose.yml` に記述します。

```yaml
ports:
  - "127.0.0.1:3434:3434"
```

---

## 補足

**サブパスでの配信**（`example.com/novelhub`）はサポートされていません。フロント
エンドは絶対パスのアセットパスでビルドされています。代わりにサブドメインを使ってく
ださい。

**WebSocket** は使用していないため、アップグレード用のヘッダーは不要です。

**大きなアップロード** にはプロキシのボディサイズ上限を引き上げる必要があります。
nginx のデフォルトは 1 MB で、`client_max_body_size` を設定するまで書籍のアップロー
ドを `413` で拒否します。Caddy と Traefik にはそのようなデフォルトはありません。
