# waxflow — vendored core (NovelHub)

This directory is a **trimmed vendor copy** of the core of
[github.com/colespringer/waxflow](https://github.com/colespringer/waxflow).

## Source & license

- **Upstream**: https://github.com/colespringer/waxflow
- **Version**: `v0.0.0-20260807051046-647ecd7841ce` (pseudo-version, commit `647ecd7`)
- **License**: MIT, Copyright (c) 2026 Cole Springer — see `LICENSE` (preserved verbatim)
- **Copied on**: 2026-08-12
- Import paths rewritten: `github.com/colespringer/waxflow` → `novelhub/pkg/waxflow`

## Why vendored

Upstream is a young single-maintainer library (0 stars, 65 commits at copy time).
NovelHub's audiobook merger (EPIC 9.1) depends on its core pipeline
(Probe/OpenStream/Concat/Transcode → AAC-LC M4B with Nero chapters), verified
against real files on 2026-08-12. Vendoring makes the build hermetic: the repo
could vanish tomorrow and `go build ./...` still works.

## What was stripped (vs upstream)

Exactly what NovelHub does not run:

- `cli/`, `server/` (HTTP service incl. HLS), `client/`, `internal/` (server
  infra: hls, jobs, cache, metrics, ulid, uploads, …)
- `tests/` (integration suite), `testdata/` (fixtures), `*_test.go` (need the
  dropped `internal/testutil`)
- `Makefile`, `Dockerfile`, `compose.yaml`, `scripts/`, `MAINTENANCE.md`

Kept: `audio/`, `codec/` (pcm, flac, alac, mp3, aac, opus, vorbis),
`container/` (riff, aiff, ogg, mp4, adts, mpa, flacn, mka), `dsp/`, `format/`,
`source/`, `waxerr/`, root package (`waxflow.go`, `timeline.go` = Concat,
`options.go`, …), `docs/` (ADRs), `LICENSE`, `THIRD-PARTY-NOTICES.md`, README.

## Usage contract (documented upstream limitations, relevant to the merger)

- AAC encoder is **AAC-LC, mono/stereo only** (no SBR/PS/HE-AAC; multichannel
  decode refuses silently-unsupported configs by name).
- Output is **progressive fragmented MP4** (fMP4) — plays in browsers, VLC,
  Apple Books, Audiobookshelf. If a device fails to seek fMP4, re-mux to
  classic MP4 with `Eyevinn/mp4ff` (re-mux, no re-encode) — scoped fix, not built.
- Engines route through `waxflow.New()`; only the merger path is exercised.

## Update policy

Treat as frozen. Do not edit unless a bug forces it. To upgrade: re-copy the
kept dirs from upstream, re-run the import rewrite, re-verify with the merger
integration tests (`internal/services/audiobook_service_test.go`).