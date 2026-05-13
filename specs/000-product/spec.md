# Product Specification: Dogtap

## Status

Release-candidate baseline. The canonical behavior is the implemented Go
backend, embedded React dashboard, diagnostics API, and workflow contract
surface in this repository.

## Summary

Dogtap is a local and production-safe intake inspector and workflow contract
validator for Datadog-compatible and OpenTelemetry telemetry. It helps
engineering teams verify that instrumentation is useful, correlated, safe, and
cost-aware before and after deployment.

## Problem

Datadog instrumentation is difficult to validate in environments where telemetry is disabled, sampled, or expensive to inspect. Developers can write instrumentation code and still miss whether the emitted payload includes the context needed for incident response and customer support.

The most costly failures are not syntax errors. They are semantic failures:

- A frontend error is visible but cannot be tied to a user, workspace, case, trace, or backend error.
- Backend logs have `service` and `env` but no stable route or tenant context.
- Successful request logs create cost without adding diagnostic value.
- Query strings or tokens leak into logs.
- Logout does not clear RUM user or account context.
- Trace, log, and RUM payloads each contain useful fragments but cannot be joined.

## Users

### Backend engineer

Wants to verify APM traces, structured logs, service tags, route IDs, and error correlation for gateway and platform services.

### Frontend engineer

Wants to verify RUM user/account/workspace/case context and make sure logout or workspace switching does not leak stale context.

### QA engineer

Wants a deterministic local or CI target that confirms telemetry is emitted for key workflows without requiring Datadog access.

### SRE or platform owner

Wants a safe way to inspect telemetry shape, detect PII risks, and evaluate Datadog configuration changes before production rollout.

### Customer support engineer

Wants a debug bundle that explains what to search in Datadog for a user, workspace, case, or error report.

## Core Scenarios

### Scenario 1: Local RUM debugging

Given a frontend app configured to send RUM through Dogtap, when a user logs in, changes workspace, opens a design case, triggers an error, and logs out, Dogtap shows each RUM event with user, account, workspace, route, case, and session context.

Acceptance criteria:

- Events are grouped by session and view.
- Missing `user.id`, `account.id`, `workspace.id`, `route`, or `case_id` is flagged according to workflow rules.
- Logout events after `clearUser` and `clearAccount` show no stale user or account context.

### Scenario 2: Backend trace and log correlation

Given a backend service instrumented for Datadog APM and JSON logs, when an API
call fails, Dogtap shows the inbound request, outgoing trace spans, structured
logs, status code, route, service, env, version, and correlation IDs.

Acceptance criteria:

- `DD_ENV`, `DD_SERVICE`, and `DD_VERSION` are present.
- Logs and traces can be joined by trace ID or request correlation ID.
- Query strings are absent from log fields unless explicitly allowed.

### Scenario 3: CI telemetry contract

Given an automated test workflow, when the app exercises login, workspace,
subscription, design case, viewer, and export flows, Dogtap validates generic
expectations and optional workflow observability contracts, then exits with
machine-readable artifacts.

Acceptance criteria:

- CI can fail on missing required fields.
- CI can fail on PII patterns.
- CI can fail on a supplied workflow contract with
  `-fail-on-workflow-contract`.
- CI produces `summary.md`, `assertions.json`, optional
  `workflow-contracts.json`, and retained event evidence that links failures to
  received payloads.

### Scenario 4: Staging forward mode

Given Dogtap configured with a Datadog API key and site, when telemetry arrives, Dogtap validates, redacts local samples, forwards to Datadog, and records forwarding success or failure.

Acceptance criteria:

- Forwarding preserves Datadog behavior as much as possible.
- Forwarding failures are visible in Dogtap.
- Local storage is bounded by count and TTL.

### Scenario 5: Production tee mode

Given Dogtap deployed in production as an optional telemetry tee, when telemetry arrives, Dogtap forwards to Datadog first or receives a sampled copy, stores only redacted metadata, and never blocks application behavior.

Acceptance criteria:

- Production mode has explicit sampling.
- Raw payload persistence is disabled by default.
- Backpressure policy is visible and testable.
- Operators can disable Dogtap without redeploying applications where possible.

## Functional Requirements

### Intake

