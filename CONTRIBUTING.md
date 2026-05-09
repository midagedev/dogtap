# Contributing

Dogtap uses spec-driven development. Before changing behavior, check:

- `.specify/memory/constitution.md`
- `specs/000-product/spec.md`
- `specs/000-product/plan.md`
- `specs/000-product/tasks.md`
- `specs/000-product/gates.md`
- `docs/AGENT_ORCHESTRATION.md`

## Development Setup

```bash
npm --prefix web install
npm --prefix web run build
go test ./...
go run ./cmd/dogtap serve
```

## Expected Checks

Run the relevant subset before opening a pull request:

```bash
go test ./...
npm --prefix web run build
make shell-check
make smoke-adoption
```

Dashboard changes should include browser-driven verification when possible:

```bash
DOGTAP_E2E_BASE_URL=http://127.0.0.1:5178 npm --prefix web run test:e2e -- --project=chromium --project=mobile
```

## Scope Rules

- Keep Dogtap an intake inspector, not a Datadog clone.
- Prefer standard Datadog and OpenTelemetry configuration surfaces.
- Do not add a Dogtap-specific app SDK unless the spec changes.
- Production path changes must be bounded, fail-open where appropriate, and
  covered by safety tests.
- Do not persist raw production telemetry by default.
- Redaction and sampling behavior must be testable.

## Fixtures

Protocol changes should add or update fixture-backed evidence. Prefer real SDK
or tracer payloads where practical. Synthetic fixtures are useful for smoke
coverage, but should be clearly labeled as such.

## Private Or Company-Specific Material

Do not commit private adoption profiles, company-specific runbooks, customer
payloads, credentials, or local evidence. Keep those outside the public tree or
under an ignored local directory such as `.private/`.
