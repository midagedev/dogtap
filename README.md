# Dogtap

Dogtap is a Datadog telemetry intake inspector for local development, CI validation, and production-safe forwarding.

It is not a Datadog clone. The goal is narrower and more useful for teams that already use Datadog:

- Receive telemetry from Datadog SDKs and agents without sending it to Datadog in local mode.
- Show exactly what RUM, logs, traces, and OTLP payloads contain.
- Validate that required observability context is present before a release.
- Detect PII, token, query-string, and context-leak risks before payloads reach Datadog.
- Optionally forward or tee traffic to Datadog in staging and production with explicit safety controls.

## Why This Exists

Datadog is valuable, but expensive and hard to validate locally. Teams often discover telemetry problems only after deployment:

- Missing `DD_ENV`, `DD_SERVICE`, or `DD_VERSION`
- RUM events without `user.id`, `account.id`, `workspace.id`, or `case_id`
- Frontend errors that cannot be correlated with backend traces
- Logs that include query strings, emails, access tokens, or unstable high-cardinality fields
- Instrumentation that looks correct in code but does not produce useful Datadog facets

Dogtap makes these failures visible before they become operational debt.

## Project Shape

This repository is structured for spec-driven development. Product intent, architecture, tasks, test strategy, and release rules live in versioned documents before implementation.

```text
.
|-- .specify/
|   `-- memory/
|       `-- constitution.md
|-- specs/
|   `-- 000-product/
|       |-- spec.md
|       |-- plan.md
|       |-- tasks.md
|       |-- gates.md
|       |-- research.md
|       |-- data-model.md
|       |-- quickstart.md
|       `-- contracts/
|           `-- intake-api.md
`-- docs/
    |-- ARCHITECTURE.md
    |-- AGENT_ORCHESTRATION.md
    |-- CONCEPT.md
    |-- FINAL_GOAL.md
    |-- PRD.md
    |-- PRODUCTION_SAFETY.md
    |-- ROADMAP.md
    |-- TESTING.md
    |-- decisions/
    |-- references/
    `-- runbooks/
```

## Initial Modes

| Mode | Purpose | Datadog forwarding |
| --- | --- | --- |
| `local` | Replace Datadog during local development | No |
| `ci` | Assert telemetry contract from tests | No |
| `forward` | Inspect and forward telemetry to Datadog | Yes |
| `tee` | Forward first, sample-copy metadata to Dogtap | Yes |
| `redact-only` | Enforce PII and payload policy before forwarding | Yes |

## MVP Boundary

MVP should cover the telemetry surfaces that unblock day-to-day debugging:

- Browser RUM intake proxy and inspector
- Datadog APM agent-compatible trace intake on port `8126`
- Logs HTTP intake inspector
- OTLP HTTP and gRPC receiver
- Dashboard for recent payloads, validation errors, and correlation hints
- Exportable debug bundle for a single user, workspace, case, trace, or time window

## Development Model

Dogtap is intended to be built with agent orchestration. Work is split into independently testable lanes such as runtime core, RUM intake, logs intake, APM intake, OTLP intake, validation, dashboard, CI, forwarding, production safety, and adoption profiles.

Each lane has explicit gates. Fast parallel implementation is encouraged, but integration should stop when a gate fails.

## Non-Goals

- Reimplement Datadog dashboards, monitors, notebooks, or query engine
- Store production telemetry as a long-term observability backend
- Replace OpenTelemetry Collector
- Support every private Datadog endpoint in the first releases

## Current Status

Initial runtime implementation exists:

- Go backend with embedded React dashboard
- Config loading from YAML and environment variables
- Bounded in-memory event store and optional file persistence
- RUM, logs, APM HTTP, OTLP HTTP, and OTLP gRPC intake endpoints
- Normalization, validation, redaction, fixture replay, JSON and Markdown reports
- Dashboard stream/detail/payload, validation failure inbox, correlation hints,
  Datadog query builder, and debug bundle export
- RUM/logs forwarding with bounded retries and forwarding metrics

G1 fixture evidence has passed for the first product slice. Some bundled fixtures are still smoke fixtures, but promoted real-evidence fixtures now exist for Browser RUM, Datadog APM tracer, and OTLP SDK exports.

## Generic Adoption Quickstart

Start Dogtap locally:

```bash
docker compose up --build
```

Open:

```text
http://localhost:8080
```

Point an existing browser app at:

```text
http://localhost:8080/datadog-intake-proxy
```

Point a backend at Dogtap through either OTLP or an existing Datadog tracer:

```bash
# Host process
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf

DD_AGENT_HOST=localhost
DD_TRACE_AGENT_PORT=8126
```

For Docker Compose apps, use the service name `dogtap` instead of `localhost`.
Copyable templates live in [examples/adoption-kit](examples/adoption-kit/),
and the generic runbook is [Adopting Dogtap In A Generic App](docs/runbooks/ADOPTING_DOGTAP.md).

Smoke-test the generic path:

```bash
make smoke-adoption
```

## Local Development

```bash
npm --prefix web install
npm --prefix web run build
go test ./...
go run ./cmd/dogtap serve
```

Use persistent local storage:

```bash
go run ./cmd/dogtap serve -config configs/generic-local.yaml
```

With Docker Compose, events persist in the `dogtap-data` volume:

```bash
docker compose up --build
```

Replay sample fixtures:

```bash
go run ./cmd/dogtap replay fixtures/rum/login.json fixtures/logs/json-log.json fixtures/apm/trace.json fixtures/otlp/traces.json
```

## License

Apache-2.0. See [LICENSE](LICENSE).

Start with:

- [Final Goal](docs/FINAL_GOAL.md)
- [Concept](docs/CONCEPT.md)
- [PRD](docs/PRD.md)
- [Generic Adoption Runbook](docs/runbooks/ADOPTING_DOGTAP.md)
- [Spec](specs/000-product/spec.md)
- [Plan](specs/000-product/plan.md)
- [Tasks](specs/000-product/tasks.md)
- [Success Gates](specs/000-product/gates.md)
- [Agent Orchestration](docs/AGENT_ORCHESTRATION.md)
