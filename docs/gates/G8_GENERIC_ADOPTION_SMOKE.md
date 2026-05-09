# G8 Generic Adoption Smoke

Date: 2026-05-09

## Scope

This gate covers the generic quickstart subset of G8. It does not complete the
full release candidate gate, which still requires at least one realistic
adoption profile to validate successfully.

## Evidence

Implemented:

- Generic adoption ADR:
  `docs/decisions/0006-generic-adoption-kit.md`
- Generic local config:
  `configs/generic-local.yaml`
- Copyable adoption templates:
  `examples/adoption-kit/`
- Empty-state dashboard intake targets:
  `web/src/main.tsx`, `web/src/styles.css`
- Smoke script:
  `scripts/generic/smoke.sh`
- Seeded dashboard demo and visual check:
  `scripts/demo/seed.sh`, `scripts/demo/visual-check.sh`
- Release workflow:
  `.github/workflows/release.yml`
- Support matrix and release candidate runbook:
  `docs/SUPPORT_MATRIX.md`, `docs/runbooks/RELEASE_CANDIDATE.md`

Verification:

```bash
make shell-check
go test ./...
npm --prefix web run build
make smoke-adoption
make demo-visual-check
DOGTAP_E2E_BASE_URL=http://127.0.0.1:5178 npm --prefix web run test:e2e -- --project=chromium --project=mobile
go run ./cmd/dogtap replay -config configs/generic-local.yaml -format markdown fixtures/rum/login.json fixtures/logs/json-log.json fixtures/apm/trace.json fixtures/otlp/traces.json
```

Results:

- Shell syntax passed.
- Go tests passed.
- Dashboard production build passed.
- Generic smoke passed and verified received `rum`, `logs`, `apm`, `otlp`, and
  OTLP metric events.
- Seeded visual check passed and verified RUM, Session Replay, logs, APM spans,
  OTLP metrics, service map, traffic, failure inbox, and dashboard screenshots
  on desktop and mobile.
- Dashboard E2E passed on desktop Chromium and mobile.
- Generic config replay passed with four passing fixture events.
- Release workflow exists for tag-based binary archives and GHCR images.

Visual evidence was inspected locally through a Playwright screenshot during
development. Generated screenshots are not committed.

## Gate Status

Passed for the generic quickstart subset.
