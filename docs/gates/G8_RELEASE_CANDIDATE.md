# G8 Release Candidate Evidence

Date: 2026-05-09

## Status

Passed.

Dogtap has public project structure, safety gates, executable adoption
fixtures, visual verification, and one realistic sanitized adoption profile for
the first public release candidate.

## Passed Release-Candidate Subsets

- Public README, support matrix, release candidate runbook, and community
  templates exist.
- CI runs Go tests, dashboard build/E2E, seeded demo visual verification,
  generic adoption smoke, filelog bridge smoke, StatsD bridge smoke, shell
  syntax checks, docs/spec checks, workflow contract checks, and container
  build.
- Tag-based release automation exists for binary archives and GHCR images.
- Generic adoption kit and seeded demo are public and reproducible.
- G0 through G7 evidence is documented, with APM and OTLP forwarding explicitly
  scoped out of the first forwarding slice.
- Sanitized adoption profile evidence is recorded in
  `docs/gates/G8_SANITIZED_ADOPTION_PROFILE.md`.

## Sanitized Adoption Evidence

G8 includes a realistic sanitized adoption profile that proves a normal
frontend/backend app can adopt Dogtap locally and in CI without Dogtap-specific
SDK code or application dependencies.

Private or raw evidence must stay under `.private/adoption/`. Public G8 evidence
must contain only sanitized summaries, fixture names, commands, validation
results, and screenshots that are safe for a public repository.

The public evidence confirms:

- Browser RUM and multipart Session Replay arrive with expected
  user/session/route context and a decoded rrweb DOM snapshot.
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

## Verification Commands

```bash
go test ./...
npm --prefix web run build
make shell-check
make doc-check
make contract-check
make smoke-adoption
make smoke-log-bridge
make smoke-statsd-bridge
make demo-visual-check
make smoke-external-injection
go run ./cmd/dogtap replay \
  -config configs/generic-local.yaml \
  -format markdown \
  fixtures/rum/login.json \
  fixtures/logs/json-log.json \
  fixtures/apm/trace.json \
  fixtures/otlp/traces.json
```

## Gate Decision

G8 is passed for the first public release candidate. Tagging still requires the
maintainer release checklist in `docs/runbooks/RELEASE_CANDIDATE.md`.
