# G7 Production Safety Evidence

## Status

Passed on 2026-05-08.

Implemented safety controls cover redaction, retention guardrails, operational
metrics, sampling, explicit queue/backpressure behavior, Datadog-unavailable
behavior tests, storage-unavailable behavior tests, and a runbook.

## Evidence Implemented

- Raw payload retention is allowed by default only in local mode.
- Forward, tee, CI, and redact-only modes store decoded redacted payloads by
  default.
- Headers and query values are redacted before event persistence.
- File storage writes bounded snapshots with `0600` temp files.
- Event count and TTL are configurable.
- Sampling is configurable with `DOGTAP_SAMPLING_RATE`; production forward,
  tee, and redact-only modes default to sampled Dogtap copies.
- Sampling controls Dogtap copy retention only; configured forwarding still runs
  before a Dogtap copy is sampled out.
- Intake queue pressure is bounded with `DOGTAP_QUEUE_MAX_IN_FLIGHT` and the
  explicit `drop-newest` backpressure policy.
- Production queue-full behavior drops the Dogtap copy, returns accepted, and
  increments backpressure metrics.
- Production storage failure behavior drops the Dogtap copy, returns accepted,
  and increments storage-drop metrics.
- Datadog-unavailable behavior is tested for production forward mode: intake
  remains accepted, retries are bounded, and forwarding metadata records the
  upstream failure.
- `/metrics` exposes retained event, validation, forwarding, sampling,
  backpressure, and storage-drop counters.
- Production deployment and bypass procedure is documented in
  `docs/runbooks/PRODUCTION_DEPLOYMENT.md`.

## Verification

```bash
go test ./internal/config ./internal/server ./internal/store
go test ./...
```
