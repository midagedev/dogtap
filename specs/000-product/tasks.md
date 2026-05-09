# Tasks: Dogtap Product Foundation

## Phase 0: Repository Foundation

- [x] T000 Create spec-driven repository structure
- [x] T001 Write project constitution
- [x] T002 Write product specification
- [x] T003 Write implementation plan
- [x] T004 Write roadmap, testing, and production safety docs
- [x] T005 Write final goal, agent orchestration plan, and success gates

Gate:

- [x] G0 Spec Readiness review

## Phase 1: Technical Spike

- [ ] T010 Evaluate `dd-apm-test-agent` behavior and fixture format
- [x] T011 Capture sample RUM payloads from a minimal browser app
- [x] T012 Capture sample Datadog tracer payloads from a minimal instrumented app
- [x] T013 Capture sample logs HTTP intake payloads with JSON, text, and gzip
- [x] T014 Capture OTLP HTTP and gRPC payloads from OpenTelemetry SDKs
- [x] T015 Decide initial backend language and storage, with review trigger after G1 evidence

Suggested agents:

- RUM Agent
- Logs Agent
- APM Agent
- OTLP Agent

Gate:

- [x] G1 Fixture Evidence

Evidence note 2026-05-08:

- Fixture evidence harness added under `scripts/fixtures/`.
- Harness documentation added at `docs/fixtures/G1_FIXTURE_EVIDENCE.md`.
- Browser RUM SDK evidence captured from `@datadog/browser-rum` and promoted to
  `fixtures/rum/browser-rum-sdk-batch.json`.
- Logs JSON, text, and gzip evidence captured locally through Dogtap.
- APM evidence captured from the official Datadog Node tracer and promoted
  under `fixtures/apm/`; Java/Spring and `dd-apm-test-agent` comparison are
  deferred by `docs/decisions/0005-apm-fixture-scope.md`.
- OTLP HTTP and gRPC evidence captured from OpenTelemetry Node SDK exporters and
  promoted under `fixtures/otlp/`.
- `testdata/g1-evidence/latest/` is the ignored local output path for generated
  capture artifacts.

## Phase 2: Minimal Runtime

- [x] T020 Create backend service skeleton
- [x] T021 Add config loading from env and YAML
- [x] T022 Add bounded in-memory event store with optional file persistence
- [x] T023 Add request capture metadata model
- [x] T024 Add health and readiness endpoints
- [x] T025 Add Dockerfile and local compose example

Suggested agents:

- Runtime Core Agent
- CI Agent

Gate:

- [x] G2 Runtime Contract

## Phase 3: Intake Adapters

- [x] T030 Add RUM proxy-compatible HTTP endpoint
- [x] T031 Add logs HTTP intake endpoint
- [x] T032 Add APM trace endpoint on port `8126`
- [x] T033 Add OTLP HTTP endpoint on port `4318`
- [x] T034 Add OTLP gRPC endpoint on port `4317`
- [x] T035 Add gzip and content-type decoding

Suggested agents:

- RUM Agent
- Logs Agent
- APM Agent
- OTLP Agent

Gate:

- [x] G3 Protocol Intake

## Phase 4: Normalization and Validation

- [x] T040 Normalize service tags
- [x] T041 Normalize RUM user, account, workspace, route, and case fields
- [x] T042 Normalize trace and span IDs
- [x] T043 Normalize log attributes and message fields
- [x] T044 Implement required field validation
- [x] T045 Implement PII and secret detection
- [x] T046 Implement query-string and token leakage detection
- [x] T047 Implement context leak detection for logout and workspace switch flows

Suggested agents:

- Validation Agent
- Production Safety Agent

Gate:

- [x] G4 Product Usability validation subset

## Phase 5: Dashboard

- [x] T050 Create dashboard shell
- [x] T051 Add request stream table
- [x] T052 Add payload detail view
- [x] T053 Add validation failure inbox
- [x] T054 Add correlation view
- [x] T055 Add copyable Datadog search query builder
- [x] T056 Add debug bundle export API
- [x] T057 Add RUM Session Replay payload timeline viewer
- [x] T058 Add structured log viewer
- [x] T059 Add trace/span viewer

