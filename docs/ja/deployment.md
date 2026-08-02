# デプロイ

NovelHub を動かす方法は 2 つあります。Docker か、ネイティブバイナリです。どちらも
[設定](configuration.md) に記載された同じ 3 つのシークレットを必要とします。

フロントエンドはバイナリにコンパイルされて組み込まれるため、プロセスは 1 つ、ポー
トは 1 つ、バックアップするディレクトリも 1 つです。Web サーバーも、フロントエンド
用の別ホストも不要です。

---

## Docker

```bash
cp .env.example .env
openssl rand -hex 32   # three times, one per secret
$EDITOR .env
docker compose up -d
```

`http://<host>:3434` を開いてください。初回起動時にセットアップウィザードが実行さ
れ、ルート管理者が作成されます。

compose ファイルはコンテナ向けに `SERVER_HOST`、`SERVER_PORT`、`DATA_DIR` を設定し
ます。これらはそのままにしておいてください — 特に `SERVER_HOST` は `0.0.0.0` のまま
でなければ、公開ポートにホストから到達できません。

### リバースプロキシの背後で

`.env` に追加します。

```bash
TRUST_PROXY=true
```

その上で、`X-Forwarded-For` と `X-Forwarded-Proto` を転送するようプロキシを設定して
ください — [リバースプロキシ](reverse-proxy.md) を参照してください。

`TRUST_PROXY` は、多くの Docker デプロイにプロキシがあるにもかかわらず、Docker でも
デフォルトでは有効になっていません。公開ポート経由のリクエストは Docker ブリッジ
（`172.17.0.1`）から到達しますが、これは *プライベート* アドレスです — つまり、素の
`docker compose up` では `true` にすると直接アクセスしてくるすべての訪問者を等しく
信頼してしまいます。そうなると訪問者は自分で `X-Forwarded-For` を設定してリクエスト
ごとに新しいレート制限バケットを得られるようになり、さらに
`X-Forwarded-Proto: https` を偽装して、平文 HTTP なのにログインクッキーに `Secure`
を付けさせることができます。そしてブラウザはそのクッキーを黙って破棄します。

プロキシが同一ホスト上で動いている場合は、ループバックに公開して他の何もコンテナに
到達できないようにしてください。

```yaml
ports:
  - "127.0.0.1:3434:3434"
```

### データ

すべては `/data` にマウントされる `novelhub_data` ボリュームに置かれます。

```
/data
├── novelhub.db      SQLite database
├── books/           imported books and covers
├── inbox/           drop files here for automatic import
├── uploads/         in-progress chunked uploads
├── logs/            rotating application logs
└── backups/         database backups
```

名前付きボリュームではなくホストのディレクトリを使う場合は、次のようにします。

```yaml
volumes:
  - /srv/novelhub:/data
```

コンテナは root として動作し、ディレクトリの中身は自分で作成します。

### 更新

```bash
docker compose pull
docker compose up -d
```

スキーマのマイグレーションは起動時に自動的に適用されます。先にバックアップを取って
ください — 下記を参照。

### ログ

```bash
docker compose logs -f
```

`/data/logs/novelhub.log` にも書き出され、10 MB でローテーションし、5 ファイルを保
持します。

---

## ネイティブ

