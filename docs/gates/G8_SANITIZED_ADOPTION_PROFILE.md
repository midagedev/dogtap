# G8 Sanitized Adoption Profile

Date: 2026-05-09

## Status

Passed.

This evidence promotes `examples/external-injection-smoke/` from a basic smoke
fixture to Dogtap's first public-safe adoption profile. It models a normal
frontend plus backend stack that runs without Dogtap, then enables Dogtap only
through a removable Compose override and standard Datadog/OTLP endpoint
settings.

No private application names, hosts, credentials, customer data, or production
payloads are included.

## Profile Shape

Base stack:

- `frontend`: a browser-facing service with a runtime RUM proxy setting
- `backend`: an API service that can emit logs, Datadog APM traces, OTLP traces,
  and OTLP metrics
- no Dogtap service
- no Dogtap-specific SDK or application dependency

Injected stack:

- adds `dogtap` as a Compose service
- injects `DATADOG_RUM_PROXY_URL`
- injects `DD_TRACE_AGENT_URL`, `DD_AGENT_HOST`, and `DD_TRACE_AGENT_PORT`
- injects Datadog logs HTTP and OTLP exporter endpoints
- persists Dogtap events only in a disposable Docker volume

Rollback:

- rerunning the base Compose file removes Dogtap and all endpoint overrides

## Signal Evidence

The profile validates:

- Browser RUM from `external-smoke-frontend` with user, account, workspace,
  case, session, view, route, service, env, and version context.
- Session Replay through the same RUM proxy with `payloadKind=replay`, a
  multipart/zlib segment, and a decoded rrweb full snapshot suitable for
  dashboard DOM replay.
- A deliberate missing-context RUM event that fails required RUM validation.
- Backend Datadog logs HTTP with trace and span correlation fields.
- Backend Datadog APM trace intake on `/v0.5/traces`.
- Backend OTLP trace intake on `/v1/traces`.
- Backend OTLP metric intake on `/v1/metrics` with
  `external_injection.workflow.duration`.
- Service presence for both frontend and backend.
- Configuration-only enablement and rollback.

## Verification

Command:

```bash
make smoke-external-injection
```

Result:

```text
Dogtap external injection smoke passed.
Base stack: frontend and backend run without Dogtap-specific settings.
Injected stack: override adds Dogtap plus standard Datadog/OTLP endpoints.
Rollback: omitting the override removes Dogtap and endpoint overrides.
```

The smoke script asserts that Dogtap received:

- `source=rum`
- `payloadKind=replay`
- `source=logs`
- `source=apm`
- `source=otlp`
- `payloadKind=metric`
- `segmentEncoding=zlib`
- validation `status=fail`
- `required.rum.userId`
- `External Injection Workflow`
- `external_injection.workflow.duration`
- `external-smoke-frontend`
- `external-smoke-backend`

Related verification used for this gate:

```bash
git diff --check
go test ./...
npm --prefix web ci --ignore-scripts
npm --prefix web run build
make shell-check
make smoke-adoption
make smoke-external-injection
make demo-visual-check
```

The seeded demo visual check also verifies and screenshots the dashboard's DOM
Session Replay viewer on desktop and mobile. Generated screenshots are reviewed
locally and not committed.

## Public Safety

- Raw profile payloads are synthetic and public-safe.
- The profile uses placeholder IDs only.
- The Docker volume is removed at the end of the smoke.
- The public repository contains only sanitized commands, fixture code, and
  summarized results.
- Project-specific or long-running private adoption material remains under
  `.private/adoption/`, which is ignored by Git.

## Gate Decision

G8's realistic sanitized adoption profile requirement is satisfied for the
first public release candidate. Future real-project adoption evidence can add
private raw notes under `.private/adoption/` and publish only sanitized
summaries under `docs/gates/`.
