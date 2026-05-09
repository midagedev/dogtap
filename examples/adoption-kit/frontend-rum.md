# Frontend RUM

Configure the Datadog Browser RUM SDK proxy option to point at Dogtap.

For a browser app running on the host:

```ts
datadogRum.init({
  applicationId: "local",
  clientToken: "local",
  site: "datadoghq.com",
  service: "your-frontend",
  env: "local",
  version: "local",
  proxy: "http://localhost:8080/datadog-intake-proxy",
  sessionSampleRate: 100,
  sessionReplaySampleRate: 100,
});
```

For Vite-style runtime config:

```bash
VITE_DATADOG_RUM_PROXY_URL=http://localhost:8080/datadog-intake-proxy
```

Then wire the value into the existing Datadog RUM init object:

```ts
const proxy = import.meta.env.VITE_DATADOG_RUM_PROXY_URL;

datadogRum.init({
  applicationId: "local",
  clientToken: "local",
  site: "datadoghq.com",
  service: "your-frontend",
  env: "local",
  version: "local",
  ...(proxy ? { proxy } : {}),
});
```

Session Replay uses the same proxy path. Dogtap recognizes Datadog replay
uploads routed through `ddforward=/api/v2/replay`.
