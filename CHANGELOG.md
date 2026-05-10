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

### Notes

- Official container images and binary archives are published from version tags;
  stable tags also update the `latest` image tag.
- APM forwarding is deferred; RUM/log forwarding and safety accounting are
  implemented for the current scope.
- Supported endpoints and current limitations are maintained in
  `docs/SUPPORT_MATRIX.md`.
- G8 release-candidate evidence passed for the current public scope; later
  compatibility work should continue to add fixture-backed workflow contracts.
