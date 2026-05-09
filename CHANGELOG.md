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

### Notes

- No official container image or binary release is published yet.
- APM forwarding is deferred; RUM/log forwarding and safety accounting are
  implemented for the current scope.
