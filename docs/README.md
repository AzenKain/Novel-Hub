# NovelHub Documentation

| Language | |
| --- | --- |
| English | [Configuration](en/configuration.md) · [Deployment](en/deployment.md) · [Reverse Proxy](en/reverse-proxy.md) |
| Tiếng Việt | [Cấu hình](vi/configuration.md) · [Triển khai](vi/deployment.md) · [Reverse Proxy](vi/reverse-proxy.md) |
| 日本語 | [設定](ja/configuration.md) · [デプロイ](ja/deployment.md) · [リバースプロキシ](ja/reverse-proxy.md) |
| 한국어 | [설정](ko/configuration.md) · [배포](ko/deployment.md) · [리버스 프록시](ko/reverse-proxy.md) |
| 简体中文 | [配置](zh/configuration.md) · [部署](zh/deployment.md) · [反向代理](zh/reverse-proxy.md) |

## Where settings live

NovelHub keeps configuration in two places, and the split is deliberate.

**Environment variables** hold only what must be known before the database
opens, or what cannot be stored in the database because it protects the
database. That is three secrets, one proxy setting, and a set of optional
performance knobs.

**The admin UI** holds everything else: site identity, registration, guest
access, per-role permissions, upload limits, and rate limits. These take effect
immediately, without a restart.

If you are looking for a setting and it is not in [Configuration](en/configuration.md),
it is in the admin Settings page.
