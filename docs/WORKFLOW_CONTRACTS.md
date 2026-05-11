# Workflow Contracts

Workflow contracts turn received telemetry into a small, agent-readable answer:
did this user path emit enough evidence to debug it?

They are intended for local dev, isolated E2E, and CI runs. They do not require
Datadog credentials and they do not replace production observability.

## Contract File

```yaml
name: login-workflow
description: Verifies login telemetry.
checks:
  - id: login-rum-context
    type: event
    source: rum
    routeRegex: "^/(login|signin|auth)"
    fields:
      - sessionId
      - userId
  - id: login-backend-log
    type: log-message
    source: logs
    pattern: "(?i)login|signin|auth"
  - id: login-latency-metric
    type: metric
    pattern: "http\\.server\\.request\\.duration"
  - id: login-browser-to-api-trace
    type: trace-correlation
    from:
      source: rum
      fields:
        - sessionId
        - traceId
    to:
      payloadKind: trace
      service: api-service
  - id: login-no-sensitive-values
    type: no-sensitive-values
```

Example files live under `configs/contracts/`:

- `frontend-backend.yaml`: generic readiness for the dashboard and smoke runs
- `login.yaml`: login/sign-in/auth
- `case-open.yaml`: opening a case or record detail page
- `checkout.yaml`: checkout/payment/purchase
- `subscription.yaml`: account, billing, subscription, or plan-change flows
- `report-export.yaml`: report export or file generation

## Starter Pack

For a typical frontend/backend app, start with these contracts and delete the
ones that do not match your product:

```text
configs/contracts/frontend-backend.yaml
configs/contracts/login.yaml
configs/contracts/subscription.yaml
configs/contracts/checkout.yaml
configs/contracts/case-open.yaml
configs/contracts/report-export.yaml
```

The fields most teams should edit first are `service`, `route`, `routeRegex`,
`pattern`, and `metric`. Keep check IDs stable once CI starts consuming them so
review comments and diagnostics archives remain comparable across runs.

## Check Types

| Type | Purpose |
| --- | --- |
| `event` | At least one event matches source, payload kind, service, route, route regex, and required normalized fields. |
| `log-message` | At least one matching log detail contains the configured regex pattern. |
| `metric` | At least one matching metric detail has a name matching `metric` or `pattern`. |
| `trace-correlation` | A source selector and destination selector share a canonical trace ID. Decimal, hex, and base64 trace IDs are normalized. |
| `no-sensitive-values` | Visible normalized fields, tags, headers, query strings, and log messages do not contain obvious email, bearer token, or JWT values. |

## CLI

Validate a contract before running a workflow:

```bash
dogtap contract validate configs/contracts/login.yaml
```

For agent-readable output:

```bash
dogtap contract validate -format json configs/contracts/login.yaml
```

The validator catches missing names, empty check lists, duplicate check IDs,
unsupported check types, unsupported sources, unsupported selector fields,
unknown YAML/JSON fields, and invalid regular expressions.

Editor integrations can use the JSON Schema at
`schemas/workflow-contract.schema.json`.

Run a contract against retained Dogtap telemetry:

```bash
dogtap diagnose \
  -base-url http://127.0.0.1:8080 \
  -workflow-contract configs/contracts/login.yaml \
  -fail-on-workflow-contract
```

Without `-fail-on-workflow-contract`, contract failures are written to
`workflow-contracts.json` and `summary.md` but do not change the command exit
code if the normal diagnostics assertions pass.

## API

```bash
curl -sS -X POST http://127.0.0.1:8080/api/diagnostics \
  -H 'Content-Type: application/json' \
  -d '{"useDefaultWorkflowContracts":true}'
```

Responses include `workflowContracts` when contracts are supplied or requested.
Archives include `workflow-contracts.json`.

## CI Pattern

In CI, run Dogtap beside the app, execute the normal E2E suite, then add one
assertion step:

```bash
dogtap diagnose \
  -base-url http://127.0.0.1:8080 \
  -output .tmp/dogtap-diagnostics \
  -workflow-contract .dogtap/contracts/login.yaml \
  -fail-on-workflow-contract \
  -expect-non-empty
```

The key design choice is that Dogtap verifies the E2E suite's telemetry output;
it does not replace the suite. A complete GitHub Actions template is available
at `examples/github-actions/workflow-contract.yml`.

## Dashboard

The dashboard requests the built-in frontend/backend readiness contract. It
checks for browser context, Session Replay payloads, backend logs, backend
traces, at least one metric, and obvious sensitive value leaks. The dashboard
shows pass and fail checks, matched event IDs, and trace IDs that can be opened
in the event detail pane when matching retained telemetry is available. Failed
checks also show evaluated selector criteria and nearby retained events so a
user can see whether the wrong source, route, service, or context fields were
observed.
