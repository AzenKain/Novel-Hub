# 리버스 프록시

NovelHub은 평문 HTTP로 서비스합니다. HTTPS로 접근하려면 앞단에 리버스 프록시를
둡니다. nginx, Caddy, Traefik 또는 Cloudflare가 TLS를 종료하고 요청을 전달합니다.

프록시는 실제 클라이언트를 NovelHub으로부터 가립니다. 두 헤더가 그 정보를 다시
전달하지만, 어떤 클라이언트든 이를 위조할 수 있기 때문에 NovelHub은 기본적으로
무시합니다. `TRUST_PROXY`가 누구에게 그 헤더를 설정할 권한이 있는지 지정합니다.

두 부분으로 나뉩니다. 둘 다 필요합니다.

| 구성 요소 | 위치 |
| --- | --- |
| NovelHub이 프록시를 신뢰 | `.env`의 `TRUST_PROXY` |
| 프록시가 헤더를 전송 | 사용하는 프록시의 설정 |

어느 한쪽이라도 빠지면 같은 증상이 나타납니다. 모든 사용자가 하나의 요청 수 제한
버킷을 공유하고, 로그인 쿠키에 `Secure`가 절대 붙지 않습니다.

---

## 1단계 — 프록시 신뢰하기

`.env`에서:

```bash
TRUST_PROXY=true
```

`true`는 루프백, 사설 또는 링크 로컬 주소에 있는 프록시를 포함하며, 거의 모든
셀프호스팅 구성이 여기에 해당합니다. 같은 머신의 nginx나 Caddy, 또는 같은 Docker
네트워크의 다른 컨테이너입니다.

프록시가 **공용** 주소에서 NovelHub에 접근한다면 — Cloudflare나 별도 호스트에 있는
프록시 — 대신 주소를 나열하십시오.

```bash
TRUST_PROXY=173.245.48.0/20,103.21.244.0/22,103.22.200.0/22
```

Cloudflare는 현재 사용 중인 범위를 <https://www.cloudflare.com/ips/>에 공개합니다.
쉼표로 구분해 전부 나열하십시오.

이 값을 변경한 뒤에는 NovelHub을 재시작하십시오.

---

## 2단계 — 헤더 전송하기

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

Caddy는 세 헤더를 모두 전송하고 인증서도 자동으로 발급받습니다. 그 외에 필요한
것은 없습니다.

### Traefik (docker-compose 라벨)

```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.novelhub.rule=Host(`library.example.com`)"
  - "traefik.http.routers.novelhub.entrypoints=websecure"
  - "traefik.http.routers.novelhub.tls.certresolver=letsencrypt"
  - "traefik.http.services.novelhub.loadbalancer.server.port=3434"
```

Traefik은 기본적으로 forwarded 헤더를 전송합니다.

### Cloudflare

Cloudflare는 `X-Forwarded-For`와 `X-Forwarded-Proto`를 알아서 전송하지만 공용
주소에서 접속하므로 `TRUST_PROXY=true`로는 포함되지 **않습니다**. 1단계에서 설명한
대로 공개된 범위를 나열하십시오.

SSL/TLS 모드는 **Full** 또는 **Full (strict)**로 설정하십시오. Flexible 모드에서는
Cloudflare가 브라우저에는 사이트가 HTTPS라고 알리면서 오리진과는 평문 HTTP로
통신하므로, 쿠키가 알아채기 어려운 방식으로 깨집니다.

---

## 3단계 — 검증

HTTPS로 로그인한 뒤 **DevTools → Application → Cookies**를 여십시오.
`access_token` 행에 `Secure`가 표시되어야 합니다.

그렇지 않다면 거꾸로 짚어 보십시오.

1. `TRUST_PROXY`가 설정되어 있고, NovelHub을 재시작했습니까?
2. 그 값이 프록시가 접속하는 주소를 포함합니까? NovelHub 로그에서 들어오는 요청의
   출처 IP를 확인하십시오. 공용 주소라면 `true`만으로는 부족하고 명시적인 목록이
   필요합니다.
3. 프록시가 `X-Forwarded-Proto`를 보내고 있습니까? 프록시 쪽에서 확인하십시오.

```bash
curl -sI https://library.example.com/api/v1/health
```

---

## 프록시로만 접근 제한하기

프록시가 같은 호스트에 있다면 그 외의 어떤 것도 NovelHub에 접근할 필요가 없습니다.
루프백에 바인딩해서 LAN에 포트가 노출되지 않도록 하십시오.

네이티브 설치는 `.env`에서:

```bash
SERVER_HOST=127.0.0.1
```

Docker는 `docker-compose.yml`에서:

```yaml
ports:
  - "127.0.0.1:3434:3434"
```

---

## 참고

**서브패스로 서비스하기**(`example.com/novelhub`)는 지원되지 않습니다. 프론트엔드가
절대 에셋 경로로 빌드되기 때문입니다. 대신 서브도메인을 사용하십시오.

**WebSocket**은 사용하지 않으므로 업그레이드 헤더가 필요하지 않습니다.

**대용량 업로드**를 위해서는 프록시의 본문 크기 제한을 올려야 합니다. nginx는
기본값이 1 MB이며, `client_max_body_size`를 설정하기 전까지 책 업로드를 `413`으로
거부합니다. Caddy와 Traefik에는 그런 기본값이 없습니다.
