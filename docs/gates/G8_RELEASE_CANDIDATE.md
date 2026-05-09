# G8 Release Candidate Evidence

Date: 2026-05-09

## Status

Blocked.

Dogtap has enough public project structure for source review and community
testing, but it is not a final release candidate until one realistic sanitized
adoption profile validates successfully.

## Passed Release-Candidate Subsets

- Public README, support matrix, release candidate runbook, and community
  templates exist.
- CI runs Go tests, dashboard build/E2E, seeded demo visual verification,
  generic adoption smoke, shell syntax checks, and container build.
- Tag-based release automation exists for binary archives and GHCR images.
- Generic adoption kit and seeded demo are public and reproducible.
- G0 through G7 evidence is documented, with APM and OTLP forwarding explicitly
  scoped out of the first forwarding slice.

## Blocking Evidence

G8 still requires a realistic sanitized adoption profile that proves a normal
frontend/backend app can adopt Dogtap locally and in CI without application code
changes beyond standard Datadog or OTLP endpoint configuration.

Private or raw evidence must stay under `.private/adoption/`. Public G8 evidence
must contain only sanitized summaries, fixture names, commands, validation
results, and screenshots that are safe for a public repository.

The public evidence must confirm:

- Browser RUM and Session Replay arrive with expected user/session/route context.
- Backend logs and traces correlate by trace/span or workflow context.
- OTLP metrics appear in the dashboard and replay reports where applicable.
- Dogtap can be enabled and removed by deleting external overrides or restoring
  standard Datadog/OTLP endpoint values, without Dogtap-specific application
  SDK code.
- Existing Datadog Agent-only behaviors such as stdout/file log tailing and
  DogStatsD are either bridged into Dogtap or explicitly preserved on the
  Datadog production lane.
- Required context and redaction rules catch at least one meaningful failure.
- No company names, customer payloads, credentials, private hosts, or raw
  production telemetry are committed.

## Verification Commands For The Next G8 Attempt

```bash
go test ./...
npm --prefix web run build
make shell-check
make smoke-adoption
make demo-visual-check
go run ./cmd/dogtap replay \
  -config configs/generic-local.yaml \
  -format markdown \
  fixtures/rum/login.json \
  fixtures/logs/json-log.json \
  fixtures/apm/trace.json \
  fixtures/otlp/traces.json
```

Add the realistic sanitized adoption replay command and its result here before
marking G8 passed.
