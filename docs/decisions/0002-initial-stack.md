# Decision 0002: Prefer Go Backend with Embedded Web Dashboard

## Status

Accepted

## Context

Dogtap needs to receive HTTP, gRPC, compressed payloads, and potentially high-volume telemetry. It also needs to be easy to run as a single Docker image.

## Decision

Prefer a Go backend with an embedded React or TypeScript dashboard for the initial implementation.

Use:

- Go backend with one process and static dashboard embedding.
- React, TypeScript, and Vite for the dashboard build.
- In-memory bounded storage for local and CI MVP behavior.
- SQLite only after concrete restart-safe local, CI, isolated E2E, or
  dev-cluster retention needs require it. That trigger is now addressed by
  [Decision 0015](0015-sqlite-storage.md).

## Rationale

- Single binary deployment
- Strong HTTP and gRPC support
- Good fit for proxy and forwarder behavior
- Lower production runtime footprint
- Static dashboard assets can be embedded

## Alternatives

### TypeScript full stack

Faster UI iteration, easier for many frontend-heavy teams, but less ideal for a production-safe proxy.

### Rust

Excellent performance and safety, but slower iteration for this product stage.

### Python

Good for prototyping, weaker fit for production proxy and single-binary packaging.

## Review Trigger

Revisit after the protocol spike if APM or OTLP libraries make another stack materially better.

## Acceptance Notes

Accepted before runtime implementation to keep G0 scope locked. The decision can still be reopened after G1 fixture evidence if APM trace decoding or OTLP support proves materially better in another runtime.
