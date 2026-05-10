# Implementation Plan: Dogtap Product Foundation

## Status

Active implementation baseline. This plan records the current architecture and
remaining boundaries, not a pre-implementation proposal.

## Technical Direction

Dogtap is implemented as a single Docker image with one Go backend process and
an embedded React dashboard. The implementation optimizes for clear local
setup, deterministic CI behavior, agent-readable diagnostics, workflow
observability contracts, and production-safe extension points.

Implementation should be orchestrated through parallel agent lanes. Each lane owns a bounded surface, produces fixture-backed evidence, and passes the relevant gate before integration. The canonical orchestration plan lives in `docs/AGENT_ORCHESTRATION.md`; the canonical gates live in `specs/000-product/gates.md`.

## Recommended Stack

### Backend

Go is the preferred initial backend language.

Reasons:

- Strong fit for HTTP proxying, streaming, bounded concurrency, and single-binary Docker images
- Easy static asset embedding for the dashboard
- Lower operational overhead than a multi-process Node plus backend stack
- Good libraries for OTLP, gRPC, gzip, msgpack, and structured logging

Alternative:

- TypeScript backend with React dashboard is faster for UI iteration but less attractive for production proxy mode.

Decision:

- Start with Go backend and React/TypeScript dashboard embedded as static assets.

### Storage

Storage currently uses bounded in-memory retention with optional JSON file
persistence. SQLite remains deferred until forwarding or production metadata
retention needs require a relational store.

Modes:

- `local`: bounded memory plus optional file snapshot path
- `ci`: bounded memory plus replay reports or diagnostics artifacts
- `forward`: bounded memory/file metadata plus forwarding results
- production-facing modes: redacted metadata only, bounded by TTL and count

### Dashboard

React with a compact operational UI:

- Request stream
- Validation failure inbox
- Payload detail
- Correlation view
- Service map, traffic, metric samples, and browser session timeline
- Debug bundle export
- Workflow contract status

### Protocols

Current protocol support:

- RUM HTTP proxy endpoint
- APM trace HTTP endpoint on `8126`
- Logs HTTP intake endpoint
- OTLP HTTP on `4318`
- OTLP gRPC on `4317`
- Faro SDK smoke intake on `/faro`, `/collect`, and `/collect/`

Native DogStatsD intake and profiles should be future work. DogStatsD-style
metrics can be inspected through the Collector StatsD-to-OTLP bridge recipe.

## Architecture

```text
SDK / Agent / App
      |
      v
Intake adapters
  - rum
  - apm
  - logs
  - otlp
  - faro, smoke only
      |
      v
Decoder and normalizer
      |
      v
Validator pipeline
  - required fields
  - PII and secret scanning
  - context leak detection
  - cardinality hints
      |
      +-----------> Store redacted event metadata
      |
      +-----------> Forwarder, optional
                     - disabled
                     - forward
                     - tee
                     - redact-only

Dashboard, diagnostics, workflow contracts, and CI reporter read from the same
retained event envelopes.
```

## Milestones

### M0: Documentation and contracts

- Product spec
- Architecture
- Intake contracts
- Test strategy
- Production safety policy
- Final goal
- Agent orchestration plan
- Success gates

Gate:

- G0 Spec Readiness

### M1: Local RUM and logs inspector

- HTTP intake server
- RUM endpoint
- Logs endpoint
- Payload decoding
- Basic dashboard list/detail
- Required field validator

Gate:

- G1 Fixture Evidence
- G2 Runtime Contract
- G3 Protocol Intake

### M2: APM and correlation

- APM trace endpoint
- msgpack/JSON decoding as needed
- Trace tree view
- Trace/log/RUM correlation hints

Gate:

- G3 Protocol Intake
- G4 Product Usability

### M3: CI mode

- Headless runner
- Validation config
- JSON and Markdown reports
- Exit codes
- Fixture replay

Gate:

- G5 CI Contract

### M4: Forwarding

- Datadog forward mode for RUM and logs
- APM forwarding strategy
- Retry and failure accounting
- Redacted local storage

Gate:

