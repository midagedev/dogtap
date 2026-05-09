# 0009: API-First Live Diagnostics

Date: 2026-05-10

## Status

Accepted

## Context

Dogtap is commonly run as a Docker Compose service or sidecar. In that shape,
agents and CI jobs should not need shell access to the Dogtap container just to
answer whether RUM, replay, logs, traces, spans, service map inputs, traffic, or
metrics were collected.

The existing `dogtap diagnose` command already produced the right evidence, but
it was a CLI-first workflow. That made local host runs convenient while leaving
Compose and isolated E2E stacks to either mount volumes or exec into the
container.

## Decision

Expose live diagnostics through Dogtap's HTTP API:

- `POST /api/diagnostics` returns JSON diagnostics with health, readiness,
  retained events, latest validation report, debug bundle, metrics, assertions,
  and missing-signal hints.
- `POST /api/diagnostics/archive` returns the same evidence as a zip archive
  with `summary.md`, `assertions.json`, `events.json`, `report.json`,
  `debug-bundle.json`, `metrics.txt`, `healthz.json`, `readyz.json`, and
  `manifest.json`.
- The server API reuses the same diagnostics assertion and summary rendering
  code used by `dogtap diagnose`.
- Expectation failures remain data-level failures in `assertions.status`; the
  HTTP response stays `200` when the diagnostics query itself succeeds.

## Consequences

- Docker Compose and external agents can collect diagnostics with `curl` from
  outside the container.
- `dogtap diagnose` remains useful for host-side artifact directories and
  existing smoke scripts.
- Diagnostics API responses may include raw retained payloads in local mode,
  following the existing Dogtap mode and retention policy. Private adoption
  evidence must stay under ignored paths.
