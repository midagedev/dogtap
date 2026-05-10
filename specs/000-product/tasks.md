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
- [x] T057 Add RUM Session Replay DOM replay viewer with payload timeline
  fallback
- [x] T058 Add structured log viewer
- [x] T059 Add trace/span viewer

Suggested agents:

- Dashboard Agent

Gate:

- [x] G4 Product Usability

Evidence note 2026-05-08:

- Dashboard source-specific inspectors were added for RUM Session Replay
  payloads, logs, and trace spans.
- Session Replay now renders decoded rrweb full snapshot records in an iframe
  DOM replay and keeps the payload timeline fallback for partial payloads.
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

- [x] T090 Configure a browser frontend to point RUM to Dogtap in local mode.
- [x] T091 Configure a backend service to emit traces to Dogtap in local mode.
- [x] T092 Configure backend logs to reach Dogtap in local mode.
- [x] T093 Configure OTLP metrics to reach Dogtap in local mode.
- [x] T094 Create a validation profile for representative identity, workspace,
  object detail, export, and missing-context workflows.
- [x] T095 Capture sanitized local telemetry evidence from a realistic
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

- [x] G8 Release Candidate adoption profile subset

Evidence note 2026-05-09:

- Phase 9 is satisfied by the public sanitized external-injection profile
  rather than a private project-specific profile.
- Evidence is recorded in `docs/gates/G8_SANITIZED_ADOPTION_PROFILE.md`.
  Raw/private long-running adoption notes remain under `.private/adoption/`.

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

## Phase 11: Datadog-Preserving External Injection

- [x] T110 Research official Datadog and OpenTelemetry injection surfaces for
  RUM proxying, tracer agent endpoints, Agent log collection, DogStatsD, and
  Collector sidecar patterns.
- [x] T111 Record an external injection ADR that distinguishes supported
  endpoint redirection from unsupported Datadog Agent parity claims.
- [x] T112 Add Compose and Kubernetes sidecar templates that can be applied as
  removable overlays to existing applications.
- [x] T113 Add Datadog-preserving env overlays for existing tracers and optional
  OTLP exporters.
- [x] T114 Document frontend runtime-config requirements for externally
  injected RUM proxy values.
- [x] T115 Document collector/log-forwarder bridge patterns for teams whose
  Datadog logs currently depend on Agent-side tailing.
- [x] T116 Add an executable Compose adoption fixture that proves Dogtap can be
  enabled and removed by changing only override files.
- [x] T117 Add an OpenTelemetry Collector tee example for traces/logs/metrics
  with Datadog primary and Dogtap inspection as a sampled secondary path.
- [x] T118 Add a RUM proxy canary runbook with Browser SDK version, raw-body,
  header stripping, allowlist, and rollback requirements.
- [x] T119 Capture a realistic sanitized adoption profile and publish only the
  safe summary under `docs/gates/`.

Gate:

- [x] G8 Release Candidate external injection subset

Evidence note 2026-05-09:

- RUM proxy canary runbook added at `docs/runbooks/RUM_PROXY_CANARY.md`.
- RUM/replay forwarding now preserves safe relative `ddforward` path/query
  values for `/api/v2/rum` and `/api/v2/replay` while rejecting absolute
  upstream URLs.
- Forwarding tests cover `ddforward` query preservation, sensitive inbound
  header stripping, replay path preservation, and unsafe URL rejection.
- Sanitized adoption profile evidence added at
  `docs/gates/G8_SANITIZED_ADOPTION_PROFILE.md`.
- `make smoke-external-injection` now validates normal RUM, multipart Session
  Replay with a decoded rrweb DOM snapshot, logs, APM traces, OTLP traces,
  OTLP metrics, one required-context validation failure, and configuration-only
  rollback.

## Phase 12: Faro SDK Compatibility Smoke

- [x] T120 Record the Faro compatibility decision and keep the production
  recommendation on Grafana Alloy `faro.receiver` to OTLP.
- [x] T121 Document Dogtap's experimental native Faro intake endpoints:
  `/faro`, `/collect`, and `/collect/`.
- [x] T122 Document the external-injection smoke frontend `/faro` workflow and
  `make smoke-faro` verification command.
- [ ] T123 Promote Faro compatibility beyond smoke only after fixture-backed
  production receiver behavior, retention, forwarding, and schema drift risks
  are covered by gates.

Gate:

- [x] G8 Release Candidate Faro SDK smoke subset

