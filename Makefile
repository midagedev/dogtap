.PHONY: build test web-build run replay diagnose contract-check smoke-adoption smoke-external-injection smoke-faro smoke-log-bridge smoke-statsd-bridge demo-seed demo-visual-check shell-check doc-check

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

diagnose:
	go run ./cmd/dogtap diagnose

contract-check:
	go run ./cmd/dogtap contract validate configs/contracts/*.yaml

smoke-adoption:
	bash scripts/generic/smoke.sh

smoke-external-injection:
	bash scripts/external-injection/smoke.sh

smoke-faro:
	bash scripts/faro/smoke.sh

smoke-log-bridge:
	bash scripts/log-bridge/smoke.sh

smoke-statsd-bridge:
	bash scripts/statsd-bridge/smoke.sh

demo-seed:
	bash scripts/demo/seed.sh

demo-visual-check:
	bash scripts/demo/visual-check.sh

shell-check:
	bash scripts/check-shell-syntax.sh

doc-check:
	bash scripts/check-doc-spec-alignment.sh
