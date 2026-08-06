# 배포

NovelHub을 실행하는 방법은 두 가지입니다. Docker 또는 네이티브 바이너리입니다. 둘
다 [설정](configuration.md)에 나오는 동일한 시크릿 세 개가 필요합니다.

프론트엔드는 바이너리에 함께 컴파일되므로 프로세스 하나, 포트 하나, 백업할 디렉터리
하나면 끝입니다. 웹 서버도, 별도의 프론트엔드 호스트도 필요하지 않습니다.

---

## Docker

```bash
cp .env.example .env
openssl rand -hex 32   # three times, one per secret
$EDITOR .env
docker compose up -d
```

`http://<host>:3434`을 여십시오. 첫 실행 시 설정 마법사가 실행되어 루트 관리자를
생성합니다.

compose 파일이 컨테이너용으로 `SERVER_HOST`, `SERVER_PORT`, `DATA_DIR`을 설정합니다.
이 값들은 그대로 두십시오. 특히 `SERVER_HOST`는 `0.0.0.0`으로 유지되어야 하며,
그러지 않으면 호스트에서 발행된 포트에 접근할 수 없습니다.

### 리버스 프록시 뒤에서

추가할 것은 없습니다. compose 파일은 이미 `TRUST_PROXY=true`를 기본값으로 둡니다.
compose 배포는 거의 모두 프록시 뒤에 있기 때문입니다. 프록시가
`X-Forwarded-For`와 `X-Forwarded-Proto`를 전달하도록만 설정하십시오.
[리버스 프록시](reverse-proxy.md)를 참고하십시오.

**프록시 없이 포트를 인터넷에 그대로 노출한다면 `.env`에 `TRUST_PROXY=false`를
설정하십시오.** 발행된 포트를 통과한 요청은 Docker 브리지(`172.17.0.1`), 즉 *사설*
주소에서 도착하므로 `true`는 실제 프록시와 똑같이 직접 접속하는 모든 방문자를
신뢰합니다. 그러면 그 방문자들이 직접 `X-Forwarded-For`를 설정해 요청마다 새 요청 수
제한 버킷을 받을 수 있어 로그인 제한이 완전히 무력화되고, `X-Forwarded-Proto: https`를
위조해 평문 HTTP에서 로그인 쿠키에 `Secure`가 붙게 만들 수 있습니다. 브라우저는 그
쿠키를 조용히 버리므로 증상은 "비밀번호가 틀렸다"로 나타납니다.

프록시가 같은 호스트에서 실행된다면, 그 외의 어떤 것도 컨테이너에 접근하지 못하도록
루프백으로 발행하십시오.

```yaml
ports:
  - "127.0.0.1:3434:3434"
```

### 데이터

모든 데이터는 `/data`에 마운트되는 `novelhub_data` 볼륨에 저장됩니다.

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

명명된 볼륨 대신 호스트 디렉터리를 사용하려면:

```yaml
volumes:
  - /srv/novelhub:/data
```

컨테이너는 root로 실행되며 디렉터리 내용을 스스로 생성합니다.

### 업데이트

```bash
docker compose pull
docker compose up -d
```

새 스키마는 시작 시 적용됩니다. 먼저 백업하십시오. 아래를 참고하십시오.

### 헬스체크

이미지에는 헬스체크가 포함되어 있어 20초의 유예 후 30초마다 `/api/v1/health`를
확인합니다. 따라서 `docker compose ps`는 단순히 "running"이 아니라 컨테이너의 실제
상태를 보고합니다.

```bash
docker compose ps          # STATUS 열에 healthy / unhealthy 표시
curl http://127.0.0.1:3434/api/v1/health
```

### 로그

```bash
docker compose logs -f
```

`/data/logs/novelhub.log`에도 기록되며, 10 MB마다 순환하고 5개 파일을 보관합니다.

---

## 네이티브