Evidence note 2026-05-09:

- Faro compatibility decision recorded in
  `docs/decisions/0008-faro-compatibility.md`.
- Native Faro intake is documented as experimental and scoped to integration
  smoke on `/faro`, `/collect`, and `/collect/`.
- The smoke path uses `examples/external-injection-smoke/frontend` at `/faro`
  and is documented as `make smoke-faro`.
- Production-grade Faro routing should use Grafana Alloy `faro.receiver` and
  export OTLP to Dogtap rather than depending on Dogtap's native Faro intake.

## Phase 13: Dashboard Intake Health

- [x] T130 Add dashboard intake health summaries by source and endpoint.
- [x] T131 Add browser session timeline grouping across RUM/Faro/replay and
  correlated logs, traces, and metrics.
- [x] T132 Add E2E coverage for the intake health and session timeline panels.
- [ ] T133 Promote the service map to an interactive graph after the dashboard
  has stable intake and session diagnostics.

Gate:

- [x] G4 Product Usability dashboard diagnostics subset

Evidence note 2026-05-09:

- Dashboard intake health surfaces source activity, endpoint activity, last
  seen age, and failing validation counts.
- Session timeline groups events by browser session and related correlation
  fields so frontend, log, trace, and metric signals can be inspected as one
  workflow.

## Phase 14: Agent-Readable Live Diagnostics

- [x] T140 Add a `dogtap diagnose` command that captures live Dogtap state into
  a single artifact directory.
- [x] T141 Add expectation assertions for source, payload kind, service,
  session, trace, route, metric, case, and endpoint presence.
- [x] T142 Add practical missing-signal hints for common frontend/backend local
  dev and isolated E2E configuration mistakes.
- [x] T143 Document how agents should use diagnostics artifacts while keeping
  project-specific evidence under private, ignored paths.

Gate:

- [x] G5 CI Contract diagnostics subset

Evidence note 2026-05-09:

- `dogtap diagnose` captures `healthz`, `readyz`, retained events, latest
  report, debug bundle, metrics, `assertions.json`, and `summary.md` into one
  artifact directory.
- Smoke and demo workflows can write diagnostics through `DOGTAP_ARTIFACT_DIR`;
  CI uploads smoke/demo diagnostics artifacts.
- `docs/runbooks/AGENT_TELEMETRY_TRIAGE.md` explains local dev and isolated E2E
  triage patterns while keeping private adoption evidence out of the public
  repository.

## Phase 15: API-First Live Diagnostics

- [x] T150 Add `POST /api/diagnostics` for JSON diagnostics with retained
  events, validation report, debug bundle, metrics, assertions, and practical
  missing-signal hints.
- [x] T151 Add `POST /api/diagnostics/archive` for a zip archive containing the
  same agent-readable files produced by `dogtap diagnose`.
- [x] T152 Reuse diagnostics assertion and summary rendering between the CLI
  and server API.
- [x] T153 Document Docker Compose and local dev usage through the diagnostics
  API, with CLI capture remaining available for artifact directories.

Gate:

- [x] G5 CI Contract diagnostics API subset

Evidence note 2026-05-10:

- Server tests cover scoped diagnostics assertions and archive file contents.
- Diagnostics API requests support `limit`, `filter`, and `expect` fields so
  agents can ask whether a specific app, service, session, trace, route, metric,
  source, or endpoint was observed.
- The archive endpoint returns `summary.md`, `assertions.json`, `events.json`,
  `report.json`, `debug-bundle.json`, `metrics.txt`, `healthz.json`,
  `readyz.json`, and `manifest.json`.

## Phase 16: Workflow Observability Contracts

- [x] T160 Add an event-backed workflow contract evaluator for event presence,
  log message presence, metric presence, trace correlation, and obvious
  sensitive value checks.
- [x] T161 Add frontend/backend and login workflow contract templates under
  `configs/contracts/`.
- [x] T162 Add diagnostics API and archive support for `workflowContracts`
  without changing existing `assertions.status` behavior.
- [x] T163 Add `dogtap diagnose -workflow-contract` and optional
  `-fail-on-workflow-contract` support for CI adoption.
- [x] T164 Surface built-in workflow contract results in the dashboard.
- [x] T165 Add more workflow templates for checkout/case-open/report-export
  once public fixture evidence exists.
- [x] T166 Add a reusable GitHub Actions example that runs a project E2E suite,
  then asserts a workflow contract through Dogtap diagnostics.

