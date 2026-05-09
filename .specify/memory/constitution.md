# Dogtap Constitution

## Article 1: Dogtap Is an Inspector, Not a Datadog Clone

Dogtap must stay focused on intake inspection, validation, safe forwarding, and developer feedback. It must not grow into a full observability backend unless a future spec explicitly changes the product strategy.

## Article 2: Production Safety Comes Before Feature Breadth

Any production-facing mode must be safe under failure. Forwarding must not block application behavior. Storage must be bounded. Backpressure behavior must be explicit. Redaction must run before persistence.

## Article 3: Payload Truth Over Pretty Screens

The dashboard must make raw and normalized telemetry understandable, but it must never hide the actual payload. Debugging requires access to received headers, endpoint path, decoded body, validation decisions, and forwarding result.

## Article 4: Vendor Compatibility Is Practical, Not Perfect

Dogtap should emulate stable and documented Datadog ingestion surfaces first. Private or unstable payload formats may be accepted best-effort, but must not become hard compatibility promises without fixtures.

## Article 5: OpenTelemetry Is the Escape Hatch

Where Datadog compatibility is unstable or too broad, Dogtap should prefer OTLP and OpenTelemetry Collector interoperability instead of inventing a proprietary middle layer.

## Article 6: No Raw Production Telemetry by Default

Production modes must store metadata, validation results, and redacted samples only. Full raw payload retention must be disabled by default and gated by short TTL, access controls, and explicit operator intent.

## Article 7: Tests Are Product Features

Dogtap exists to make telemetry testable. Contract tests, replay fixtures, redaction tests, and CI assertions are part of the core product, not secondary quality work.

## Article 8: Integration Should Be Reversible

Adopting Dogtap must not lock an application into Dogtap. Applications should be able to remove Dogtap by restoring standard Datadog endpoints or OTLP exporters.

## Article 9: Decisions Must Be Written Down

When a tradeoff affects compatibility, production safety, storage, cost, or user trust, record it under `docs/decisions/`.

## Article 10: Parallel Agents Need Gates

Agent-driven development is allowed and encouraged, but only with explicit ownership, reproducible verification, and gate status. Fast parallel work must not bypass protocol fixtures, validation evidence, or production safety requirements.
