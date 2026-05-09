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
- Helm chart or ECS task example
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
