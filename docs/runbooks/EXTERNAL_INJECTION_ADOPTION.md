# External Injection Adoption

This runbook describes the intended Dogtap adoption shape for teams that already
use Datadog or OpenTelemetry and want to preserve that usage as much as
possible.

Dogtap should be added from the outside:

- no Dogtap application SDK
- no instrumentation rewrite
- no production Datadog path removal during local or CI rollout
- endpoint changes controlled by environment, Compose override, Kubernetes
  patch, or CI service configuration

## Target Shape

```text
existing app code
  |
  +-- existing Datadog Browser RUM SDK -> Dogtap RUM proxy
  +-- existing Datadog tracer ----------> Dogtap APM-compatible intake
  +-- existing OTLP exporter -----------> Dogtap OTLP intake
  +-- existing log sender/collector ----> Dogtap logs or OTLP logs intake
```

The production lane can stay unchanged:

```text
production app -> Datadog Agent or OpenTelemetry Collector -> Datadog
```

Dogtap's local and CI lane should be removable by deleting the sidecar and
restoring the original endpoints.

## Adoption Profiles

| Profile | Use when | How it works | Current fit |
| --- | --- | --- | --- |
| `local-direct` | Local dev, preview apps, CI services | Add Dogtap as a sidecar/service and override Datadog/OTLP endpoints to Dogtap | Best current path |
| `collector-bridge` | Logs or metrics currently depend on DD Agent, Vector, Fluent Bit, or OTel Collector | Keep the collector behavior and export a copy to Dogtap over HTTP/OTLP | Recommended next hardening path |
| `production-tee` | Limited staging or production diagnostics | Keep Datadog primary, sample or tee selected payloads through Dogtap with bounded retention | Requires explicit safety review |

## Docker Compose Injection

Use `examples/adoption-kit/compose.override.template.yaml` as a starting point
inside an application repository:

```bash
mkdir -p .dogtap
cp ../dogtap/examples/adoption-kit/compose.override.template.yaml .dogtap/compose.override.dogtap.yaml
cp ../dogtap/examples/adoption-kit/datadog-preserve.env .dogtap/datadog-preserve.env
```

Then edit the placeholder service name in the override file and run:

```bash
docker compose -f compose.yaml -f .dogtap/compose.override.dogtap.yaml up --build
```

For backend containers in the same Compose project, use `dogtap` as the host.
For host processes outside Docker, use `localhost`.

Dogtap includes a small frontend/backend Compose smoke that exercises this
contract:

```bash
make smoke-external-injection
```

The smoke starts a base frontend/backend stack without Dogtap, then starts the
same stack with a Dogtap override that injects standard Datadog and OTLP
endpoints. It verifies RUM, logs, APM traces, and OTLP metrics arrive, then
proves rollback by omitting the override.

## Kubernetes Sidecar Injection

Use `examples/adoption-kit/kubernetes/deployment-sidecar.template.yaml` as a
sidecar patch template.

In a same-pod sidecar, backend SDKs should target loopback:

```bash
DD_TRACE_AGENT_URL=http://127.0.0.1:8126
OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4318
```

Browser RUM is different because the browser cannot reach a pod-local loopback
address. Route RUM through a service, ingress, local port-forward, or app
reverse proxy that exposes:

```text
http://<reachable-dogtap-host>/datadog-intake-proxy
```

## Signal-Specific Guidance

### Browser RUM And Session Replay

Best case: the app already externalizes the Datadog RUM `proxy` option through
runtime config.

```bash
DATADOG_RUM_PROXY_URL=http://localhost:8080/datadog-intake-proxy
```

Then pass that value into the existing Datadog RUM initialization.

If the app hardcodes RUM initialization, make one preparatory change to read the
proxy from runtime config. After that, Dogtap can be enabled or removed without
more application code changes.

### Datadog APM Traces

Prefer `DD_TRACE_AGENT_URL` when the tracer supports it because it is explicit:

```bash
DD_TRACE_AGENT_URL=http://dogtap:8126
```

The host/port form remains useful for older or generic examples:

```bash
DD_AGENT_HOST=dogtap
DD_TRACE_AGENT_PORT=8126
```

Keep normal unified service tags:

```bash
DD_SERVICE=your-backend
DD_ENV=local
DD_VERSION=local
```

### Logs

If logs already go through OTLP, point OTLP logs at Dogtap.

If logs are sent through a Datadog logs HTTP sender, send them to:

```text
http://dogtap:8080/api/v2/logs
```

If logs currently arrive in Datadog only because the Datadog Agent tails
container stdout, Kubernetes log files, or application log files, Dogtap does
not yet replace that Agent behavior. Use a collector bridge that tails the logs
and exports OTLP logs or Datadog logs HTTP payloads to Dogtap.

### Metrics

Dogtap currently supports OTLP metrics. Prefer this for local inspection:

```bash
OTEL_METRICS_EXPORTER=otlp
OTEL_EXPORTER_OTLP_METRICS_ENDPOINT=http://dogtap:4318/v1/metrics
```

DogStatsD is not currently supported. If a service emits only DogStatsD metrics,
keep Datadog Agent in the production lane and add an OTLP metrics path for
Dogtap validation where practical.

## Definition Of Done For A Real Adoption

A real app adoption profile should prove:

- RUM appears with service, env, version, session, route, and user/workflow
  context.
- Session Replay upload payloads appear in the replay timeline.
- Backend spans appear with service, env, version, trace ID, span ID, and route
  or resource names.
- Logs appear with service, env, message, status, and trace correlation fields.
- Metrics appear as OTLP metric samples.
- The service map shows at least one backend service relationship or an
  explicitly documented reason why the fixture is single-service.
- Removing Dogtap is a configuration-only rollback.

## Known Gaps

- Dogtap is not a full Datadog Agent replacement.
- Dogtap does not tail container stdout or arbitrary log files by itself.
- Dogtap does not receive DogStatsD metrics.
- Dogtap does not run Datadog Agent integrations or Autodiscovery checks.
- RUM external injection still requires the app to expose the Datadog RUM
  `proxy` option.

These gaps are intentional until there are fixtures, tests, and production
safety gates for each behavior.
