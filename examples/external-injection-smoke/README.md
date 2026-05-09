# External Injection Smoke Stack

This example is a tiny frontend plus backend application used to verify
Dogtap's Datadog-preserving external injection contract.

The base Compose file starts only the app services:

```bash
docker compose -f examples/external-injection-smoke/compose.yaml up
```

The Dogtap override adds a Dogtap sidecar and injects standard Datadog/OTLP
endpoint settings into the existing services:

```bash
docker compose \
  -f examples/external-injection-smoke/compose.yaml \
  -f examples/external-injection-smoke/compose.override.dogtap.yaml \
  up --build
```

Run the automated check from the repository root:

```bash
make smoke-external-injection
```

Run the Faro SDK compatibility smoke from the repository root:

```bash
make smoke-faro
```

Prerequisites:

- Docker with Compose v2
- `curl`
- `lsof` for local port-conflict checks

This proves:

- the frontend and backend can run without Dogtap
- the base Compose file has no Dogtap service or telemetry endpoint overrides
- the override adds Dogtap and injects standard Datadog/OTLP settings
- the frontend workflow sends RUM and multipart Session Replay through the
  injected proxy
- the backend workflow sends logs, APM traces, and OTLP metrics through injected
  endpoints
- the profile includes one intentional missing-context RUM validation failure
- removing the override restores the base Compose shape

## Faro SDK Smoke

The frontend also exposes a `/faro` workflow for Grafana Faro SDK compatibility
smoke testing. The workflow loads the Faro Web SDK bundle, initializes it with a
collector URL that points at Dogtap, and sends a sanitized event, measurement,
and log with representative user, account, workspace, case, session, and route
context.

Dogtap's native Faro intake endpoints for this smoke are:

- `POST /faro`
- `POST /collect`
- `POST /collect/`

This native Faro path is experimental and intended for local/CI integration
smoke only. For production-grade Faro collection, route the Faro SDK through
Grafana Alloy `faro.receiver` and export OTLP to Dogtap for inspection.
