.PHONY: build test web-build run replay smoke-adoption smoke-external-injection smoke-faro demo-seed demo-visual-check shell-check

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

smoke-external-injection:
	bash scripts/external-injection/smoke.sh

smoke-faro:
	bash scripts/faro/smoke.sh

demo-seed:
	bash scripts/demo/seed.sh

demo-visual-check:
	bash scripts/demo/visual-check.sh

shell-check:
	bash scripts/check-shell-syntax.sh