Go 1.26 이상과 [Bun](https://bun.sh)이 필요합니다.

```bash
git clone https://github.com/AzenKain/Novel-Hub.git
cd Novel-Hub
cp .env.example .env
openssl rand -hex 32   # three times
$EDITOR .env

make run
```

`make run`은 프론트엔드를 빌드하고 서버를 시작합니다. 단독 실행 바이너리를 만들려면:

```bash
make build
./novelhub
```

바이너리는 자체 완결형입니다. 스키마와 웹 UI가 내장되어 있어 옆에 복사할 파일이 없습니다. 시작 시 데이터베이스를 만들고 스키마를 적용합니다.

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

`Restart=always`는 `RESTORE_AUTO_RESTART=true`도 함께 활용할 수 있게 해주며, 덕분에
데이터베이스 복원이 수동 개입 없이 완료됩니다.

---

## 백업

**관리자 → 운영 → 백업**에서 서버가 실행되는 중에도 일관성 있는 SQLite 스냅샷을
생성합니다. 데이터베이스만, 또는 데이터베이스와 책 파일을 함께 선택할 수 있습니다.
**운영 → 스케줄**에서 예약할 수 있습니다.

복원은 먼저 준비 및 검증을 거친 뒤 다음 시작 시 적용됩니다.
`RESTORE_AUTO_RESTART=true`(Docker 기본값)일 때는 NovelHub이 종료되어 감독
프로세스가 자동으로 재시작합니다. 그렇지 않은 경우에는 관리자 UI에 복원 준비가
완료되었다고 표시되면 직접 재시작하십시오.

외부에서 백업하려면 서버를 중지하고 `DATA_DIR`을 복사하십시오. 서버가 실행되는 중에
`novelhub.db`를 복사하면 쓰기가 중간에 끊긴 상태를 담을 수 있습니다. 이를 올바르게
처리하는 관리자 백업을 사용하십시오.

---

## 책 가져오기

세 가지 경로가 있습니다.

| 경로 | 방법 |
|---|---|
| 업로드 | **관리자 → 책 → 업로드**. 청크 방식이라 연결이 불안정해도 대용량 파일이 살아남습니다 |
| Inbox | `data/inbox/<libraryID>/`에 파일을 넣고 **운영 → 작업 → Inbox 스캔**을 실행합니다. 중첩 폴더는 5단계 깊이까지 스캔되며, 가져온 파일은 삭제되고 빈 디렉터리는 정리됩니다 |
| Calibre | **관리자 → 라이브러리 → Calibre에서 가져오기**. `metadata.db`가 있는 폴더를 지정합니다 |

Inbox 스캔은 파일 변경이 멈춘 뒤 10초를 기다린 다음 가져오므로, 복사가 완료되지
않은 파일이 잡히는 일은 없습니다.

---

## 전자책 리더 클라이언트

| 프로토콜 | 엔드포인트 | 인증 |
|---|---|---|
| OPDS 1.2 | `/api/opds/v1` | HTTP Basic — NovelHub 이메일과 비밀번호 |
| OPDS 2.0 | `/api/opds/v2/catalog` | HTTP Basic |
| Kobo | `/kobo/<token>/v1/…` | 경로에 담긴 토큰 — Kobo는 Authorization 헤더를 보내지 않습니다 |
| Mihon / Tachiyomi | `/komga/api/v1` | HTTP Basic 또는 `X-API-Key: <이메일>:<비밀번호>` |

KOReader, Calibre, Moon+ Reader, Thorium 및 기타 OPDS 클라이언트에서 동작합니다.

Mihon(구 Tachiyomi)에서는 기본 **Komga** 확장을 설치하고
`http://<host>:3434/komga`를 가리키게 하면 됩니다. 클라이언트 쪽은 수정하지
않습니다 — NovelHub가 그 확장이 이미 사용하는 Komga REST API에 응답하며,
CBZ/CBR 아카이브에서 페이지를 바로 제공합니다. 진행률은 Mihon 내장 Komga
트래커를 통해 양방향으로 동기화됩니다. `komga.sync` 권한이 필요합니다.

Kobo 엔드포인트는 직접 입력하지 않습니다. **프로필 → Kobo 동기화**를 열어 생성된 URL을
복사하십시오. 사용자별 비밀 토큰이 들어 있습니다. 비밀번호처럼 취급하십시오 — 그것을
가진 사람은 당신의 라이브러리에 접근할 수 있습니다.

OPDS는 인증 *실패*에만 요청 수 제한이 걸리므로 정상적인 주기 조회는 결코 제한되지
않습니다. 카탈로그 링크가 잘못된 호스트를 가리킨다면 — 예를 들어 경로를 재작성하는
프록시 뒤에 있을 때 — **관리 → 설정**의 **서버 URL**을 올바른 절대 기본 URL로
설정하십시오. 재시작 없이 즉시 적용됩니다.

---

## 문제 해결

**Docker에서 서버에 접근할 수 없습니다.** 컨테이너 안에서 `SERVER_HOST`는
`0.0.0.0`이어야 합니다. `.env`에서 `127.0.0.1`로 설정했다면 제거하십시오. compose
파일이 이미 올바른 값을 설정합니다.

**HTTPS인데 로그인 쿠키에 `Secure` 플래그가 없습니다.** `TRUST_PROXY`가 설정되지
않았거나, 프록시의 주소를 포함하지 않거나, 프록시가 `X-Forwarded-Proto`를 보내지
않는 것입니다. [리버스 프록시](reverse-proxy.md#3단계--검증)를 참고하십시오.

**모든 사용자가 하나의 요청 수 제한 버킷을 공유합니다.** 원인이 같습니다.
`TRUST_PROXY`가 없으면 모든 요청이 프록시에서 온 것처럼 보입니다.

**업로드 시 `413`이 발생합니다.** NovelHub이 아니라 프록시의 본문 크기 제한입니다.
nginx는 기본값이 1 MB이므로 `client_max_body_size 0`을 설정하십시오.

**재시작 후 데이터베이스가 없습니다.** `SQLITE_DB_PATH` 또는 `DATA_DIR`이 마운트된
볼륨 밖을 가리킨 것입니다. Docker에서는 compose 파일이 둘 다 올바르게 설정하므로,
`.env`가 이를 재정의하고 있는지 확인하십시오.

**비밀번호를 한 번 틀렸는데 잠겼습니다.** 현재 버전에서는 수정되었습니다. 이전
빌드는 시도가 실패한 뒤 빈 비밀번호 해시를 캐시해서, 캐시가 만료될 때까지 올바른
비밀번호도 거부했습니다. 업데이트하십시오.
