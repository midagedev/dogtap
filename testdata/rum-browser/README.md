# Minimal Browser RUM App

This test app is used by `scripts/fixtures/capture-rum-browser.sh` to capture a
Datadog Browser RUM SDK payload without calling Datadog intake APIs.

Setup:

```bash
npm --prefix testdata/rum-browser install
scripts/fixtures/capture-rum-browser.sh
```

The local app server exposes `/datadog-intake-proxy` on the same origin as the
browser page and forwards that request to Dogtap. This avoids browser CORS
requirements while keeping telemetry local.

