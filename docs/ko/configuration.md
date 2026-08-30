# 설정

NovelHub은 설정을 두 갈래로 나눕니다. 환경 변수는 데이터베이스가 열리기 전에
알아야 하는 값, 또는 데이터베이스를 보호하기 때문에 그 안에 둘 수 없는 값을
담당합니다. 나머지는 모두 관리자 UI에 있으며 재시작 없이 적용됩니다.

중요한 변수는 네 개입니다. 나머지는 모두 그대로 동작하는 기본값이 있고, 대부분은
머신 사양에 맞춰 자동으로 조정됩니다.

---

## 필수

`.env.example`을 `.env`로 복사한 뒤 시크릿 세 개를 채웁니다. 이 값이 없으면 서버는
토큰을 발급하지 않습니다.

```bash
cp .env.example .env
openssl rand -hex 32   # run three times, one value each
```

| 변수 | 용도 |
| --- | --- |
| `JWT_SECRET` | 액세스 토큰 서명 |
| `JWT_REFRESH_SECRET` | 리프레시 토큰 서명 |
| `DB_ENCRYPTION_KEY` | 데이터베이스에 저장되는 서드파티 토큰(AniList, MAL) 및 SMTP 비밀번호 암호화 |

각각 서로 다른 무작위 값을 사용합니다.

**나중에 변경할 때.** 두 JWT 시크릿 중 하나라도 변경하면 모든 사용자가
로그아웃됩니다. 데이터가 사라지지는 않습니다. `DB_ENCRYPTION_KEY`를 변경하면 이미
암호화된 트래커 토큰을 영구히 읽을 수 없게 되며, 사용자가 해당 계정을 다시 연결해야
합니다. 이 값을 건드리기 전에 데이터베이스를 백업하십시오.

이 값들이 환경 변수로 남아 있는 이유는 데이터베이스 *안에* 있는 데이터를 서명하고
암호화하기 때문입니다. 데이터베이스에 저장한다면 순환 구조가 되어 버립니다.

---

## 리버스 프록시

```bash
TRUST_PROXY=false
```

프록시가 보내는 두 헤더를 NovelHub이 신뢰할지 여부를 결정합니다.

- `X-Forwarded-For` — 실제 클라이언트가 누구인지, 요청 수 제한에 사용됩니다
- `X-Forwarded-Proto` — 원래 요청이 HTTPS였는지 여부로, 로그인 쿠키에 `Secure`
  플래그를 붙일지 결정합니다

