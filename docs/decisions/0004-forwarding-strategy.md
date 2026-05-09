# Decision 0004: Start Forwarding with RUM and Logs HTTP Payloads

## Status

Accepted

## Context

G6 requires Dogtap to record forwarding target, status, duration, failure reason, bounded retry behavior, and drop accounting without exposing Datadog API keys. The first forwarding surface also needs to stay separate from server wiring so parallel runtime work can integrate it deliberately.

Datadog RUM and logs both have HTTP intake paths that can be forwarded without Dogtap understanding the full private payload shape. APM forwarding is different: Datadog trace intake is agent-oriented, versioned by trace endpoint, may use msgpack, and needs a separate forwarding compatibility contract before Dogtap should claim pass-through behavior.

## Decision

Add a standalone Go package at `internal/forwarding` for Datadog-compatible RUM and logs HTTP payload forwarding.

The package will:

- Build Datadog target URLs for RUM and logs from a site, with an override for tests or explicit deployment targets.
- Forward the raw HTTP payload body with a strict allowlist of non-secret headers.
- Add the Datadog API key only to outbound logs requests through `DD-API-KEY`.
- Return `event.ForwardingResult` with mode, safe target, status, status code, duration, retry count, error class, and error message.
- Strip query strings and userinfo from persisted target values.
- Track payload, attempt, retry, success, failure, and drop counters.
- Bound retry attempts with a hard package limit so bad configuration cannot create infinite forwarding loops.

APM forwarding is explicitly deferred. Dogtap may inspect APM intake payloads,
but should not forward them through this package until a follow-up compatibility
decision defines the contract and safety evidence. Production deployments that
need trace forwarding should keep the Datadog Agent or OpenTelemetry Collector
on the primary forwarding path.

## Consequences

Positive:

- G6 forwarding safety primitives can be tested without touching server, config, or event ownership.
- API keys remain outbound-only and are not part of `ForwardingResult`.
- Retry and drop behavior is deterministic and easy to surface in later status APIs.

Tradeoffs:

- Server wiring, environment configuration, and dashboard visibility remain lead-owned integration work.
- RUM forwarding compatibility is intentionally limited to HTTP payload pass-through and does not claim full Browser SDK private protocol coverage.
- APM forwarding remains unsupported until forwarding compatibility and safety
  behavior are reviewed.
