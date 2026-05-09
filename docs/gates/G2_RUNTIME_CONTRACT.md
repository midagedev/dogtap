# G2 Runtime Contract Evidence

## Status

Passed for current runtime skeleton.

## Evidence

Runtime contract criteria:

- Config loads from environment and YAML: `internal/config` tests cover defaults, env overrides, and `dogtap.example.yaml`.
- Event store has bounded retention: `internal/store` tests cover memory eviction and file-backed reload with max event bounds.
- Health and readiness endpoints exist: `GET /healthz` and `GET /readyz` are registered in `internal/server`.
- Intake adapters write event envelopes through a stable interface: all HTTP intake handlers use `intake.CaptureRequest` and `store.Store`.
- Dashboard API can read event envelopes: `GET /api/events`, `GET /api/events/{id}`, and `GET /api/validation/failures` are implemented.
- CI reporter can read validation results: `dogtap replay` returns JSON reports with validation status and exit codes.

## Verification

```bash
go test ./...
npm --prefix web run build
go run ./cmd/dogtap replay fixtures/rum/login.json fixtures/logs/json-log.json fixtures/apm/trace.json fixtures/otlp/traces.json
```

## Notes

The runtime currently supports `memory` and `file` storage. `file` storage is intended for local persistence and bounded snapshots, not long-term production telemetry retention.