- G6 Forwarding Safety

### M5: Production-safe tee

- Sampling
- Bounded queues
- Backpressure policy
- Redaction hardening
- Operational metrics

Gate:

- G7 Production Safety
- G8 Release Candidate

### M6: Generic adoption and diagnostics

- Generic frontend/backend adoption kit
- Datadog-preserving external injection strategy
- API-first diagnostics and downloadable diagnostics archive
- Agent telemetry triage runbook

Gate:

- G5 CI Contract diagnostics subset
- G8 Release Candidate generic adoption subset

### M7: Compatibility smokes

- RUM Session Replay direct and proxied intake
- Faro SDK smoke intake on `/faro`, `/collect`, and `/collect/`
- Production guidance to route Faro through Grafana Alloy and OTLP

Gate:

- G8 Release Candidate compatibility subset

### M8: Workflow observability contracts

- Event-backed workflow contract evaluator
- Built-in frontend/backend readiness contract
- Login, case-open, checkout, and report-export templates
- Dashboard workflow contract panel
- GitHub Actions recipe for E2E telemetry assertion

Gate:

- G5 CI Contract workflow contract subset

## Configuration Model

Dogtap should support both environment variables and YAML.

Example:

```yaml
mode: local
site: datadoghq.com
storage:
  kind: memory
  maxEvents: 1000
  ttl: 2h
validation:
  required:
    serviceTags: true
    rum:
      - user.id
      - account.id
      - workspace.id
  pii:
    enabled: true
    failOn:
      - access_token
      - authorization
      - email
forwarding:
  enabled: false
```

## Integration Points

### Browser RUM

Use Datadog Browser RUM proxy configuration to route events to Dogtap.
Prefer a runtime-configured proxy value so Dogtap can be injected without
editing application source for each adoption.

### Java APM

Point the Datadog Java tracer to Dogtap using `DD_TRACE_AGENT_URL` where
available, or `DD_AGENT_HOST` plus `DD_TRACE_AGENT_PORT`.

### External Injection

The generic adoption path should ship removable overlays for Docker Compose,
Kubernetes, CI services, and runtime frontend config. Dogtap must not require a
Dogtap-specific SDK. Unsupported Datadog Agent behaviors, including DogStatsD
and container log tailing, must be preserved on the Datadog production lane or
bridged through OTLP/log-forwarder integrations until Dogtap implements them
deliberately.

### Logs

For local and CI, send HTTP logs directly or route log forwarders to Dogtap. For production, prefer tee mode or a dedicated forwarding path.

### OTLP

Support direct OTLP and OpenTelemetry Collector pipelines.

### Workflow Contracts

Run workflow observability contracts after retained telemetry exists. Contracts
are YAML or JSON files evaluated by `dogtap diagnose`, the diagnostics API, and
the dashboard. Validate authored contract files first with
`dogtap contract validate <path>` or `make contract-check`. They should stay
event-backed and avoid becoming monitor/query logic.

## Risks

| Risk | Mitigation |
| --- | --- |
| Datadog private payload formats drift | Prefer documented surfaces and fixture-based compatibility tests |
| Production proxy introduces telemetry loss | Default to local/CI first, production tee later, fail-open behavior |
| Dashboard scope grows into Datadog clone | Constitution and roadmap keep scope focused |
| Raw payload storage creates compliance risk | Redact before persistence, disable raw prod storage |
| APM payload decoding is complex | Evaluate reuse or reference behavior from `dd-apm-test-agent` |

## Release-Candidate Done Baseline

- One Docker command starts Dogtap locally.
- RUM, logs, Datadog APM, OTLP, and smoke-level Faro payloads appear in the
  dashboard where supported.
- Session Replay, logs, trace spans, metrics, service map, traffic, browser
  session timeline, and workflow contract status are inspectable.
- Missing service tags and missing user/account/workspace/case context are
  flagged.
- CI mode can replay fixtures, capture live diagnostics, and fail on validation
  errors or explicit workflow contract failures.
- Documentation explains supported endpoints, limitations, production safety,
  and reversible adoption.