| 값 | 사용 시점 | 신뢰 대상 |
| --- | --- | --- |
| `false` | 브라우저가 NovelHub에 직접 연결하는 경우 | 없음 |
| `true` | nginx/Caddy가 같은 호스트에 있거나, 같은 Docker 네트워크의 다른 컨테이너인 경우 | 루프백, 사설 및 링크 로컬 주소: `127.0.0.0/8`, `::1`, `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `fc00::/7`, `169.254.0.0/16` |
| `1.2.3.0/24,5.6.7.8` | 프록시가 공용 주소에 있는 경우 — 대개 Cloudflare | 나열된 IP와 CIDR만 |

앞단에 실제 프록시가 없다면 `true`로 설정하지 마십시오. 그러면 어떤 클라이언트든
해당 헤더를 직접 보낼 수 있습니다. 요청마다 새 요청 수 제한 버킷을 받게 되어
로그인 제한이 완전히 무력화되고, 위조된 `https` 표시 때문에 평문 HTTP로 전달되는
쿠키에 `Secure`가 붙어 브라우저가 이를 조용히 버립니다.

이 값을 설정하는 것은 절반에 불과합니다. 프록시가 실제로 헤더를 보내야 합니다.
프록시별 설정 방법과 검증 방법은 [리버스 프록시](reverse-proxy.md)를 참고하십시오.

`TRUST_PROXY`는 시작 시 한 번만 읽히며 관리자 설정으로 둘 수 없습니다. 로그인
쿠키에 `Secure`를 붙일지 결정하는 값이므로, 데이터베이스에 잘못된 값이 들어 있으면
로그인을 막는 그 문제를 고치기 위해 로그인을 해야 하는 상황이 됩니다.

---

## 선택

아래 항목은 모두 `.env.example`에서 주석 처리되어 있습니다. 재정의가 필요한 것만
주석을 해제하십시오.

### 네트워크

| 변수 | 기본값 | 참고 |
| --- | --- | --- |
| `SERVER_HOST` | `127.0.0.1` | Docker에서는 `0.0.0.0`으로 설정됩니다. 그곳에서는 재정의하지 마십시오. 그러지 않으면 발행된 포트에 접근할 수 없습니다 |
| `SERVER_PORT` | `3434` | |

### 저장소

| 변수 | 기본값 | 참고 |
| --- | --- | --- |
| `DATA_DIR` | `./data` | 아래 모든 항목의 루트 |
| `SQLITE_DB_PATH` | `$DATA_DIR/novelhub.db` | |
| `CALIBRE_IMPORT_DIR` | `$DATA_DIR/calibre` | 이 루트 아래의 디렉터리만 가져올 수 있습니다. Calibre 라이브러리가 다른 곳에 있으면 그 경로를 지정하십시오. |

`DATA_DIR`에는 다음이 들어 있습니다.

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

`DATA_DIR`만 백업하면 설치 전체를 백업한 것입니다.

### 성능

모두 자동으로 조정됩니다. 자원 사용량을 의도적으로 제한할 때만 설정하십시오.

| 변수 | 기본값 |
| --- | --- |
| `SQLITE_CACHE_SIZE_KB` | 시스템 메모리에 따라 결정(64 MB–512 MB) |
| `SQLITE_MMAP_SIZE_BYTES` | 시스템 메모리에 따라 결정(256 MB–2 GB) |
| `SQLITE_MAX_OPEN_CONNS` | CPU 개수 × 2, 4–16으로 제한 |
| `SQLITE_MAX_IDLE_CONNS` | 최대 연결 수와 동일 |
| `CACHE_MAX_COST_BYTES` | 시스템 메모리에 따라 결정 |
| `ASSET_CACHE_MAX_COST_BYTES` | 시스템 메모리 ÷ 32, 32 MB–512 MB로 제한 — 만화 페이지와 표지. 별도 예산에서 원시 바이트로 보관하므로 도서 레코드를 밀어내지 않습니다 |
| `JOB_WORKERS` | `1` — 백그라운드 작업 동시 실행 수 |
| `GOGC` | `200` — Go GC 목표치. 낮추면 메모리를 아끼고 CPU를 더 씁니다 |
| `FIBER_CONCURRENCY` | Fiber 기본값 |
| `FIBER_READ_BUFFER_SIZE` | Fiber 기본값 |
| `FIBER_WRITE_BUFFER_SIZE` | Fiber 기본값 |

### 로깅

| 변수 | 기본값 | 참고 |
| --- | --- | --- |
| `LOG_MAX_SIZE_MB` | `10` | 활성 로그가 순환되는 크기 |
| `LOG_MAX_FILES` | `5` | 보관할 순환 파일 수 |
| `DISABLE_REQUEST_LOG` | `true` | 처리량을 위해 요청별 로깅을 끕니다 |
| `DISABLE_STARTUP_MESSAGE` | `false` | |

이 값들이 환경 변수로 남아 있는 이유는 로깅이 데이터베이스가 열리기 전에 시작되기
때문입니다. 데이터베이스 실패도 기록할 수 있어야 합니다.

### 동작

| 변수 | 기본값 | 참고 |
| --- | --- | --- |
| `TOKEN_VERSION_CACHE` | `true` | 여러 인스턴스가 하나의 데이터베이스를 공유하는 경우 `false`로 설정하십시오. 그런 환경에서는 인메모리 캐시가 오래된 값을 들고 있게 됩니다 |
| `DISABLE_RESPONSE_COMPRESSION` | `false` | |
| `ENABLE_PREFORK` | `false` | 다중 프로세스 워커. 토큰 버전 캐시를 비활성화합니다 |
| `RESTORE_AUTO_RESTART` | `false` | 데이터베이스 복원을 준비한 뒤 종료하여 Docker나 systemd가 재시작하면서 적용하도록 합니다. Docker에서는 `true`로 설정됩니다 |

---

## 관리자 UI에서 설정하는 항목

환경 변수가 아닙니다. 첫 실행 시 설정 마법사에서 지정하고, 이후에는 **관리자 →
설정**에서 변경합니다. 변경은 즉시 적용됩니다.

| 영역 | 범위 |
| --- | --- |
| 사이트 | 제목, 설명, 로고, 파비콘, 사이드바 항목, 홈 섹션 |
| 서버 URL | OPDS 카탈로그와 Kobo 동기화 링크에 사용되는 절대 기본 URL. 비워 두면 요청마다 자동 감지합니다. 프록시 뒤에 있을 때만 설정하십시오 |
| 접근 | 회원가입 허용 여부, 로그인 필수 여부, 게스트 접근 모드, 라이브러리별 게스트 노출 |
| 권한 | 39개 권한 전체를 역할별로 제어 — 읽기, 개인 기능, 라이브러리 콘텐츠, 연동, 관리 |
| 이메일 (SMTP) | 호스트, 포트, 사용자명, 비밀번호, 발신 주소, TLS 모드, 최대 첨부 용량 (MB, 기본값 50MB), 사설 네트워크 연결 허용, 연결 테스트. 이메일 인증과 비밀번호 재설정 활성화도 여기입니다 |
| 리더 기능 | 본문 심층 검색, 사용자 폰트 업로드, 표지에 표시할 참여 지표, 사용자 정의 CSS |
| OAuth / SSO | Google, GitHub, Discord 및 사용자 지정 OIDC (OpenID Connect) 클라이언트 인증 정보, 리디렉션, 발급자 URL 설정 |
| 트래커 | AniList, MyAnimeList, Hardcover.app 독서 진도 동기화 및 인증 |
| 업로드 제한 | 청크 크기, 청크 개수, 동시 세션 수, 총 용량, 세션 TTL, 표지 및 사이트 에셋 크기 |
| 요청 수 제한 | 창(window)당 로그인 및 OPDS 시도 횟수와 창 길이 |

### 요청 수 제한

NovelHub은 정확히 두 가지에만 요청 수 제한을 걸며, 둘 다 같은 설정 한 쌍으로
관리됩니다. **로그인**(`/api/v1/auth/*`)과 **OPDS**(`/api/opds/*`)입니다.

두 경로 모두 bcrypt 비밀번호 검증을 수행하는데, 시도당 CPU를 약 50–100 ms
소모합니다. 요청의 다른 모든 처리를 합친 것의 약 600배입니다. 이것이 보호할 가치가
있는 자원입니다. OPDS가 포함된 이유는 HTTP Basic 인증을 사용해 세션이 없고, 따라서
*모든* 요청마다 bcrypt가 실행되기 때문입니다.

기본값은 60초당 5회 시도이며, 클라이언트 IP를 기준으로 집계합니다.

OPDS에서는 실패한 시도만 집계됩니다. 유효한 자격 증명으로 카탈로그를 주기적으로
조회하는 리더 앱은 정상 트래픽이므로 결코 제한되지 않습니다.

일반 API 요청 수 제한은 의도적으로 두지 않았습니다. 만화 챕터는 페이지마다 이미지
요청 하나로 렌더링되므로 200페이지 분량을 열면 정상적으로 200개의 요청이 발생합니다.
일반 제한을 걸면 공격자가 아니라 독자를 막게 됩니다.

---

### OPDS 1.2 & 2.0 서버

NovelHub에는 완전한 OPDS 1.2 (Atom XML) 및 OPDS 2.0 (JSON) 카탈로그 서버가 내장되어 있습니다:

- **OPDS 1.2 카탈로그**: `/api/opds/v1` (Atom XML 형식. KOReader, Moon+ Reader, Calibre, PocketBook, Aldiko 지원). 탐색 피드(`/recent`, `/authors`, `/series`, `/tags`), OpenSearch XML(`/api/opds/v1/opensearch.xml`), 전문 검색(`/api/opds/v1/search?q={searchTerms}`)을 포함합니다.
- **OPDS 2.0 카탈로그**: `/api/opds/v2/catalog` (JSON 형식 `application/opds+json`. Thorium과 같은 최신 리더 지원). 루트 탐색 링크, 간행물 메타데이터, 커버 이미지 링크, 다운로드 링크를 포함합니다.
- **인증**: HTTP Basic 인증(사용자 계정 이메일 및 비밀번호 사용) 및 Admin Settings에서 라이브러리별로 설정된 게스트 접근 정책을 지원합니다.

### PWA & 오프라인 독서

NovelHub는 네이티브 설치를 지원하는 완전한 Progressive Web App (PWA)입니다:

- **오프라인 엔진**: 도서 전체, 챕터 및 임베디드 이미지를 브라우저의 IndexedDB 스토리지에 직접 저장하여 네트워크 연결 없이 100% 오프라인으로 읽을 수 있습니다.
- **Service Worker & 업데이트**: `vite-plugin-pwa` 및 `workbox`로 작동하며 자동 업데이트 알림 배너와 스토리지 용량 모니터링 기능을 제공합니다.
- **권한**: 오프라인 도서 저장은 `book.offline` 권한을 통해 역할별로 제어됩니다.

### OAuth / SSO (단일 로그인)

**관리자 → 설정 → OAuth**에서 외부 서비스 인증을 설정할 수 있습니다.

- **지원 공급자**: Google, GitHub, Discord, OIDC (OpenID Connect).
- **설정**: 클라이언트 ID, 클라이언트 보안 비밀번호, 리디렉션 URI(`/api/v1/auth/oauth/:provider/callback`과 일치해야 함), 발급자 URL(OIDC용)을 입력합니다.
- **동작**: 외부 인증 공급자를 통해 로그인한 사용자는 회원가입 정책이 켜져 있는 경우 계정이 자동으로 생성되며, 동일한 이메일로 이미 가입된 계정이 있으면 해당 계정에 연동됩니다.

### 팟캐스트

**관리자 → 설정 → Podcasts**(또는 팟캐스트 관리 페이지)에서 RSS 피드를 구독하고 관리할 수 있습니다.

- 절대 경로의 RSS 피드 URL을 통해 구독합니다.
- 빌드되지 않은 Jekyll 템플릿 피드를 자동으로 감지하고 차단하여 등록 오류를 미연에 방지합니다.
- 최대 250MB 크기의 대용량 RSS 피드 구독을 기본적으로 지원합니다.
- 자동 갱신 및 신규 에피소드 다운로드 작업을 주기적으로 실행하거나 수동으로 시작할 수 있습니다.

### 자녀 보호 모드 & 연령 제한 (Kids Mode)

콘텐츠 등급 정책에 따라 성인용 콘텐츠 노출을 제어합니다.

- **콘텐츠 등급**: G, PG, R, R18.
- **자녀 보호 모드**: **사용자 프로필 → 자녀 보호 모드**에서 PIN 코드를 설정하여 활성화합니다. 활성화된 상태에서는 허용 등급을 초과하는 모든 책이 서재와 검색 결과에서 자동으로 숨겨집니다. 이를 해제하려면 설정한 6자리 숫자 PIN 코드가 필요합니다.

### VBook 연동

Android의 VBook 리더 앱을 통해 라이브러리 내 책들을 검색하고 읽을 수 있습니다.

- **설정**: 프로필 페이지에서 VBook용 플러그인 JSON URL을 복사하거나 `plugin.zip`을 다운로드합니다.
- **레지스트리**: `/api/v1/vbook/plugin.json` 경로로 플러그인 메타데이터를 제공하며, `/api/v1/vbook/plugin.zip` 경로로 VBook 플러그인을 제공합니다.

#### 독서 목록 & `.cbl` 가져오기

컬렉션은 "이 책이 어느 그룹에 속하는가"에 답합니다. 독서 목록은 "다음에 어떤 책을
읽어야 하는가"에 답합니다: 각 항목이 명시적인 위치를 가지므로, 순서는 파일이
등록된 순서가 아니라 사용자가 정한 순서입니다.

- **사용자별**: 독서 목록은 컬렉션과 마찬가지로 이를 만든 계정에만 속하며, 동일한 `book.collection` 권한으로 보호됩니다.
- **순서 변경**: `/read-lists`에서 항목을 드래그하거나 위/아래 버튼을 사용합니다. 전체 순서가 한 번의 요청으로 저장됩니다.
- **순서대로 읽기**: 첫 항목을 `?readlist=<id>`와 함께 엽니다. 마지막 챕터가 끝나면 리더의 기존 다음 버튼이 멈추지 않고 목록의 다음 책으로 이어집니다. 보관된(archived) 책은 건너뜁니다. 읽던 위치는 기억되지 않으며 — "순서대로 읽기"는 항상 첫 항목에서 시작합니다.
- **`.cbl` 가져오기**: ComicRack 독서 목록(최대 8 MB)을 업로드합니다. 문서 순서가 *그대로* 독서 순서이며 다시 정렬되지 않습니다. 항목은 시리즈 이름(대소문자 구분 없음)과 권 번호로 매칭되며, `01`, `1`과 `1.0`은 같은 번호로 취급됩니다. books 테이블에 연도 컬럼이 없으므로 `Year`와 `Volume`은 무시됩니다. 라이브러리에 없는 항목은 시리즈와 번호와 함께 가져오기 보고서로 반환됩니다. 시리즈와 번호가 같은 책이 둘 이상이면 먼저 발견된 책이 선택됩니다.
- **엔드포인트**(모두 `/api/v1/read-lists` 하위): `GET /`, `POST /`, `POST /import`, `GET|PUT|DELETE /:id`, `GET|POST /:id/books`, `DELETE /:id/books/:bookId`, `PUT /:id/order`, `GET /:id/next`.

### Book Doctor (EPUB 복구 엔진)

NovelHub에는 손상되었거나 표준 규격을 벗어난 EPUB 아카이브를 자동으로 진단하고 복구하는 도구가 포함되어 있습니다:

- **검증 및 오류 진단**: ZIP 헤더, `mimetype` 위치, `container.xml` 구조, `content.opf` 구문, 중복 매니페스트 ID/href, 미등록 파일, 누락된 spine 참조, 깨진 내부 XHTML 링크, 이스케이프되지 않은 XML 엔티티(`&nbsp;` 등), 누락된 NCX/Nav 목차를 정밀하게 검사합니다.
- **자동 복구 파이프라인**:
  - 압축되지 않은 `mimetype`을 ZIP의 첫 번째 오프셋에 재구축.
  - 중복 매니페스트 항목 정리 및 삭제된 파일에 대한 고아 참조 제거.
  - XML 네임스페이스 수정 및 손상된 내부 링크 자동 정리.
  - 표준 목차(`toc.ncx` / `nav.xhtml`) 자동 생성 및 구형 EPUB 2.0 형식을 최신 EPUB 3.0 표준으로 업그레이드.
- **해시 동기화 및 캐시 무효화**: 복구 완료 후 파일의 SHA-256 해시를 재계산하여 DB를 갱신하고, 연관된 RAM 캐시(`book:*`, `book_file:*`, `chapter:*`)를 즉시 삭제합니다.
- **일괄 복구 작업**: 관리자는 **관리자 → 운영 → 유지보수**에서 백그라운드 정기 작업(`repair_books`)을 통해 전체 라이브러리를 일괄 복구할 수 있습니다.

### 내장 WebDAV 서버

NovelHub 라이브러리를 컴퓨터의 네트워크 드라이브로 직접 연결하여 탐색할 수 있는 WebDAV 서버를 제공합니다:

- **엔드포인트**: `http(s)://<호스트>/webdav`
- **지원 클라이언트**: macOS Finder("서버에 연결"), Windows 파일 탐색기("네트워크 드라이브 연결"), Linux(Nautilus, Dolphin, `davfs2`), 모바일 및 eReader 리더 앱(KOReader, Moon+ Reader).
- **인증**: NovelHub 사용자 계정의 이메일 및 비밀번호를 사용하는 HTTP Basic 인증.
- **권한 관리**: `system.webdav` 권한으로 보호되며, 사용자 역할에 따라 제한된 라이브러리는 WebDAV 폴더 트리에서 자동으로 제외됩니다.

### Anki 연동 (AnkiConnect)

독서 중 하이라이트, 단어, 인용구를 Anki 암기 카드 덱으로 직접 내보낼 수 있습니다:

- **AnkiConnect 브릿지**: AnkiConnect 플러그인이 설치된 Anki(기본: `http://127.0.0.1:8765`)와 원활하게 연동됩니다.
- **맞춤형 매핑**: **프로필 → 연동 설정 → Anki**에서 대상 덱, 카드 템플릿/모델(Basic, Cloze 등), 필드 매핑(앞면, 뒷면, 출처, 책 제목, 저자), 태그를 자유롭게 설정할 수 있습니다.

---

## 인증 쿠키

설정할 것이 없습니다. 두 속성 모두 요청마다 유도됩니다.

**`Secure`**는 요청이 HTTPS로 도착했을 때 설정됩니다. 직접 연결이든, 신뢰하는
프록시의 `X-Forwarded-Proto`를 통한 것이든 마찬가지입니다(위의 `TRUST_PROXY` 참고).
평문 HTTP에서는 생략되는데, 브라우저가 안전하지 않은 연결에서 `Secure` 쿠키를
조용히 버리기 때문이며 그 증상은 비밀번호가 틀린 것처럼 나타납니다.

**`Domain`**은 설정하지 않으므로, 쿠키는 이를 내려준 호스트에만 적용됩니다.

**`SameSite`**는 `Lax`이며 그대로 두어야 합니다. `None`으로 완화하면 공격자의
페이지가 당신의 쿠키를 함께 보낼 수 있게 됩니다.

**`csrf_token`**은 세 번째 쿠키로, 의도적으로 JavaScript가 읽을 수 있게 되어
있습니다. 프론트엔드가 이 값을 `X-CSRF-Token` 헤더에 복사하고, 서버는 모든
POST/PUT/PATCH/DELETE에서 둘을 비교합니다. `Authorization` 헤더를 가진 요청과
`/kobo/`, `/komga/`, `/api/opds/`, `/api/v1/sync/` 접두사는 예외입니다. 이들은
요청마다 인증하며 쿠키를 보내지 않으므로 위조할 대상이 없습니다.

---

## 검증

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

프록시 뒤에서는 HTTPS로 로그인한 뒤 **DevTools → Application → Cookies**를
확인하십시오. `access_token` 행에 `Secure`가 표시되어야 합니다. 그렇지 않다면
프록시가 `X-Forwarded-Proto`를 보내지 않거나, `TRUST_PROXY`가 프록시의 주소를
포함하지 않는 것입니다.
