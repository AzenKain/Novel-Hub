.PHONY: run sqlc tidy test build web-install web-dev web-build check docker-up docker-down

run: web-build
	go run ./cmd/api

sqlc:
	sqlc generate

tidy:
	go mod tidy

test:
	go test ./...

build: sqlc
	go build ./cmd/api

BUN := $(shell command -v bun 2> /dev/null || echo $(HOME)/.bun/bin/bun)

web-install:
	cd web && $(BUN) install

web-dev:
	cd web && $(BUN) run dev

web-build:
	cd web && $(BUN) run build

check: sqlc test web-build build

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down
