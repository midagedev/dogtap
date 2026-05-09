# Implementation Plan: Dogtap Product Foundation

## Status

Draft

## Technical Direction

Dogtap should start as a single Docker image with one backend process and an embedded dashboard. The first implementation should optimize for clear local setup, deterministic CI behavior, and production-safe extension points.

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

MVP storage should use an in-memory ring buffer with optional SQLite persistence.

Modes:

- `local`: ring buffer plus optional snapshot files
- `ci`: ring buffer plus JSON report
- `forward`: ring buffer plus bounded SQLite metadata
- `prod`: redacted metadata only, bounded by TTL and count

### Dashboard

React with a compact operational UI:

- Request stream
- Validation failure inbox
- Payload detail
- Correlation view
- Config view
- Debug bundle export

### Protocols

Initial protocol support:

- RUM HTTP proxy endpoint
- APM trace HTTP endpoint on `8126`
- Logs HTTP intake endpoint
- OTLP HTTP on `4318`
- OTLP gRPC on `4317`

DogStatsD and profiles should be future work.

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

Dashboard and CI reporter read from the same event store.
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

## Risks

| Risk | Mitigation |
| --- | --- |
| Datadog private payload formats drift | Prefer documented surfaces and fixture-based compatibility tests |
| Production proxy introduces telemetry loss | Default to local/CI first, production tee later, fail-open behavior |
| Dashboard scope grows into Datadog clone | Constitution and roadmap keep scope focused |
| Raw payload storage creates compliance risk | Redact before persistence, disable raw prod storage |
| APM payload decoding is complex | Evaluate reuse or reference behavior from `dd-apm-test-agent` |

## Definition of Done for MVP

- One Docker command starts Dogtap locally.
- RUM and logs payloads appear in the dashboard.
- APM traces from a sample app appear in the dashboard.
- Missing service tags and missing user/account/workspace context are flagged.
- CI mode can replay fixtures and fail on validation errors.
- Documentation explains safe production boundaries.