- FR-001: Accept browser RUM payloads through a Datadog RUM proxy-compatible endpoint.
- FR-002: Accept Datadog APM trace payloads on an agent-compatible HTTP port.
- FR-003: Accept Datadog logs HTTP intake payloads for JSON, text, gzip, and logplex-like inputs.
- FR-004: Accept OTLP HTTP and gRPC for traces, logs, and metrics.
- FR-005: Preserve request headers, endpoint path, query parameters, content encoding, and body size metadata.
- FR-006: Accept Grafana Faro SDK payloads on experimental native Faro intake
  endpoints for integration smoke validation only.

### Normalization

- FR-010: Normalize events into common fields: source, service, env, version, host, trace ID, span ID, session ID, user ID, account ID, workspace ID, case ID, route, status code, duration, timestamp.
- FR-011: Keep raw decoded payload accessible in local and CI modes.
- FR-012: Keep raw production payload disabled by default.

### Validation

- FR-020: Validate required unified service tags.
- FR-021: Validate configured workflow context requirements.
- FR-022: Detect PII and secret patterns in headers, query strings, log messages, tags, and RUM context.
- FR-023: Detect high-cardinality field risks.
- FR-024: Detect context leak across user logout and workspace switching.
- FR-025: Emit machine-readable validation results.

### Dashboard

- FR-030: Provide a dashboard for recent requests and validation failures.
- FR-031: Support filters by source, service, env, user, account, workspace, case, trace, route, and status.
- FR-032: Show correlation hints across RUM, logs, and traces.
- FR-033: Provide copyable Datadog search queries.
- FR-034: Provide a debug bundle export.
- FR-035: Provide source-specific inspectors for RUM Session Replay payloads,
  structured logs, and trace spans so developers can confirm telemetry was
  captured in a usable form without opening Datadog.
- FR-036: Show dashboard intake health by source and endpoint, including recent
  activity, validation failures, and missing-context signals.
- FR-037: Group browser telemetry by session so developers can inspect the
  frontend timeline across RUM, Faro, replay, logs, metrics, and correlated
  traces.
- FR-038: Show an interactive service map graph for retained telemetry, with
  selectable service nodes, trace-derived edges, upstream/downstream context,
  routes, and evidence links back to the matching retained events.

### Forwarding

- FR-040: Support disabled, forward, tee, and redact-only modes.
- FR-041: Forward to the configured Datadog site and endpoint.
- FR-042: Record forwarding status without storing secret values.
- FR-043: Support bounded retry where safe.

### Configuration

- FR-050: Provide config through environment variables and a YAML file.
- FR-051: Provide workflow-specific validators.
- FR-052: Provide production-safe defaults.
- FR-053: Provide a generic adoption kit for typical frontend and backend
  applications using standard Datadog and OpenTelemetry configuration surfaces,
  without requiring a Dogtap-specific application SDK.

### Adoption

- FR-060: Provide copyable Docker Compose, frontend RUM, backend OTLP, backend
  Datadog tracer, and logs examples that can be applied to an existing app and
  removed by restoring the original telemetry endpoints.
- FR-061: Provide a dashboard-accessible target summary for the active local
  Dogtap intake endpoints.
- FR-062: Provide a smoke verification path that proves RUM, logs, traces, and
  metrics can be received before a team wires a real application.
- FR-063: Provide a Datadog-preserving external injection adoption profile that
  documents how Dogtap can be enabled through sidecars, Compose overrides,
  Kubernetes patches, CI services, and runtime config without adding a
  Dogtap-specific application SDK.
- FR-064: Clearly distinguish supported endpoint redirection from unsupported
  Datadog Agent behaviors such as container log tailing, DogStatsD, and Agent
  integrations unless those behaviors receive fixture-backed support.
- FR-065: Provide an experimental Faro SDK compatibility smoke that runs the
  external-injection frontend at `/faro`, sends Faro SDK telemetry to Dogtap's
  native `/faro`, `/collect`, or `/collect/` intake, and can be verified with
  `make smoke-faro`.
- FR-066: Document that production-grade Faro adoption should prefer Grafana
  Alloy `faro.receiver` into OTLP until Dogtap has a fixture-backed production
  Faro compatibility contract.
- FR-067: Provide an agent-readable live diagnostics bundle for local dev,
  isolated E2E, and external app adoption runs. The bundle must capture Dogtap
  health, retained events, latest validation report, debug bundle, metrics,
  expectation assertions, and practical hints for missing logs, traces,
  metrics, RUM sessions, and session replay.
- FR-068: Expose live diagnostics through Dogtap's HTTP API so Docker Compose,
  local dev servers, isolated E2E stacks, and external agents can request JSON
  diagnostics or a downloadable archive without shelling into the Dogtap
  container. The API must reuse the same assertion and missing-signal hint
  semantics as `dogtap diagnose`.
