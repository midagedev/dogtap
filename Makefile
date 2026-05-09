.PHONY: build test web-build run replay smoke-adoption shell-check

build:
	npm --prefix web run build
	go build ./cmd/dogtap

test:
	go test ./...

web-build:
	npm --prefix web run build

run:
	go run ./cmd/dogtap serve

replay:
	go run ./cmd/dogtap replay fixtures/rum/login.json fixtures/logs/json-log.json fixtures/apm/trace.json fixtures/otlp/traces.json

smoke-adoption:
	bash scripts/generic/smoke.sh

shell-check:
	bash scripts/check-shell-syntax.sh