Gate:

- [x] G5 CI Contract workflow contract subset

Evidence note 2026-05-10:

- Contract unit tests cover pass/fail, trace ID alias correlation, and
  sensitive value detection.
- Diagnostics tests cover CLI artifacts, diagnostics API JSON, and archive file
  inclusion for `workflow-contracts.json`.
- Dashboard build succeeds with the built-in frontend/backend readiness panel.
- Follow-up templates cover login, case-open, checkout, and report-export, and
  `examples/github-actions/workflow-contract.yml` shows the intended CI
  assertion step after an existing E2E suite.

## Phase 17: Spec/Docs/Code Alignment

- [x] T170 Update `spec.md`, `plan.md`, `data-model.md`, and `quickstart.md`
  to reflect the implemented diagnostics, Faro smoke, metrics, and workflow
  contract surfaces.
- [x] T171 Update public docs and orchestration docs so the product positioning
  matches the code baseline.
- [x] T172 Add a docs/spec alignment check to prevent high-signal drift from
  returning in CI.
- [x] T173 Record the next implementation roadmap discovered during alignment,
  including contract authoring guardrails, dashboard evidence drilldown,
  diagnostics root-cause classification, Agent gap bridge recipes, and public
  deployment packaging.

Gate:

- [x] G0 Spec Readiness maintenance subset

Evidence note 2026-05-10:

- `make doc-check` verifies the most important Spec Kit and docs/code alignment
  markers.
- `docs/ROADMAP.md` now includes a prioritized Next Implementation Roadmap so
  alignment findings become implementation candidates instead of cleanup notes.

## Phase 18: Contract Authoring Guardrails

- [x] T180 Add a JSON Schema for workflow contract YAML/JSON authoring.
- [x] T181 Add `dogtap contract validate <path>` with text and JSON output.
- [x] T182 Validate missing names, empty check lists, duplicate check IDs,
  unsupported check types, unsupported sources, unsupported selector fields,
  unknown fields, and invalid regular expressions.
- [x] T183 Add `make contract-check` and CI coverage for bundled contract
  templates.
- [x] T184 Document schema and validation usage for local development and CI.

Gate:

- [x] G5 CI Contract authoring guardrails subset

Evidence note 2026-05-10:

- `schemas/workflow-contract.schema.json` documents the authoring shape for
  YAML/JSON workflow contracts.
- `dogtap contract validate` supports text and JSON output and keeps
  `dogtap diagnose -workflow-contract` behavior unchanged.
- `make contract-check` validates bundled templates in CI before smoke runs.

## Phase 19: Dashboard Contract Evidence Drilldown

- [x] T190 Show pass and fail workflow contract checks in the dashboard.
- [x] T191 Link matched event IDs from workflow checks to the event detail pane.
- [x] T192 Link trace IDs to a matching retained trace event when available.
- [x] T193 Add dashboard E2E coverage for workflow contract evidence links.
- [x] T194 Show evaluated selectors and closest observed alternatives for
  failing checks after contract results carry selector metadata.

Gate:

- [x] G4 Product Usability workflow contract evidence subset

## Phase 20: Diagnostics Root-Cause Classifier

- [x] T200 Add `assertions.rootCauses` for common missing telemetry causes.
- [x] T201 Classify Dogtap API reachability, no retained events, browser
  telemetry/replay/session, backend logs, traces, metrics, OTLP exporter,
  endpoint routing, and context propagation failures.
- [x] T202 Include observed evidence, next checks, and related assertion IDs in
  machine-readable diagnostics output.
- [x] T203 Render likely causes in `summary.md`.
- [x] T204 Document root-cause diagnostics in testing and agent triage docs.

Gate:

- [x] G5 CI Contract root-cause diagnostics subset

## Phase 21: Agent Gap Bridge Recipes

- [x] T210 Add an executable OpenTelemetry Collector filelog bridge recipe for
  JSON stdout/file logs into Dogtap OTLP logs.
- [x] T211 Add a Compose smoke stack and `make smoke-log-bridge` verification
  target.
- [x] T212 Document that Dogtap still does not tail logs or replace Datadog
  Agent container/Kubernetes log collection behavior.
- [ ] T213 Add DogStatsD-to-OTLP guidance or a fixture-backed bridge example
  without making Dogtap bind UDP `8125`.

Gate:

- [x] G8 Release Candidate agent gap bridge log subset
