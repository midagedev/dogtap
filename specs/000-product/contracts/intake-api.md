# Intake API Contract

This document defines the initial Dogtap-compatible endpoints. Exact Datadog payload compatibility is implemented incrementally with fixtures.

## Health

### `GET /healthz`

Returns process health.

### `GET /readyz`

Returns readiness for configured intake and storage.

## Dashboard API

### `GET /api/events`

Query parameters:

- `source`
- `service`
- `env`
- `userId`
- `accountId`
- `workspaceId`
- `caseId`
- `traceId`
- `route`
- `status`
- `limit`

Returns recent event envelopes, excluding raw body by default.

### `GET /api/events/{id}`

Returns one event envelope with decoded payload if allowed by mode.

### `GET /api/validation/failures`

Returns validation failures grouped by rule and source.

### `POST /api/debug-bundles`

Creates a debug bundle from a filter.

Request body:

- `source`
- `service`
- `env`
- `userId`
- `accountId`
- `workspaceId`
- `caseId`
- `traceId`
- `route`
- `status`
- `limit`

Returns:

- `bundleId`
- `createdAt`
- `filter`
- `summary`
- `events`
- `validationFailures`
- `datadogQueries`
- `redactionReport`

### `POST /api/diagnostics`

Creates a live diagnostics snapshot for agents, CI jobs, local dev servers, and
Docker Compose users.

Request body:

- `limit`: maximum retained events to inspect
- `filter`: same fields as `POST /api/debug-bundles`; scopes returned events,
  report, debug bundle, metrics, and assertions
- `expect`: assertion expectations for observed telemetry
- `workflowContract`: one inline workflow contract definition
- `workflowContracts`: multiple inline workflow contract definitions
- `useDefaultWorkflowContracts`: when true and no explicit workflow contracts
  are supplied, evaluate Dogtap's built-in dashboard readiness contract

`expect` fields:

- `nonEmpty`
- `sources`
- `payloadKinds`
- `services`
- `sessions`
- `traces`
- `cases`
- `routes`
- `metrics`
- `endpoints`

Returns:

- `healthz`
- `readyz`
- `events`
- `report`
- `debugBundle`
- `metrics`
- `assertions`
- `workflowContracts` when workflow contracts were supplied or requested

Expectation failures are represented in `assertions.status` and do not turn the
HTTP response into an error. This lets agents parse missing-signal hints without
special HTTP status handling.

Workflow contract failures are represented in each contract result under
`workflowContracts[].status`. They are separate from `assertions.status` so
existing diagnostics callers can keep using missing-signal assertions without a
semantic change.

### `POST /api/diagnostics/archive`

Creates a downloadable zip archive with the same diagnostic evidence as
`POST /api/diagnostics`.

Request body:

- same as `POST /api/diagnostics`

Returns `application/zip` containing:

- `summary.md`
- `assertions.json`
- `workflow-contracts.json` when workflow contracts were supplied or requested
- `events.json`
- `report.json`
- `debug-bundle.json`
- `metrics.txt`
- `healthz.json`
- `readyz.json`
- `manifest.json`

## RUM Intake

### `POST /rum`

Dogtap local endpoint for RUM payloads.

### `POST /datadog-intake-proxy`

Proxy-compatible endpoint for Datadog Browser RUM `proxy` configuration.

Expected behavior:

- Capture query parameter used by the browser SDK to identify the Datadog forward path.
- Decode request body.
- Normalize RUM context fields.
- Validate user/account/workspace/session fields.
- Forward only if forwarding mode is enabled.
- When forwarding, preserve only safe relative `ddforward` values matching
  `/api/v2/rum` or `/api/v2/replay`; reject absolute URLs and path mismatches
  to avoid open-proxy behavior.

## Logs Intake

### `POST /v1/input`

Compatible local endpoint for Datadog logs v1 style intake.

### `POST /api/v2/logs`

Compatible local endpoint for Datadog logs v2 style intake.

Expected content types:

- `application/json`
- `application/json;simple`
- `text/plain`
- `application/logplex-1`

Expected encodings:

- identity
- gzip

## APM Intake

### `PUT /v0.3/traces`

Datadog Agent trace API style endpoint.

### `PUT /v0.4/traces`

Compatibility endpoint.

### `PUT /v0.5/traces`

Compatibility endpoint.

Expected behavior:

- Capture trace payload.
- Decode known payload encodings.
- Normalize trace and span IDs.
- Build trace tree where possible.
- Validate service/env/version and route/resource fields.

## OTLP Intake

### `POST /v1/traces`

OTLP HTTP traces.

### `POST /v1/logs`

OTLP HTTP logs.

### `POST /v1/metrics`

OTLP HTTP metrics.

### gRPC on `4317`

Initial services:

- traces
- logs
- metrics

## Validation Report API

### `GET /api/reports/latest`

Returns latest validation report.

### `POST /api/replay`

Replays fixture payloads into the local validator.

## Security Requirements

- Never return configured Datadog API keys.
- Redact sensitive headers by default.
- Require explicit config to return raw payloads outside local mode.
