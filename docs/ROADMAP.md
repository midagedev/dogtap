# Roadmap

## Final Target

Dogtap should become a Datadog-compatible telemetry intake inspector and validation gateway that supports local mock behavior, CI contract validation, staging forwarding, and production-safe teeing.

The roadmap is designed for parallel agent execution. Each phase has a gate defined in `specs/000-product/gates.md`.

## Phase 0: Definition

Status: complete

- Product concept
- PRD
- Spec Kit style artifacts
- Architecture
- Testing strategy
- Production safety rules
- Final goal
- Agent orchestration plan
- Success gates

Gate:

- G0 Spec Readiness

## Phase 1: Local Inspector MVP

Goal: make local telemetry visible.

Status: complete for the fixture-backed local inspector MVP

- Docker image
- RUM intake
- logs intake
- bounded in-memory and optional file event store
- dashboard stream, detail, failure inbox, correlation, and query views
- required field validation
- PII detection

Gate:

- G1 Fixture Evidence: passed
- G2 Runtime Contract: passed
- G3 Protocol Intake: passed

## Phase 2: APM and Correlation

Goal: connect frontend, backend, and logs.

Status: smoke fixture-backed MVP complete

- APM trace intake
- trace/log/RUM correlation hints
- Datadog query builder
- debug bundle export

Gate:

- G3 Protocol Intake: passed
- G4 Product Usability: passed

## Phase 3: CI Contract Mode

Goal: prevent telemetry regressions before deployment.

Status: complete with local and GitHub Actions verification

- headless validation command
- fixture replay
- JSON and Markdown reports
- exit code policy
- CI replay/report shape
- API-first live diagnostics for Docker Compose and external agents

Gate:

- G5 CI Contract: passed

## Phase 4: Forward Mode

Goal: inspect staging telemetry while still sending to Datadog.

Status: RUM/logs complete, APM deferred

- RUM forwarding
- logs forwarding
- forwarding result visibility
- retry and drop policy
- redacted local samples

Gate:

- G6 Forwarding Safety: passed for RUM/logs

## Phase 5: Production-Safe Tee

Goal: support limited production use without becoming a reliability risk.

Status: complete

- sampling: implemented
- bounded queue: implemented
- redaction-before-persistence: implemented
- no raw payload by default: implemented
- operational metrics: implemented
- deployment runbook: written

Gate:

- G7 Production Safety: passed

## Phase 6: Ecosystem

Goal: make the project useful outside one company.

Status: release-candidate evidence complete for the first public release
candidate.

- public documentation cleanup
- sample apps
- validation profile examples
- OpenTelemetry Collector recipes
- generic frontend/backend adoption kit: complete
- copyable Docker Compose and environment snippets: complete
- Datadog-preserving external injection strategy: complete
- Compose and Kubernetes sidecar injection templates: complete
- frontend/backend Compose external injection smoke: complete
- log-forwarder bridge guidance: complete
- OpenTelemetry Collector tee recipe: complete
- RUM proxy runtime-config guidance: complete
- RUM proxy canary guide: complete
- public CI and community contribution surface: complete
- release binary and container publishing workflow: complete
- seeded dashboard demo and live visual verification: complete
- public support matrix and release candidate runbook: complete
- realistic sanitized adoption profile: complete
- experimental Faro SDK compatibility smoke: complete
- production-grade Faro routing guidance through Grafana Alloy `faro.receiver`
  to OTLP: complete

Gate:

- G8 Release Candidate: passed for first public release-candidate evidence
  (`docs/gates/G8_RELEASE_CANDIDATE.md`)

## Phase 7: Compatibility Smokes

Goal: validate adjacent telemetry SDKs without turning Dogtap into a second
collector implementation.

Status: Faro SDK compatibility smoke complete; production native Faro parity is
not in scope.

- native Faro intake for smoke at `/faro`, `/collect`, and `/collect/`
- external-injection frontend workflow at `/faro`
- `make smoke-faro` verification path
- documented production guidance to use Grafana Alloy `faro.receiver` and OTLP
  export into Dogtap

