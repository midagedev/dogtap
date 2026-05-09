# G4 Product Usability Evidence

## Status

Passed for the smoke fixture-backed MVP dashboard.

Release readiness remains blocked by G1/G3 real fixture evidence, but the G4
product surface criteria are implemented and locally verified.

## Evidence

Implemented usability surface:

- Request stream and source/status filters.
- Event detail view with endpoint, timing, normalized context, validation rules,
  and payload view.
- Validation failure inbox with failing-rule filter.
- Correlation hints across trace, user, workspace, and case keys.
- Copyable Datadog search query builder.
- Debug bundle export API for filtered event evidence.
- Local mode exposes raw payloads; non-local modes hide raw payloads by default.

## Verification

```bash
npm --prefix web run build
DOGTAP_E2E_BASE_URL=http://127.0.0.1:4175 npm --prefix web run test:e2e
go test ./internal/server
```

## Remaining Risks

- Datadog search field names are best-effort until real Datadog query behavior is
  checked against promoted G1 evidence.
- Correlation currently uses the recent event snapshot returned by
  `/api/events?limit=100`.
