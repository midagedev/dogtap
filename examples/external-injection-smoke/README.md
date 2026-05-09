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

Prerequisites:

- Docker with Compose v2
- `curl`
- `lsof` for local port-conflict checks

This proves:

- the frontend and backend can run without Dogtap
- the base Compose file has no Dogtap service or telemetry endpoint overrides
- the override adds Dogtap and injects standard Datadog/OTLP settings
- the frontend workflow sends RUM through the injected proxy
- the backend workflow sends logs, APM traces, and OTLP metrics through injected
  endpoints
- removing the override restores the base Compose shape
