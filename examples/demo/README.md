# Dogtap Demo

This demo seeds a local Dogtap instance with one compact telemetry story:

- Browser RUM with user, account, workspace, case, session, and route context
- one intentionally failing RUM event for the validation inbox
- a RUM Session Replay payload with decoded replay frames
- a structured log correlated by trace ID and workflow context
- a Datadog APM trace with two spans across two services
- an OTLP metric sample for the same route

## Start Dogtap

```bash
docker compose up --build
```

Open the dashboard:

```text
http://localhost:8080
```

## Seed Demo Telemetry

In another shell:

```bash
make demo-seed
```

The dashboard should show the seeded service map, traffic, metrics, log viewer,
trace spans, failure inbox, and Session Replay payload timeline.

## Run The Visual Check

For maintainers, the visual check starts an isolated Dogtap server, seeds the
same demo telemetry, opens the dashboard with Playwright, and stores screenshots
under `web/test-results/`.

```bash
make demo-visual-check
```

Override ports when needed:

```bash
DOGTAP_DEMO_HTTP_PORT=19090 \
DOGTAP_DEMO_APM_PORT=19126 \
DOGTAP_DEMO_OTLP_HTTP_PORT=19318 \
make demo-visual-check
```
