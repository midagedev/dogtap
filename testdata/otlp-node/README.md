# Minimal OpenTelemetry Node SDK App

This app emits spans to Dogtap through OTLP HTTP and gRPC.

Setup:

```bash
npm --prefix testdata/otlp-node install
scripts/fixtures/capture-otlp-node-sdk.sh
```

The exporters point to local Dogtap endpoints and should not send telemetry to
external collectors.

