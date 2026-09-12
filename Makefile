.PHONY: build run tidy test compose-up compose-down

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
