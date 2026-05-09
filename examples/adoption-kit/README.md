# Dogtap Generic Adoption Kit

This folder contains copyable templates for applying Dogtap to a normal
frontend plus backend development stack.

The intended path is:

1. Start Dogtap as a sidecar or standalone process.
2. Point browser RUM at the Dogtap RUM proxy.
3. Point backend telemetry at Dogtap through OTLP or Datadog tracer settings.
4. Send logs either through OTLP logs or Datadog logs HTTP intake.
5. Open the Dogtap dashboard and verify that events correlate by service,
   route, trace ID, user, workspace, or case context.

No Dogtap application SDK is required. Removing Dogtap should be a config-only
change that restores the original Datadog or OTLP endpoints.

## Files

| File | Use |
| --- | --- |
| `compose.dogtap.yaml` | Dogtap sidecar for Docker Compose projects |
| `dogtap.local.yaml` | Local persistent Dogtap config |
| `backend-otel-http.env` | Backend OTLP HTTP defaults |
| `backend-otel-grpc.env` | Backend OTLP gRPC defaults |
| `backend-datadog-tracer.env` | Existing Datadog tracer defaults |
| `frontend-rum.md` | Browser RUM proxy snippets |
| `logs-http.md` | Logs HTTP intake examples |

## Compose Use

From an application repository that has this file copied under `.dogtap/`:

```bash
DOGTAP_REPO=../dogtap docker compose -f .dogtap/compose.dogtap.yaml up --build
```

If Dogtap is published as an image in your environment, replace the `build`
section in `compose.dogtap.yaml` with the image name and run without
`DOGTAP_REPO`.

For backend containers in the same Compose project, use `dogtap` as the host.
For host processes outside Docker, use `localhost`.
