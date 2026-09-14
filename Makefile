.PHONY: build run tidy test compose-up compose-down migrate-existing smoke

CONF ?= configs/config.yaml
ifneq (,$(wildcard .env))
include .env
export
endif

build:
	go build -o bin/app ./cmd/app

run: build
	./bin/app -conf $(CONF)

tidy:
	go mod tidy

test:
	go test ./internal/pkg/... ./internal/biz/... ./internal/conf/... -count=1

compose-up:
	docker compose up --build -d

compose-down:
	docker compose down

migrate-existing:
	bash scripts/migrate_existing.sh

smoke:
	bash scripts/smoke.sh
