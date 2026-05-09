# Minimal Datadog Node Tracer App

This app emits one Datadog APM span to the local Dogtap APM endpoint.

Setup:

```bash
npm --prefix testdata/apm-node install
scripts/fixtures/capture-apm-node-tracer.sh
```

The app uses `dd-trace` and points `DD_TRACE_AGENT_URL` at Dogtap. It should not
send telemetry to Datadog.

