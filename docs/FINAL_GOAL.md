# Final Goal

## North Star

Dogtap should become the fastest way for a Datadog-using team to prove that its telemetry is useful, safe, correlated, and production-ready.

The final product should support three adoption levels:

1. Local inspector: replace Datadog during development and show the exact telemetry payloads emitted by applications.
2. CI contract validator: fail builds when required telemetry context, redaction, or correlation contracts regress.
3. Production-safe tap: inspect, sample, forward, or tee telemetry without becoming a critical application dependency.

## Final Product Statement

Dogtap is a Datadog-compatible telemetry intake inspector and validation gateway for RUM, logs, traces, and OTLP. It helps teams debug what they send to Datadog, enforce observability contracts, and safely operate telemetry forwarding paths.

## End State Capabilities

### Local and CI

- Start with one Docker command.
- Receive browser RUM, Datadog traces, logs, and OTLP.
- Show raw and normalized payloads in a dashboard.
- Validate required service tags and workflow context.
- Detect PII, tokens, query strings, stale user/account context, and high-cardinality risks.
- Replay fixtures and produce JSON/Markdown validation reports.
- Generate Datadog search queries and debug bundles.

### Staging

- Forward RUM/logs/traces/OTLP to Datadog where protocol support is stable.
- Store redacted samples and forwarding results.
- Compare "what Dogtap received" with "what Dogtap forwarded."
- Validate release candidate telemetry before monthly or low-frequency deployments.

### Production

- Run as an optional telemetry tap or carefully bounded forwarder.
- Store redacted metadata by default, not raw production payloads.
- Apply sampling, queue limits, TTL, and fail-open policies.
- Emit its own health and operational metrics.
- Be removable without changing application code where standard Datadog or OTLP configuration is used.

## Target User Outcome

An engineer should be able to answer these questions in minutes:

- Did a user action create the expected RUM event?
- Did logout clear user and account context?
- Can a frontend error be correlated with backend traces and logs?
- Are `env`, `service`, and `version` present everywhere?
- Are workspace, account, route, case, and trace identifiers present where needed?
- Did any payload leak a query string, token, cookie, email, or unsafe header?
- What Datadog query should support or engineering run for this issue?

## Strategic Constraints

- Dogtap must not become a Datadog clone.
- Dogtap must not require a custom application SDK.
- Dogtap must prefer standard Datadog and OpenTelemetry configuration.
- Dogtap must treat production raw telemetry as sensitive by default.
- Dogtap must be built in independently testable protocol and product slices so agent-driven development can run in parallel.

## Final Release Definition

Dogtap reaches the final goal when all of these are true:

- RUM, logs, APM, and OTLP intake adapters have fixture-backed compatibility tests.
- CI mode can validate a realistic multi-step product workflow.
- Dashboard supports stream, detail, validation, correlation, and debug bundle views.
- Forward or tee mode has documented and tested failure behavior.
- Production safety gates pass under load, malformed payloads, upstream Datadog failure, and storage pressure.
- A real service can adopt Dogtap locally and in CI without application code changes beyond standard Datadog or OTLP endpoint configuration.

