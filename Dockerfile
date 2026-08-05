
FROM --platform=$BUILDPLATFORM oven/bun:alpine AS frontend-builder
WORKDIR /app/web

COPY web/package.json web/bun.lock* ./
RUN bun install --frozen-lockfile

COPY web/ ./
RUN bun run build

FROM --platform=$BUILDPLATFORM golang:1.26.5-alpine AS backend-builder
WORKDIR /app

RUN apk add --no-cache git build-base

COPY go.mod go.sum ./
RUN go mod download

COPY db/ ./db/
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

COPY --from=frontend-builder /app/cmd/api/dist ./cmd/api/dist

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o novelhub ./cmd/api

FROM --platform=$BUILDPLATFORM alpine:3.24 AS alpine-assets
RUN apk add --no-cache ca-certificates tzdata
RUN mkdir -p /data

FROM alpine:3.24
WORKDIR /app

COPY --from=alpine-assets /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=alpine-assets /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=alpine-assets /data /data

COPY --from=backend-builder /app/novelhub .

EXPOSE 3434

ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=3434
ENV DATA_DIR=/data
ENV SQLITE_DB_PATH=/data/novelhub.db
ENV RESTORE_AUTO_RESTART=true

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -q -O /dev/null "http://127.0.0.1:${SERVER_PORT:-3434}/api/v1/health" || exit 1

CMD ["./novelhub"]
