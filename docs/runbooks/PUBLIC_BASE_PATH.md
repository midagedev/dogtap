# Public Base Path

Dogtap can run behind a shared reverse proxy under a public path prefix such as
`/dogtap`.

## Contract

Use the same public base path for the web build and the Dogtap runtime:

```bash
PUBLIC_BASE_PATH=/dogtap npm --prefix web run build
PUBLIC_BASE_PATH=/dogtap dogtap serve
```

`DOGTAP_PUBLIC_BASE_PATH=/dogtap` is also accepted for the Dogtap runtime. YAML
config can set:

```yaml
server:
  publicBasePath: /dogtap
```

When Dogtap is mounted at `/dogtap`, browser-visible URLs become:

```text
Dashboard: https://localhost:8081/dogtap/
Assets:    https://localhost:8081/dogtap/assets/...
API:       https://localhost:8081/dogtap/api/...
RUM proxy: https://localhost:8081/dogtap/datadog-intake-proxy
```

Internal service URLs stay unprefixed:

```text
http://dogtap:8080/api/...
http://dogtap:8080/datadog-intake-proxy
```

## Reverse Proxy

The proxy can either preserve or strip the prefix. Dogtap accepts both
`/dogtap/api/...` and `/api/...` when configured with `PUBLIC_BASE_PATH=/dogtap`
or when the proxy sends `X-Forwarded-Prefix: /dogtap`.

Recommended headers:

```nginx
proxy_set_header X-Forwarded-Prefix /dogtap;
proxy_set_header X-Forwarded-Proto $scheme;
proxy_set_header X-Forwarded-Host $host;
```

With that contract, the proxy does not need to rewrite HTML or JavaScript
responses.

## Notes

`PUBLIC_BASE_PATH` affects Vite asset URLs at build time. If you use the
prebuilt root-path assets, Dogtap can still accept prefixed API requests, but
the browser may request root `/assets/...` URLs. Build with the intended public
base path when serving the dashboard under a prefix.