Gate:

- G8 Release Candidate: passed for Faro SDK smoke subset

## Phase 8: Workflow Observability Contracts

Goal: make Dogtap valuable as a telemetry contract test runner for real
frontend/backend workflows.

Status: first workflow contract slice complete.

- event-backed workflow contract evaluator
- built-in frontend/backend readiness contract
- login workflow contract template
- diagnostics API `workflowContracts` field
- diagnostics archive `workflow-contracts.json`
- `dogtap diagnose -workflow-contract` CLI support
- dashboard workflow contract panel
- templates for login, checkout, case open, and report export
- reusable GitHub Actions example for running a workflow contract after E2E

Gate:

- G5 CI Contract: passed for workflow contract diagnostics subset

## Next Implementation Roadmap

This section tracks valuable implementation work discovered while aligning the
Spec Kit artifacts, public docs, and code. These are intentionally ordered by
user value and implementation leverage, not by protocol breadth.

### Chunk A: Contract Authoring Guardrails

Goal: make workflow contracts easy to write correctly.

Status: complete.

- Add a JSON Schema for workflow contract YAML/JSON: complete.
- Add `dogtap contract validate <path>` to validate names, duplicate check IDs,
  supported check types, selector fields, and regex syntax before CI runs:
  complete.
- Add editor-friendly examples for service names and route regexes: covered by
  bundled templates and `schemas/workflow-contract.schema.json`.

Why it matters: workflow contracts are now Dogtap's strongest differentiator,
but users need fast feedback before they run a full app workflow.

### Chunk B: Dashboard Contract Evidence Drilldown

Goal: make the dashboard explain why each contract passed or failed.

Status: complete.

- Show pass and fail checks, not only failing checks: complete.
- Link matched event IDs and trace IDs directly into the stream/detail pane:
  complete.
- Show the selector that was evaluated and the closest observed alternatives
  when a check fails: complete.

Why it matters: this turns contract failures into immediate debugging guidance
for humans and coding agents.

### Chunk C: Diagnostics Root-Cause Classifier

Goal: make missing telemetry diagnosis more mechanical.

Status: first slice plus structured debugging maintenance complete.

- Add a diagnostics section that classifies likely causes: SDK not initialized,
  endpoint not reachable, wrong route/service selector, sampling disabled,
  replay consent/sample mismatch, log forwarder missing, OTLP exporter disabled,
  trace propagation missing: first slice complete for common source, payload
  kind, context, metric, trace, endpoint routing, OTLP exporter, and Dogtap API
  failures.
- Include source-specific next commands and expected network targets: first
  slice complete in `assertions.rootCauses` and `summary.md`.
- Keep it evidence-backed by observed endpoints, sources, sessions, traces, and
  recent validation failures: first slice complete from diagnostics assertions.

Why it matters: Dogtap should help agents explain why telemetry did not arrive,
not only state that it is missing.

### Chunk D: Agent Gap Bridge Recipes

Goal: preserve existing Datadog usage while covering common Agent-only gaps.

Status: complete for the first Agent-gap scope.

- Add practical bridge recipes for stdout/container logs into Dogtap logs HTTP
  or OTLP logs: complete for OpenTelemetry Collector `filelog` to OTLP HTTP
  JSON, with `make smoke-log-bridge`.
- Add DogStatsD-to-OTLP guidance or a fixture-backed bridge example: complete
  for OpenTelemetry Collector `statsd` to OTLP HTTP JSON, with
  `make smoke-statsd-bridge`.
- Run bridge smokes in GitHub Actions and upload diagnostics artifacts:
  complete.
- Keep Dogtap itself from becoming a full Datadog Agent replacement.

Why it matters: teams often rely on Datadog Agent behavior. Bridge recipes keep
Dogtap adoption reversible without pretending full Agent parity exists.

### Chunk E: Spec/Doc Drift Enforcement