Go 1.26 以降と [Bun](https://bun.sh) が必要です。

```bash
git clone https://github.com/AzenKain/Novel-Hub.git
cd Novel-Hub
cp .env.example .env
openssl rand -hex 32   # three times
$EDITOR .env

make run
```

`make run` はフロントエンドをビルドしてサーバーを起動します。単体のバイナリを作る場
合は次のようにします。

```bash
make build
./novelhub
```

バイナリは同じ場所に `db/schema/` を必要とします — 起動時にスキーマファイルを適用す
るためです。

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

`Restart=always` は `RESTORE_AUTO_RESTART=true` も有効に機能させ、データベースのリス
トアを手作業なしで完了できるようにします。

---

## バックアップ

**管理 → 運用 → バックアップ** では、サーバーの稼働中に一貫性のある SQLite スナップ
ショットを作成できます。データベースのみ、またはデータベースと書籍ファイルの両方を
選べます。スケジュール設定は **運用 → スケジュール** から行います。

リストアはまずステージングと検証が行われ、次回起動時に適用されます。
`RESTORE_AUTO_RESTART=true`（Docker のデフォルト）の場合、NovelHub は終了し、スーパー
バイザが自動的に再起動します。それ以外の場合は、管理 UI がリストアの準備完了を示し
たら自分で再起動してください。

外部でバックアップを取る場合は、サーバーを停止して `DATA_DIR` をコピーしてくださ
い。サーバー稼働中に `novelhub.db` をコピーすると書き込みが途切れた状態を取り込んで
しまう可能性があります — これを正しく処理する管理 UI のバックアップを使ってくださ
い。

---

## 書籍のインポート

3 つの経路があります。

| 経路 | 方法 |
|---|---|
| アップロード | **管理 → 書籍 → アップロード**。チャンク分割されているため、接続が不安定でも大きなファイルを扱えます |
| Inbox | `data/inbox/<libraryID>/` にファイルを置き、**運用 → ジョブ → Inbox をスキャン** を実行します。入れ子のフォルダは 5 階層までスキャンされ、インポート済みのファイルは削除され、空のディレクトリは片付けられます |
| Calibre | **管理 → ライブラリ → Calibre からインポート**。`metadata.db` を含むフォルダを指定します |

Inbox のスキャンは、ファイルの変更が止まってから 10 秒待ってからインポートするた
め、コピー途中のファイルが取り込まれることはありません。

---

## 電子書籍リーダークライアント

| プロトコル | エンドポイント | 認証 |
|---|---|---|
| OPDS 1.2 | `/opds/v1` | HTTP Basic — NovelHub のメールアドレスとパスワード |
| OPDS 2.0 | `/opds/v2/catalog` | HTTP Basic |
| Kobo | `/kobo/v1` | Bearer トークン |

KOReader、Calibre、Moon+ Reader、Thorium などの OPDS クライアントで動作します。

OPDS のレート制限は認証の *失敗* のみを対象とするため、通常のポーリングがスロットリ
ングされることはありません。カタログのリンクが誤ったホストを指している場合 — たとえ
ばパスを書き換えるプロキシの背後にある場合 — は、`SERVER_URL` に正しい絶対ベース URL
を設定してください。

---

## トラブルシューティング

**Docker でサーバーに到達できない。** コンテナ内では `SERVER_HOST` が `0.0.0.0` で
なければなりません。`.env` で `127.0.0.1` を設定している場合は削除してください。
compose ファイルが既に正しい値を設定しています。

**HTTPS なのにログインクッキーに `Secure` フラグが付かない。** `TRUST_PROXY` が未設
定か、プロキシのアドレスをカバーしていないか、プロキシが `X-Forwarded-Proto` を送っ
ていません。[リバースプロキシ](reverse-proxy.md#ステップ-3--検証) を参照してくださ
い。

**全員が 1 つのレート制限バケットを共有してしまう。** 原因は同じです。
`TRUST_PROXY` がないと、すべてのリクエストがプロキシから来たように見えます。

**アップロードで `413` が出る。** NovelHub ではなくプロキシのボディサイズ上限です。
nginx のデフォルトは 1 MB なので、`client_max_body_size 0` を設定してください。

**再起動後にデータベースが消えている。** `SQLITE_DB_PATH` または `DATA_DIR` がマウン
トされたボリュームの外を指していました。Docker ではどちらも compose ファイルが正し
く設定しています — `.env` がそれらを上書きしていないか確認してください。

**パスワードを 1 回間違えただけでロックアウトされる。** 現行バージョンでは修正され
ています。古いビルドでは、試行失敗後に空のパスワードハッシュをキャッシュしてしまい、
キャッシュが期限切れになるまで正しいパスワードも拒否していました。更新してくださ
い。
