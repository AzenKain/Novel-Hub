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
|---|---|
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
|---|---|---|
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
|---|---|---|
| `SERVER_HOST` | `127.0.0.1` | Docker에서는 `0.0.0.0`으로 설정됩니다. 그곳에서는 재정의하지 마십시오. 그러지 않으면 발행된 포트에 접근할 수 없습니다 |
| `SERVER_PORT` | `3434` | |

### 저장소

| 변수 | 기본값 | 참고 |
|---|---|---|
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
|---|---|
| `SQLITE_CACHE_SIZE_KB` | 시스템 메모리에 따라 결정(64 MB–512 MB) |
| `SQLITE_MMAP_SIZE_BYTES` | 시스템 메모리에 따라 결정(256 MB–2 GB) |
| `SQLITE_MAX_OPEN_CONNS` | CPU 개수 × 2, 4–16으로 제한 |
| `SQLITE_MAX_IDLE_CONNS` | 최대 연결 수와 동일 |
| `CACHE_MAX_COST_BYTES` | 시스템 메모리에 따라 결정 |
| `JOB_WORKERS` | `1` — 백그라운드 작업 동시 실행 수 |
| `GOGC` | `200` — Go GC 목표치. 낮추면 메모리를 아끼고 CPU를 더 씁니다 |
| `FIBER_CONCURRENCY` | Fiber 기본값 |
| `FIBER_READ_BUFFER_SIZE` | Fiber 기본값 |
| `FIBER_WRITE_BUFFER_SIZE` | Fiber 기본값 |

### 로깅

| 변수 | 기본값 | 참고 |
|---|---|---|
| `LOG_MAX_SIZE_MB` | `10` | 활성 로그가 순환되는 크기 |
| `LOG_MAX_FILES` | `5` | 보관할 순환 파일 수 |
| `DISABLE_REQUEST_LOG` | `true` | 처리량을 위해 요청별 로깅을 끕니다 |
| `DISABLE_STARTUP_MESSAGE` | `false` | |

이 값들이 환경 변수로 남아 있는 이유는 로깅이 데이터베이스가 열리기 전에 시작되기
때문입니다. 데이터베이스 실패도 기록할 수 있어야 합니다.

### 동작

| 변수 | 기본값 | 참고 |
|---|---|---|
| `TOKEN_VERSION_CACHE` | `true` | 여러 인스턴스가 하나의 데이터베이스를 공유하는 경우 `false`로 설정하십시오. 그런 환경에서는 인메모리 캐시가 오래된 값을 들고 있게 됩니다 |
| `DISABLE_RESPONSE_COMPRESSION` | `false` | |
| `ENABLE_PREFORK` | `false` | 다중 프로세스 워커. 토큰 버전 캐시를 비활성화합니다 |
| `RESTORE_AUTO_RESTART` | `false` | 데이터베이스 복원을 준비한 뒤 종료하여 Docker나 systemd가 재시작하면서 적용하도록 합니다. Docker에서는 `true`로 설정됩니다 |

---

## 관리자 UI에서 설정하는 항목

환경 변수가 아닙니다. 첫 실행 시 설정 마법사에서 지정하고, 이후에는 **관리자 →
설정**에서 변경합니다. 변경은 즉시 적용됩니다.

| 영역 | 범위 |
|---|---|
| 사이트 | 제목, 설명, 로고, 파비콘, 사이드바 항목, 홈 섹션 |
| 서버 URL | OPDS 카탈로그와 Kobo 동기화 링크에 사용되는 절대 기본 URL. 비워 두면 요청마다 자동 감지합니다. 감지된 호스트가 잘못된 경우, 예를 들어 경로를 재작성하는 프록시 뒤에 있을 때만 설정하십시오 |
| 접근 | 회원가입 허용 여부, 로그인 필수 여부, 게스트 접근 모드, 라이브러리별 게스트 노출 |
| 권한 | 36개 권한 전체를 역할별로 제어 — 읽기, 개인 기능, 라이브러리 콘텐츠, 연동, 관리 |
| 이메일 (SMTP) | 호스트, 포트, 사용자명, 비밀번호, 발신 주소, TLS 모드, 사설 네트워크 연결 허용, 연결 테스트. 이메일 인증과 비밀번호 재설정 활성화도 여기입니다 |
| 리더 기능 | 본문 심층 검색, 사용자 폰트 업로드, 표지에 표시할 참여 지표 |
| 트래커 | AniList / MyAnimeList 동기화 켜기·끄기 |
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

## 인증 쿠키

설정할 것이 없습니다. 두 속성 모두 요청마다 유도됩니다.

**`Secure`**는 요청이 HTTPS로 도착했을 때 설정됩니다. 직접 연결이든, 신뢰하는
프록시의 `X-Forwarded-Proto`를 통한 것이든 마찬가지입니다(위의 `TRUST_PROXY` 참고).
평문 HTTP에서는 생략되는데, 브라우저가 안전하지 않은 연결에서 `Secure` 쿠키를
조용히 버리기 때문이며 그 증상은 비밀번호가 틀린 것처럼 나타납니다.

**`Domain`**은 설정하지 않으므로, 쿠키는 이를 내려준 호스트에만 적용됩니다.

**`SameSite`**는 `Lax`이며 그대로 두어야 합니다. NovelHub에는 CSRF 토큰도 오리진
검사도 없으므로 `SameSite`가 유일한 CSRF 방어 수단입니다. `None`으로 완화하면 그
보호를 없애고 아무것도 대신 두지 않는 셈이 됩니다.

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
