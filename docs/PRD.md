# Product Requirements Document

## Product

Dogtap

## Date

2026-05-08

## Owner

Personal project, designed for later adoption by teams using Datadog.

## Background

Datadog is powerful but hard to validate before production. Many teams disable RUM or log collection in development to control cost and noise. This means instrumentation regressions are found late, often after customer support receives reports.

Dogtap provides a local and CI-friendly Datadog intake target that makes telemetry payloads visible and testable. It can later become a production-safe proxy or tee for validating telemetry before it reaches Datadog.

## Goals

- Help developers see actual telemetry payloads locally.
- Help teams validate telemetry context in CI.
- Help teams detect PII and token leakage before Datadog ingestion.
- Help customer support debug issues by generating Datadog search hints.
- Provide a path to safe staging and production forwarding.

## Target Workflows

1. Login and logout
2. Signup
3. Workspace selection and switching
4. Subscription and payment
5. Design case creation
6. Viewer session
7. Export
8. Gateway routing
9. Platform API errors

## User Stories

### RUM context

As a frontend engineer, I want to see each RUM event's user, account, workspace, route, and case context so that I can verify Datadog will be useful for CS and incident triage.

### APM context

As a backend engineer, I want to verify that gateway and platform traces include env, service, version, route, status, and correlation IDs so that errors can be traced quickly.

### CI contract

As a QA or platform engineer, I want a headless validation mode so that telemetry regressions fail before release.

### Debug bundle

As a support engineer, I want a bundle for a user, workspace, case, or trace so that I know exactly what to search in Datadog.

### Production safety

As an SRE, I want Dogtap production mode to forward safely without storing raw telemetry by default so that the tool does not become a new compliance or reliability risk.

## Requirements

See `specs/000-product/spec.md` for the canonical requirements.

## MVP Scope

- RUM and logs local intake
- APM trace intake
- Basic OTLP support
- Dashboard list/detail
- Required context validation
- PII and query-string risk detection
- CI report mode
- Docker image

## Out of Scope for MVP

- Full Datadog monitor/query language
- Full-fidelity Datadog Session Replay rendering
- Profiling
- Long-term production storage
- Billing-grade cost calculation

## Launch Criteria

- Dogtap can be started with one Docker command.
- A sample frontend sends RUM to Dogtap.
- A sample Spring service sends traces to Dogtap.
- Missing required context is flagged in dashboard and CI mode.
- PII redaction tests pass.
- Documentation explains production limitations clearly.
