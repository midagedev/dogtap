# Agent Telemetry Triage

Dogtap diagnostics are designed for local dev servers, isolated E2E stacks, and
external app adoption runs where an agent needs to answer:

- Did Dogtap receive telemetry from the app?
- Which source, endpoint, service, session, trace, route, and metric names were
  observed?
- Which expected signals are missing?
- Which configuration surface is the most likely cause?

## Capture A Live Bundle

Start Dogtap, run the app workflow, then capture diagnostics from the Dogtap HTTP
port:

```bash
go run ./cmd/dogtap diagnose \
  -base-url http://127.0.0.1:8080 \
  -output .dogtap/diagnostics/local-dev \
  -expect-non-empty \
  -expect-source rum,logs,apm,otlp \
  -expect-payload-kind replay,metric \
  -expect-service web-frontend,api-service \
  -expect-session session-123 \
  -expect-metric http.server.request.duration
```

For private or project-specific adoption work, write artifacts under an ignored
path such as `.private/adoption/` or the target project's own ignored output
directory. Do not commit raw diagnostics from a real app.

## Output Files

`dogtap diagnose` writes one directory with:

| File | Purpose |
| --- | --- |
| `summary.md` | Human-readable status, observed dimensions, and failing hints. |
| `assertions.json` | Machine-readable pass/fail checks for agents and CI. |
| `events.json` | Raw retained event envelopes from `/api/events`. |
| `report.json` | Latest validation report from `/api/reports/latest`. |
| `debug-bundle.json` | Filtered debug bundle with Datadog query hints. |
| `metrics.txt` | Dogtap self-observability metrics. |
| `healthz.json`, `readyz.json` | Dogtap process health probes. |
| `manifest.json` | Index of files and request status. |

Smoke scripts may also copy `dogtap.log`, `frontend.log`, or Compose logs into
the same directory.

## Isolated E2E Pattern

When another repository starts an isolated environment, keep Dogtap diagnostics
outside that public repository unless the contents are sanitized.

Example shape:

```bash
DOGTAP_ARTIFACT_DIR="$PWD/.tmp/dogtap-diagnostics" \
  go run /path/to/dogtap/cmd/dogtap diagnose \
    -base-url http://127.0.0.1:18080 \
    -expect-non-empty \
    -expect-source rum,logs,otlp \
    -expect-payload-kind replay,metric \
    -expect-service frontend-app,backend-api,gateway-api
```

Use `-filter-*` flags to narrow the debug bundle for a failing workflow:

```bash
go run ./cmd/dogtap diagnose \
  -base-url http://127.0.0.1:18080 \
  -output .private/adoption/dogtap-last-run \
  -filter-session-id "$SESSION_ID" \
  -filter-service backend-api \
  -expect-session "$SESSION_ID" \
  -expect-source rum,logs,otlp
```

## Local Dev Server Pattern

For a manually running frontend/backend:

1. Start Dogtap.
2. Point frontend RUM or Faro collector config at Dogtap.
3. Point backend traces/logs/metrics at Dogtap or an OTLP bridge.
4. Exercise one workflow in the browser or API client.
5. Run `dogtap diagnose` with expectations for that workflow.
6. Open `summary.md` first, then inspect `events.json` and `debug-bundle.json`.

## Reading Failures

Common failing checks and likely causes:

| Missing check | First places to inspect |
| --- | --- |
| `source:rum` | Frontend runtime config, browser network tab, RUM proxy URL, CORS/proxy path. |
| `payload-kind:replay` | Session replay enabled flag, replay sample rate, `/api/v2/replay` proxy routing, user consent/sampling. |
| `source:logs` | Dogtap does not tail containers; use HTTP logs, OTLP logs, or a log-forwarder bridge. |
| `source:apm` | `DD_TRACE_AGENT_URL`, `DD_AGENT_HOST`, `DD_TRACE_AGENT_PORT`, tracer startup order. |
| `source:otlp` | `OTEL_EXPORTER_OTLP_ENDPOINT`, protocol, port `4317` vs `4318`, exporter enablement. |
| `payload-kind:metric` | OTLP metrics exporter, export interval, endpoint. DogStatsD is not accepted directly. |
| `service:<name>` | `DD_SERVICE`, `DD_ENV`, `DD_VERSION`, `OTEL_SERVICE_NAME`, resource attributes. |
| `session:<id>` | Browser workflow did not run, SDK session sampling, SDK init order, RUM/Faro session context. |
| `trace:<id>` | Trace exporter routing or trace/log correlation propagation. |

## CI Artifacts

The repository CI uploads:

- `dogtap-smoke-diagnostics` for generic, external-injection, and Faro smoke.
- `dogtap-demo-diagnostics` for the seeded live dashboard demo.
- `dogtap-demo-screenshots` for desktop and mobile dashboard screenshots.

Agents should download diagnostics first when a check fails, then use
`assertions.json` to identify the missing signal before reading full logs.
