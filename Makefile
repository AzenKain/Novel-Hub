.PHONY: run dev sqlc tidy test build web-install web-dev web-build check docker-up docker-down docker-build-multi

ifeq ($(OS),Windows_NT)
BUN ?= bun
else
BUN ?= $(shell command -v bun 2> /dev/null || echo $(HOME)/.bun/bin/bun)
endif

run: web-build
	go run ./cmd/api

dev: web-dev

sqlc:
	sqlc generate

tidy:
	go mod tidy

test:
	go test ./...

build: sqlc
	go build -o novelhub ./cmd/api

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

DOCKER_IMAGE ?= azenkain/novel-hub:latest

docker-build-multi:
	docker buildx build --platform linux/amd64,linux/arm64 --no-cache -t $(DOCKER_IMAGE) --push .