Goal: prevent docs and Spec Kit artifacts from drifting after feature work.

Status: first slice implemented by `make doc-check`.

- Check that release-candidate spec artifacts no longer claim draft status.
- Check that the data model includes implemented sources, telemetry details,
  diagnostics snapshots, and workflow contract results.
- Run the check in CI alongside shell syntax checks.

Why it matters: Dogtap is spec-driven, so stale specs are a product quality bug.

### Chunk F: Public Deployment Packaging

Goal: make Dogtap easy to trial in common deployment environments.

Status: first slice complete.

- Add a Helm values example for sidecar or companion-service deployment:
  complete for sidecar and companion values models under
  `examples/deployment/`.
- Add an ECS task definition example for Dogtap as an internal inspection
  target: complete with Dogtap marked non-essential.
- Include explicit retention, sampling, forwarding, and private-network
  warnings in each recipe: complete and checked by `make deployment-check`.

Why it matters: Docker Compose is enough for local adoption, but public users
need copyable deployment shapes before they can run team-level trials.

### Chunk G: Datadog API Compatibility Layer

Goal: let Datadog-oriented tools and agents query Dogtap without learning a
Dogtap-specific search API.

Status: first slice complete.

- Add read-only compatibility for Datadog logs, RUM, and spans search:
  complete for retained Dogtap telemetry under `/api/v2/*/events/search`.
- Add metric query compatibility:
  complete for simple retained timeseries samples through `/api/v1/query`.
- Keep the layer explicit about limits:
  complete in `docs/DATADOG_API_COMPATIBILITY.md` and
  `docs/decisions/0014-datadog-api-compatibility.md`.
- Add structured log aliases, metric point tag scope matching, Dogtap event ID
  links, and conservative trace ID alias matching for agent debugging:
  complete.

Why it matters: agent-driven debugging improves when the same Datadog API paths
used in production snippets can be pointed at Dogtap during local and CI runs.

### Chunk H: Persistent Queryable Storage

Goal: make retained telemetry survive restarts without adding an external
database service.

Status: first slice complete.

- Add `storage.kind=sqlite` with indexed metadata columns and full redacted
  event envelope JSON: complete.
- Keep storage bounded by TTL and max event count in SQLite, not only memory:
  complete.
- Preserve production safety by relying on the existing redaction-before-
  persistence path and fail-open storage-error behavior: complete with server
  tests for SQLite.
- Switch local examples to a SQLite-backed `/data/dogtap.db` volume where
  restart-safe retained telemetry is useful: complete.

Why it matters: the Datadog-compatible API and dashboard become more useful
when local, CI, isolated E2E, and dev-cluster evidence can survive process
restarts while staying bounded.

### Chunk I: Dashboard Observability UX

Goal: make logs and metrics easier to inspect without hiding payload truth.

Status: first slice complete.

- Add structured log drilldown for service, env, route, status, trace/span IDs,
  severity, and safe attributes: complete.
- Add retained metric charts and metric detail summaries using existing metric
  samples: complete.
- Verify desktop and mobile dashboard states visually with Playwright: complete
  for the mocked dashboard state.
- Keep raw/decoded payload access visible in local and CI modes: complete.

Why it matters: Dogtap's strongest user-facing value is answering whether a
workflow emitted usable logs, traces, metrics, RUM, and replay evidence.

### Chunk J: EKS Dev-Cluster Readiness

Goal: make Dogtap credible as a bounded internal inspection target in a dev EKS
cluster.

Status: planned.

- Add a private-by-default EKS dev overlay with single replica, non-root
  security context, bounded retention, and rollback instructions.
- Add a cluster smoke runbook that posts representative telemetry and verifies
  diagnostics, metrics, config safety, and retained events.
- Keep production Datadog or OTel Collector as the primary telemetry path; use
  Dogtap as a dev-cluster inspection lane.

Why it matters: team-level adoption needs a repeatable dev-cluster deployment
path, but Dogtap should not become a production observability backend.
