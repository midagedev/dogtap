# Changelog

All notable changes to Dogtap will be documented in this file.

## Unreleased

### Added

- Go backend with embedded React dashboard.
- RUM, logs, Datadog APM HTTP, OTLP HTTP, and OTLP gRPC intake.
- RUM Session Replay payload timeline viewer.
- Structured log viewer, trace/span viewer, service map, traffic summary, and
  metric sample viewer.
- Required context validation, PII/secret detection, redaction, sampling, and
  bounded retention controls.
- Fixture replay command with JSON and Markdown reports.
- Generic adoption kit under `examples/adoption-kit/`.
- Generic adoption smoke script through `make smoke-adoption`.
- Seeded dashboard demo and Playwright visual check through `make demo-visual-check`.
- Release workflow for GitHub Release binaries and GHCR container images.
- Build metadata in `dogtap version`.
- Public support matrix and release candidate runbook.
- Workflow observability contracts through diagnostics API, diagnostics archive,
  `dogtap diagnose -workflow-contract`, built-in dashboard readiness checks,
  and example contracts under `configs/contracts/`.
- Workflow contract templates for login, case-open, checkout, and report-export,
  plus a GitHub Actions recipe under `examples/github-actions/`.
- Spec/docs/code alignment updates for diagnostics, Faro smoke, metrics, and
  workflow contracts, plus `make doc-check` to prevent the most important drift.
- Workflow contract authoring guardrails with
  `dogtap contract validate <path>`, `make contract-check`, and a JSON Schema
  at `schemas/workflow-contract.schema.json`.
- Dashboard workflow contract evidence now shows pass and fail checks with
  event and trace evidence links, plus failed-check selector criteria and
  closest retained alternatives.
- Diagnostics assertions now include root-cause classifications with evidence
  and next checks for missing telemetry, including OTLP exporter and endpoint
  routing failures.
- OpenTelemetry Collector filelog bridge recipe, Compose smoke stack, and
  `make smoke-log-bridge` for Agent-style stdout/file log adoption gaps.
- OpenTelemetry Collector StatsD bridge recipe, Compose smoke stack, and
  `make smoke-statsd-bridge` for DogStatsD-style metric adoption gaps.
- GitHub Actions now runs the filelog and StatsD bridge smokes and uploads
  bridge diagnostics artifacts on smoke failures.
- Public deployment examples for Helm sidecar, Helm companion-service, and
  ECS/Fargate trial shapes, plus `make deployment-check` safety-marker
  validation.
- Read-only Datadog API compatibility for retained logs, RUM, spans, and metric
  query debugging through Datadog-shaped paths.
- Opt-in bounded SQLite storage for restart-safe local, CI, isolated E2E, and
  dev-cluster retained telemetry.
- Dashboard structured log drilldown and retained metric snapshot charts for
  faster log/metric inspection.
- Datadog-compatible structured log aliases, metric point tag filtering,
  trace-ID alias matching, and Dogtap event IDs for agent debugging.
- EKS dev-cluster Kustomize overlay and smoke runbook with private networking,
  SQLite PVC retention, non-root security context, and rollback steps.
- Datadog-compatible query hardening for quoted log phrases, quoted path-like
  attribute values, and quoted metric scope tags.
- Public hygiene check through `make public-hygiene-check` to keep
  company-specific adoption terms out of the public repository.
- Subscription workflow contract starter and documented workflow contract
  starter pack for common frontend/backend flows.
- Interactive retained-telemetry service map with selectable nodes,
  trace-derived edges, upstream/downstream context, route summaries, and
  event evidence links.
- Prefix-aware public deployment support through `PUBLIC_BASE_PATH`,
  `DOGTAP_PUBLIC_BASE_PATH`, `server.publicBasePath`, `X-Forwarded-Prefix`,
  prefixed dashboard assets, and prefixed dashboard API calls.

### Notes

- Official container images and binary archives are published from version tags;
  stable tags also update the `latest` image tag.
- APM forwarding is deferred; RUM/log forwarding and safety accounting are
  implemented for the current scope.
- Supported endpoints and current limitations are maintained in
  `docs/SUPPORT_MATRIX.md`.
- G8 release-candidate evidence passed for the current public scope; later
  compatibility work should continue to add fixture-backed workflow contracts.
