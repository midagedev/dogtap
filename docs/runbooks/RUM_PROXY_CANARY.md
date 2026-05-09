# RUM Proxy Canary

This runbook defines the smallest safe canary for routing Datadog Browser RUM
and Session Replay payloads through Dogtap while preserving an application's
existing Datadog Browser SDK integration.

Use this when:

- the frontend already uses the Datadog Browser RUM SDK
- the RUM `proxy` option can be supplied by runtime config
- Dogtap is being evaluated in local, CI, preview, or a tightly bounded staging
  environment

Do not use this as a public unauthenticated proxy. Dogtap is an inspection
surface, not a hardened internet edge.

## Source Contract

The Datadog Browser SDK proxy contract has several requirements that Dogtap
adoptions should preserve:

- Use Browser SDK `4.34.0` or later.
- Configure the SDK `proxy` initialization value.
- Expect the SDK to send POST requests to the proxy with a `ddforward` query
  parameter containing the Datadog intake path and query.
- Preserve the request body as bytes; Session Replay payloads can contain
  binary data.
- Strip sensitive inbound headers, especially cookies and authorization data,
  before forwarding.
- Do not turn `ddforward` into an arbitrary upstream URL. It must resolve only
  to allowed Datadog intake paths.

Primary references:

- [Datadog RUM proxy guide](https://docs.datadoghq.com/real_user_monitoring/guide/proxy-rum-data/)
- [Datadog Browser Session Replay](https://docs.datadoghq.com/session_replay/browser/)

Dogtap accepts:

- `/datadog-intake-proxy?ddforward=/api/v2/rum?...`
- `/datadog-intake-proxy?ddforward=/api/v2/replay?...`
- direct `/api/v2/replay` for fixture and compatibility workflows

Dogtap's replay viewer renders decoded rrweb full snapshot records as an
iframe DOM replay. If the replay segment is unavailable, redacted, or only
partially decoded, the dashboard falls back to payload timeline and metadata.

## Canary Shape

Prefer same-origin browser routing when possible:

```text
browser
  -> https://app.example.test/datadog-intake-proxy
  -> Dogtap
  -> optional bounded Datadog RUM/replay forwarding
```

Backend sidecars can use loopback or a Compose service name. Browser RUM cannot
use pod-local loopback or a Docker-only hostname unless the browser itself can
resolve and reach that address.

## Prerequisites

Before enabling the canary:

- Confirm the frontend uses `@datadog/browser-rum` or Datadog CDN Browser SDK
  version `4.34.0` or newer.
- Confirm `proxy` is read from runtime config, such as
  `DATADOG_RUM_PROXY_URL`, and omitted when unset.
- Confirm the canary environment can expose Dogtap through localhost,
  same-origin reverse proxy, service, ingress, or port-forward.
- Keep `DOGTAP_ALLOW_RAW_PAYLOADS=false` outside local-only debugging.
- Bound Dogtap storage with `DOGTAP_STORAGE_MAX_EVENTS`,
  `DOGTAP_STORAGE_TTL`, and `DOGTAP_SAMPLING_RATE`.
- Keep normal Datadog `applicationId`, `clientToken`, `site`, `service`, `env`,
  and `version` values unless the canary intentionally isolates them.

## Runtime Config

Local host app:

```bash
DATADOG_RUM_PROXY_URL=http://localhost:8080/datadog-intake-proxy
```

Same-origin app route:

```bash
DATADOG_RUM_PROXY_URL=/datadog-intake-proxy
```

Browser SDK initialization should remain conditional:

```ts
const rumProxy = window.__RUNTIME_CONFIG__?.DATADOG_RUM_PROXY_URL;

datadogRum.init({
  applicationId,
  clientToken,
  site,
  service,
  env,
  version,
  sessionSampleRate,
  sessionReplaySampleRate,
  ...(rumProxy ? { proxy: rumProxy } : {}),
});
```

For staging canaries, keep `sessionSampleRate` and
`sessionReplaySampleRate` intentionally low or targeted. Local and fixture
environments can temporarily use higher rates to prove the path.

## Proxy And Forwarding Checks

When a same-origin reverse proxy sits in front of Dogtap, configure it to:

- forward only POST and OPTIONS for `/datadog-intake-proxy`
- pass the raw request body without JSON parsing, string conversion, or
  re-compression
- preserve `Content-Type`, `Content-Encoding`, `DD-EVP-ORIGIN`, and
  `DD-EVP-ORIGIN-VERSION`
- drop inbound `Cookie`, `Authorization`, and other application credentials
- allow only `ddforward` paths under `/api/v2/rum` and `/api/v2/replay`
- reject absolute or scheme-relative `ddforward` values

When Dogtap forwarding is enabled, Dogtap forwards RUM and replay bodies with
the allowlisted Datadog path and query from `ddforward`. Forwarding metadata
keeps the target origin and path but does not retain the client token query.

## Validation Steps

1. Start Dogtap in `local`, `forward`, or `tee` mode with bounded storage.
2. Set the runtime RUM proxy value for only the canary environment.
3. Load the app and exercise login, route navigation, an expected frontend
   action, and logout.
4. If Session Replay is enabled, exercise at least one interaction after the
   session starts recording.
5. Open the Dogtap dashboard and check:
   - RUM events are listed with `source=rum`.
   - Replay uploads show `payloadKind=replay`.
   - Service, env, version, route, user, workspace, and account context appear
     where the app is expected to set them.
   - Validation failures do not include secrets, tokens, or stale context after
     logout.
   - Forwarding status is visible when forwarding is configured.
6. Query `/api/events?source=rum` or export a debug bundle for review evidence.

Do not publish raw payloads, local client tokens, or replay segments from a real
application. Publish only sanitized summaries, commands, and screenshots.

## Rollback

Rollback must be configuration-only:

- remove `DATADOG_RUM_PROXY_URL` or set it back to the previous proxy value
- disable the same-origin proxy route or ingress rule
- stop the Dogtap sidecar/service
- keep the normal Datadog Browser SDK init and Datadog production lane intact

If Dogtap used file storage during the canary, remove the local Dogtap volume or
file after preserving any approved sanitized evidence.

## Exit Criteria

The canary is successful when:

- Dogtap receives both normal RUM and Session Replay payloads.
- The dashboard shows the expected RUM context and DOM replay when decoded
  rrweb full snapshot records are present.
- Removing the runtime proxy value restores the original Datadog path without a
  frontend rebuild.
- No raw production telemetry or internal evidence is committed to the public
  repository.