- FR-069: Provide workflow observability contracts that assert whether a named
  frontend/backend workflow produced useful RUM, Session Replay, backend logs,
  traces, metrics, trace correlation, and privacy-safe context. Contract
  results must be available through diagnostics JSON, diagnostics archive files,
  the CLI, and the dashboard without changing existing diagnostics assertion
  semantics by default.
- FR-070: Provide a read-only Datadog API compatibility subset so agents and
  existing Datadog-oriented tools can query retained Dogtap telemetry through
  familiar logs, RUM, spans, and metric query paths without learning a
  Dogtap-specific search API.
- FR-071: Provide an opt-in SQLite event store for local, CI, isolated E2E,
  and dev-cluster runs that need restart-safe and queryable retained telemetry
  while preserving Dogtap's bounded TTL/count retention and redaction-before-
  persistence rules.
- FR-072: Keep the read-only Datadog API compatibility layer practical for
  agent debugging by supporting simple quoted phrases, quoted path-like
  attribute values, and quoted metric scope tag values while still rejecting
  full Datadog query-language parity as a non-goal.
- FR-073: Provide an automated public hygiene check that keeps company- or
  project-specific service names out of the public repository while allowing
  private adoption evidence to remain under ignored local paths.
- FR-074: Support a prefix-aware public base path contract for reverse-proxy
  deployments, including `PUBLIC_BASE_PATH`, `DOGTAP_PUBLIC_BASE_PATH`,
  `server.publicBasePath`, `X-Forwarded-Prefix`, prefixed dashboard assets,
  prefixed dashboard API calls, and continued unprefixed internal service
  communication.

## Non-Functional Requirements

- NFR-001: Local startup should complete within 3 seconds on a typical developer laptop after image pull.
- NFR-002: The Docker image should run as a single container for local use.
- NFR-003: Production forwarding overhead must be measurable and bounded.
- NFR-004: Redaction rules must be deterministic and covered by tests.
- NFR-005: The dashboard must not require Datadog credentials in local mode.
- NFR-006: The project must be usable without adopting any Dogtap-specific SDK.
- NFR-007: Local adoption instructions should fit a common frontend/backend
  app in under five deliberate configuration changes: start Dogtap, point
  browser RUM, point backend traces, point backend logs or OTLP logs, and verify.
- NFR-008: Workflow contracts must be event-backed, deterministic, and small
  enough for agents to inspect from `summary.md` and `workflow-contracts.json`
  without requiring Datadog credentials.
- NFR-009: Persistent storage must remain single-node and bounded by default;
  Dogtap must not require a network database for the local, CI, or dev-cluster
  inspection lane.

## Non-Goals

- Full Datadog UI replacement
- Long-term production telemetry warehouse
- Monitor evaluation engine
- Datadog billing estimator beyond heuristic hints
- Full private Datadog endpoint compatibility in MVP
- Full Datadog API query language or mutating API compatibility
- Full boolean Datadog search semantics, facets, formulas, rollups, or cursor
  pagination
- Production-grade native Grafana Faro collector parity
- Unbounded production telemetry warehouse or multi-tenant observability
  database

## Success Metrics

- A developer can configure a browser frontend app to send RUM to Dogtap in under 10 minutes.
- A backend engineer can verify local backend unified service tags.
- CI can fail on missing telemetry context with a readable report.
- A support-oriented debug bundle can identify the Datadog search query for a user, workspace, case, or trace.
- Production mode can be disabled or bypassed without application downtime.
- A developer can add Dogtap to a generic frontend plus backend development
  stack without introducing Dogtap-specific runtime code.
- A team that already uses Datadog can run a local or CI Dogtap lane by
  applying external endpoint overrides and can roll back by removing those
  overrides.

## Resolved Decisions

- Trace intake uses a minimal compatible receiver with fixture-backed Datadog
  tracer evidence; `dd-apm-test-agent` remains a reference path, not a runtime
  dependency.
- The React dashboard is embedded behind the Go API and rebuilt into
  `web/dist`.
- Local storage uses bounded memory by default, optional JSON file persistence
  for simple snapshots, and optional SQLite persistence for restart-safe local,
  CI, isolated E2E, and dev-cluster retained telemetry.
- Datadog private payload structures are normalized only where fixture-backed
  and otherwise preserved for inspection as decoded/raw local evidence.
- The repository is Apache-2.0 licensed for public OSS and team adoption.