Suggested agents:

- Dashboard Agent

Gate:

- [x] G4 Product Usability

Evidence note 2026-05-08:

- Dashboard source-specific inspectors were added for RUM Session Replay
  payloads, logs, and trace spans.
- RUM replay intake recognizes `/datadog-intake-proxy?ddforward=/api/v2/replay`
  and direct `/api/v2/replay`, including Browser SDK multipart payloads with
  `event` metadata and `segment` attachments.
- E2E coverage validates replay, log, and trace/span panels across desktop and
  mobile Playwright projects.

## Phase 6: CI Mode

- [x] T060 Add headless validation command
- [x] T061 Add fixture replay command
- [x] T062 Add JSON report output
- [x] T063 Add Markdown report output
- [x] T064 Add exit code policy
- [x] T065 Document CI replay command shape

Suggested agents:

- CI Agent
- Validation Agent

Gate:

- [x] G5 CI Contract

## Phase 7: Forwarding

- [x] T070 Add Datadog site and API key config
- [x] T071 Add RUM forwarding
- [x] T072 Add logs forwarding
- [x] T073 Add APM forwarding or documented non-support decision
- [x] T074 Add forwarding failure accounting
- [x] T075 Add retry and drop policy

Suggested agents:

- Forwarding Agent
- Production Safety Agent

Gate:

- [x] G6 Forwarding Safety

## Phase 8: Production Safety

- [x] T080 Add redaction-before-persistence guarantee
- [x] T081 Add sampling config
- [x] T082 Add queue limit and backpressure behavior
- [x] T083 Add raw payload retention guardrails
- [x] T084 Add self-observability metrics
- [x] T085 Add production deployment runbook

Suggested agents:

- Production Safety Agent
- Release Agent

Gate:

- [x] G7 Production Safety

## Phase 9: Reference Adoption Profile

- [ ] T090 Configure a browser frontend to point RUM to Dogtap in local mode.
- [ ] T091 Configure a backend service to emit traces to Dogtap in local mode.
- [ ] T092 Configure backend logs to reach Dogtap in local mode.
- [ ] T093 Configure OTLP metrics to reach Dogtap in local mode.
- [ ] T094 Create a validation profile for representative login, workspace,
  object detail, export, and logout workflows.
- [ ] T095 Capture sanitized local telemetry evidence from a realistic
  application profile.

Notes:

- Project-specific adoption material should live outside the public repository
  unless it is sanitized and generic.
- Bundled Dogtap fixtures can validate Dogtap plumbing but do not count as real
  adoption evidence.

Suggested agents:

- Adoption Integration Agent
- CI Agent

Gate:

- [ ] G8 Release Candidate adoption profile subset

## Phase 10: Generic Adoption Kit

- [x] T100 Document the generic adoption decision and reversible integration
  boundary.
- [x] T101 Add copyable Docker Compose, frontend RUM, backend OTLP, backend
  Datadog tracer, and logs snippets for non-project-specific projects.
- [x] T102 Add a dashboard target summary for active local intake endpoints.
- [x] T103 Add a smoke verification script that exercises the generic intake
  path without requiring a real application.
- [x] T104 Update public quickstart and README to start from the generic
  adoption path instead of a project-specific integration.

Gate:

- [x] G8 Release Candidate generic quickstart subset

Evidence note 2026-05-08:

- Generic adoption decision recorded in
  `docs/decisions/0006-generic-adoption-kit.md`.
- Generic local profile added at `configs/generic-local.yaml`.
- Copyable templates added under `examples/adoption-kit/`.
- Empty dashboard state now shows copyable local intake targets for browser RUM,
  Datadog APM, OTLP HTTP, and OTLP gRPC.
- Generic smoke script added at `scripts/generic/smoke.sh` and wired to
  `make smoke-adoption`.
- Verification evidence is recorded in
  `docs/gates/G8_GENERIC_ADOPTION_SMOKE.md`.
