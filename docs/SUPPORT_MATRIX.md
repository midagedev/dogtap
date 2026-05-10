# Support Matrix

Dogtap targets practical local and CI inspection. This matrix describes what is
currently supported, what is partial, and what should remain outside the first
public release.

## Runtime Modes

| Mode | Current level | Intended use | Notes |
| --- | --- | --- | --- |
| `local` | Supported | Local payload inspection and demo workflows | Raw payloads are visible by default for debugging. |
| `ci` | Supported | Fixture replay and validation reports | Use `dogtap replay`; the dashboard is not required. |
| live diagnostics | Supported | Local dev, isolated E2E, Docker Compose, and external app triage | Use `POST /api/diagnostics` for JSON, `POST /api/diagnostics/archive` for a zip bundle, or `dogtap diagnose` for a host-side directory. |
| workflow contracts | Supported | Assert that a named app workflow emitted useful RUM, replay, logs, traces, metrics, and correlation evidence | Use YAML/JSON contracts through diagnostics API or `dogtap diagnose -workflow-contract`. |
| `forward` | Partial | Bounded RUM/log forwarding experiments | APM forwarding is deferred. |
| `tee` | Experimental | Limited production diagnostic tap | Requires explicit sampling, retention, and fail-open review. |
| `redact-only` | Experimental | Policy enforcement before forwarding | Treat as a controlled rollout mode, not a default path. |

## Intake Surfaces

| Surface | Endpoint / port | Current level | Verification |
| --- | --- | --- | --- |
| Browser RUM proxy | `/datadog-intake-proxy` | Supported for local and CI inspection | Browser RUM SDK fixture, replay tests, dashboard E2E |
| RUM Session Replay | `/api/v2/replay`, proxy `ddforward=/api/v2/replay` | Partial | DOM playback for decoded rrweb full snapshot records, with timeline fallback |
| Grafana Faro SDK | `/faro`, `/collect`, `/collect/` | Experimental smoke only | Used by the external-injection frontend `/faro` workflow and `make smoke-faro`; not a production-grade Faro receiver contract |
| Datadog logs HTTP | `/api/v2/logs`, `/v1/input` | Supported for local inspection | JSON, text, and gzip fixtures |
| Datadog APM traces | `:8126`, `/v0.3/traces`, `/v0.4/traces`, `/v0.5/traces` | Supported for intake and span inspection | Datadog tracer fixture-backed; forwarding deferred |
| OTLP HTTP traces/logs/metrics | `:4318`, `/v1/traces`, `/v1/logs`, `/v1/metrics` | Supported for local inspection | OpenTelemetry SDK fixture-backed |
| OTLP gRPC traces/logs/metrics | `:4317` | Supported for local inspection | OpenTelemetry SDK fixture-backed |
| DogStatsD | none | Not supported | Out of scope for first public release |
| Profiling | none | Not supported | Out of scope for first public release |

## External Injection Surfaces

| Existing usage | Current Dogtap fit | Notes |
| --- | --- | --- |
| Datadog Browser RUM SDK with configurable `proxy` | Supported | Use `/datadog-intake-proxy`; Session Replay arrives through the same proxy path. |
| Datadog Browser RUM SDK with hardcoded init | Requires one preparatory app change | Make the `proxy` value runtime-configurable, then Dogtap enable/disable is external. |
| Datadog backend tracer | Supported for local/CI intake | Prefer `DD_TRACE_AGENT_URL`; host/port env works for common tracer setups. |
| Datadog trace/log correlation | Supported when logs reach Dogtap | Keep `DD_LOGS_INJECTION=true`; Dogtap still needs a log input path. |
| Grafana Faro Web SDK | Experimental smoke only | Point the SDK collector URL at Dogtap `/faro`, `/collect`, or `/collect/` for integration smoke. For production-grade Faro, route through Grafana Alloy `faro.receiver` and export OTLP to Dogtap. |
| DD Agent stdout/file log tailing | Not Dogtap-native | Use a collector/log-forwarder bridge to Dogtap logs or OTLP logs; `make smoke-log-bridge` verifies the OTel filelog recipe. |
| DogStatsD metrics | Not Dogtap-native | Keep Datadog Agent for production DogStatsD; `make smoke-statsd-bridge` verifies a Collector StatsD-to-OTLP metrics bridge for local/CI inspection. |
| OTel Collector sidecar/gateway | Supported as a bridge pattern | Send OTLP traces/logs/metrics to Dogtap in local/CI or sampled tee modes. |

## Forwarding Surfaces

| Source | Current level | Notes |
| --- | --- | --- |
| Browser RUM | Supported for bounded forwarding experiments | Preserves safe relative `ddforward` path/query for `/api/v2/rum` and strips sensitive inbound headers. |
| RUM Session Replay | Supported for bounded forwarding experiments | Preserves safe relative `ddforward` path/query for `/api/v2/replay`; dashboard renders decoded rrweb records when available. |
| Datadog logs HTTP | Supported for bounded forwarding experiments | Adds the outbound Datadog API key only when forwarding logs. |
| Datadog APM traces | Deferred | Intake and dashboard inspection are supported; forwarding needs a separate compatibility contract decision. |
| OTLP traces/logs/metrics | Deferred | Keep an OpenTelemetry Collector on the production forwarding path. |
| Native Faro forwarding | Not supported | Native Faro intake is for smoke inspection only; use Alloy `faro.receiver` plus OTLP for production routing. |

## Dashboard Capabilities

| Capability | Current level | Notes |
| --- | --- | --- |
| Stream and detail views | Supported | Shows endpoint, normalized context, validation, and payload. |
| Validation failure inbox | Supported | Filterable by failing rule ID. |
| Correlation hints | Supported | Uses trace, user, workspace, and case identifiers from recent events. |
| Service map | Partial | Uses span parent/child edges and bounded trace-correlation fallback. |
| Log viewer | Supported | Shows decoded log entries and trace IDs. |
| Trace/span viewer | Supported | Shows decoded spans where available. |
| Metric viewer | Supported | Shows OTLP metric samples decoded from received payloads. |
| Session Replay viewer | Partial | Renders decoded rrweb records in an iframe and falls back to payload timeline/metadata when DOM snapshots are unavailable. |
| Diagnostics API | Supported | `POST /api/diagnostics` and `/api/diagnostics/archive` expose health, retained events, validation report, debug bundle, metrics, assertions, missing-signal hints, and root-cause classifications. |
| Datadog search hints | Best effort | Query field names should be checked against the team's Datadog conventions. |

## Release Evidence Commands

Run these before cutting a public tag:

```bash
go test ./...
npm --prefix web run build
make shell-check
make contract-check
make smoke-adoption
make smoke-log-bridge
make smoke-statsd-bridge
make smoke-faro
make demo-visual-check
go run ./cmd/dogtap replay \
  -config configs/generic-local.yaml \
  -format markdown \
  fixtures/rum/login.json \
  fixtures/logs/json-log.json \
  fixtures/apm/trace.json \
  fixtures/otlp/traces.json
```

G8 release-candidate evidence is recorded under `docs/gates/`, including the
sanitized adoption profile in `docs/gates/G8_SANITIZED_ADOPTION_PROFILE.md`.
