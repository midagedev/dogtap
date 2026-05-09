# Research Notes

## Existing Projects

### Datadog Agent

The Datadog Agent is open source and includes multiple agents such as the main agent, tracing agent, process agent, and security agent.

Use as reference, not as a dependency for MVP.

Reference:

- https://opensource.datadoghq.com/projects/agent-integrations-tracers/
- https://github.com/DataDog/datadog-agent

### dd-apm-test-agent

Datadog provides `dd-apm-test-agent`, a test agent for APM client libraries. It emulates APM endpoints and includes an optional Web UI.

Use as a close reference for APM intake behavior. Decide later whether Dogtap should embed, wrap, or reimplement compatible behavior.

Reference:

- https://github.com/DataDog/dd-apm-test-agent

### OpenTelemetry Collector

OpenTelemetry Collector already solves generic telemetry collection, routing, and export. Dogtap should interoperate with OTel instead of rebuilding collector features.

Reference:

- https://opentelemetry.io/docs/collector/
- https://docs.datadoghq.com/opentelemetry/setup/otlp_ingest/

### Open-source Datadog alternatives

SigNoz, OpenObserve, Grafana, and similar systems are observability backends. Dogtap is not competing with them. Dogtap focuses on compatibility inspection and validation before telemetry reaches Datadog or another backend.

## Official Datadog Surfaces to Support First

### RUM proxy

Datadog Browser RUM supports a proxy setting that forwards browser RUM data through a custom endpoint.

Reference:

- https://docs.datadoghq.com/real_user_monitoring/guide/proxy-rum-data/

### APM Agent API

Datadog tracing data is sent to the local Agent through HTTP, commonly on port `8126`.

Reference:

- https://docs.datadoghq.com/tracing/guide/send_traces_to_agent_by_api/

### Logs HTTP intake

Datadog logs can be submitted through HTTP intake endpoints. Dogtap should support common content types and encodings used by local tests and forwarders.

Reference:

- https://docs.datadoghq.com/api/latest/logs/

### OTLP

Datadog supports OTLP ingestion, and OTLP should be treated as the stable cross-vendor path.

Reference:

- https://docs.datadoghq.com/opentelemetry/setup/otlp_ingest/

## Product Insight

The gap is not "open-source Datadog." That already exists in several forms as alternative backends. The gap is "Datadog intake debugging and contract validation" for teams that want to keep Datadog but make it safer, cheaper, and easier to verify.

## G1 Fixture Evidence Notes

### 2026-05-08 fixture evidence harness

Status: G1 passed for the first product slice on 2026-05-08.

Added a local capture harness under `scripts/fixtures/` with documentation in
`docs/fixtures/G1_FIXTURE_EVIDENCE.md`.

Original bundled fixtures under `fixtures/` are explicitly marked as smoke
fixtures with adjacent `*.meta.json` files. Real reviewed/sanitized evidence
fixtures have now been promoted for RUM, APM, and OTLP.

Capture paths:

- Browser RUM: `scripts/fixtures/capture-rum-browser.sh` with
  `testdata/rum-browser`, using `@datadog/browser-rum` locally and forwarding to
  Dogtap through a same-origin local proxy.
- Logs: `scripts/fixtures/capture-logs.sh`, posting JSON, text, and gzip
  payloads to local Dogtap.
- APM: `scripts/fixtures/capture-apm-node-tracer.sh`, using a local `dd-trace`
  Node sample. `scripts/fixtures/capture-apm-dd-test-agent-reference.sh`
  documents the reference comparison path for `dd-apm-test-agent`.
- OTLP: `scripts/fixtures/capture-otlp-node-sdk.sh`, using OpenTelemetry Node
  SDK exporters for OTLP HTTP and gRPC.

Expected run-local artifacts are written under `testdata/g1-evidence/latest/`
and ignored by default to avoid accidentally committing raw capture data.

Promoted evidence:

- RUM: `fixtures/rum/browser-rum-sdk-batch.json`, captured from
  `@datadog/browser-rum` 6.0.0.
- APM: `fixtures/apm/node-tracer-*.json`, captured from Datadog `dd-trace`
  5.44.0. Java/Spring and `dd-apm-test-agent` comparison are deferred by ADR
  0005.
- OTLP: `fixtures/otlp/otel-node-*-traces.json`, captured from OpenTelemetry
  Node SDK HTTP and gRPC exporters.
