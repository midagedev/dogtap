# Decision 0003: Runtime Module Layout

## Status

Accepted

## Context

Dogtap needs protocol adapters, validation, storage, API, dashboard, and CI reporting to evolve in parallel without coupling each adapter directly to UI or report code.

## Decision

Use a Go module with these ownership boundaries:

- `cmd/dogtap`: process entrypoint and subcommands
- `internal/config`: YAML and environment configuration
- `internal/event`: shared event envelope and normalized telemetry models
- `internal/store`: bounded event store contracts and memory implementation
- `internal/validation`: deterministic validation rules
- `internal/intake`: request capture, decoding, normalization, and protocol handlers
- `internal/server`: HTTP API, dashboard serving, and process wiring
- `internal/report`: CI and replay report generation
- `web`: React dashboard and embedded build output

## Consequences

Protocol adapters write only event envelopes through the store interface. Dashboard and CI consume the same event and validation models. Production safety rules can be enforced before persistence without UI-specific dependencies.
