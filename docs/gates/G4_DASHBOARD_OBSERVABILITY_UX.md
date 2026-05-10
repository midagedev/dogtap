# G4 Dashboard Observability UX Evidence

## Scope

This gate covers the first dashboard usability slice for retained logs and
metrics:

- structured log field drilldown
- retained metric snapshot charts
- metric detail summary cards
- desktop and mobile Playwright verification

## Evidence

Implemented files:

- `web/src/main.tsx`
- `web/src/styles.css`
- `web/e2e/dashboard.spec.ts`

Verification commands:

```bash
npm --prefix web run build
DOGTAP_E2E_BASE_URL=http://127.0.0.1:5177 \
  npm --prefix web run test:e2e -- --project=chromium --project=mobile e2e/dashboard.spec.ts
```

Visual verification:

- Desktop screenshot: `web/test-results/dashboard-log-metric-observability.png`
- Mobile viewport was inspected at 390x844 through Playwright.

Covered behavior:

- JSON/Datadog-style log payloads show route, status, trace, span, service,
  env, user, account, and workspace fields without hiding payload access.
- Metrics snapshot groups retained samples into compact chart rows with latest,
  min, max, and sample count.
- Metric detail view shows related retained samples and summary charts.
- Dashboard E2E covers both desktop Chromium and mobile viewport projects.

## Gate Status

Passed for the dashboard observability UX first slice.

This is still an inspection UI, not a monitoring dashboard or Datadog chart
replacement.
