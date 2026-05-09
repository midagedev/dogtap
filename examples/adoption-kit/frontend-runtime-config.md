# Frontend Runtime RUM Proxy Config

Dogtap works best when the app already reads Datadog RUM settings from runtime
configuration. The application image can stay unchanged while local, CI, or
preview environments inject only the proxy URL.

## Required App Shape

The app should resolve a runtime value such as:

```text
DATADOG_RUM_PROXY_URL
```

and pass it to the existing Datadog Browser RUM initialization only when set.

```ts
const rumProxy = window.__RUNTIME_CONFIG__?.DATADOG_RUM_PROXY_URL;

datadogRum.init({
  applicationId,
  clientToken,
  site,
  service,
  env,
  version,
  ...(rumProxy ? { proxy: rumProxy } : {}),
});
```

This is a one-time application capability. After it exists, Dogtap adoption and
removal are external configuration changes.

## Local Host App

```bash
DATADOG_RUM_PROXY_URL=http://localhost:8080/datadog-intake-proxy
```

## Docker Compose App

For a browser, `dogtap` is usually not resolvable because it is a Docker network
hostname. Expose Dogtap on the host and use:

```bash
DATADOG_RUM_PROXY_URL=http://localhost:8080/datadog-intake-proxy
```

If the browser reaches the app through a reverse proxy, route a same-origin path
to Dogtap and use:

```bash
DATADOG_RUM_PROXY_URL=/datadog-intake-proxy
```

## Kubernetes App

Expose Dogtap through a reachable Service, Ingress, local port-forward, or app
reverse proxy. A same-pod sidecar loopback address only works for backend SDKs,
not for the user's browser.

## Session Replay

Session Replay uses the same RUM proxy value. Dogtap recognizes uploads routed
with `ddforward=/api/v2/replay`.

## Boundary

A generic external-only frontend path is not possible if the app hardcodes the
Datadog RUM initialization and has no runtime config mechanism. In that case,
make the RUM proxy externally configurable once, then keep future Dogtap use
config-only.
