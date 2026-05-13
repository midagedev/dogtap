# 0016 Public Base Path Contract

## Status

Accepted.

## Context

Dogtap is commonly trialed behind a shared reverse proxy next to another app.
In that shape, the browser sees a public URL such as
`https://localhost:8081/dogtap`, while container-to-container traffic still uses
unprefixed internal URLs such as `http://dogtap:8080`.

HTML or JavaScript response rewriting at the proxy is brittle. It also makes it
hard for coding agents to reason about which URL should be used for browser
traffic versus internal service traffic.

## Decision

Dogtap supports a prefix-aware public deployment contract:

- `PUBLIC_BASE_PATH=/dogtap` configures both Vite builds and Dogtap runtime
  config.
- `DOGTAP_PUBLIC_BASE_PATH=/dogtap` is accepted as an explicit Dogtap runtime
  alias.
- YAML config can set `server.publicBasePath: /dogtap`.
- HTTP routing accepts both unprefixed internal paths and prefixed public paths.
- `X-Forwarded-Prefix` is honored for request routing and generated public base
  URLs, including diagnostics output.
- Dashboard API calls and browser-facing intake links are generated with a
  prefix-aware URL helper instead of hardcoded absolute `/api/...` paths.

Internal service communication remains unprefixed. For example, Compose services
can keep using `http://dogtap:8080/api/...`, while browsers use
`https://localhost:8081/dogtap/api/...`.

## Consequences

Deployments can route `/dogtap/` to Dogtap without HTML/JS rewriting. A build
that needs absolute prefixed assets should run the web build with
`PUBLIC_BASE_PATH=/dogtap`. Runtime-only reverse proxies can also pass
`X-Forwarded-Prefix: /dogtap` so generated diagnostics URLs reflect the browser
path.

This does not make Dogtap a multi-tenant router. The prefix is a single public
mount path for one Dogtap instance.
