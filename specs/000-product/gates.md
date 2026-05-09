# Success Gates

## Purpose

Dogtap development should move quickly with parallel agents, but every area needs objective gates. A gate is passed only when its evidence is written down and reproducible.

## Gate Summary

| Gate | Name | Blocks |
| --- | --- | --- |
| G0 | Spec Readiness | implementation start |
| G1 | Fixture Evidence | protocol implementation |
| G2 | Runtime Contract | adapter integration |
| G3 | Protocol Intake | product UI and CI claims |
| G4 | Product Usability | MVP release |
| G5 | CI Contract | adoption by another repo |
| G6 | Forwarding Safety | staging forwarding |
| G7 | Production Safety | production tee or forward mode |
| G8 | Release Candidate | public release |

## G0: Spec Readiness

Pass criteria:

- Final goal is documented.
- MVP cut is explicit.
- Non-goals are documented.
- Agent orchestration plan exists.
- Success gates are documented.
- Architecture direction is documented or explicitly marked as proposed.

Failure examples:

- Agents can interpret MVP differently.
- Production mode scope is ambiguous.
- No owner exists for validation or safety.

## G1: Fixture Evidence

Pass criteria:

- RUM fixture captured from a real Datadog Browser RUM SDK.
- Logs fixtures cover JSON, text, and gzip.
- APM fixture captured from a real Datadog tracer or `dd-apm-test-agent` reference flow.
- OTLP fixture captured from a real OpenTelemetry SDK or collector.
- Each fixture has expected normalized fields and redaction expectations.

Failure examples:

- Parser implementation relies only on hand-written sample JSON.
- No fixture covers missing context.
- No fixture covers unsafe values.

## G2: Runtime Contract

Pass criteria:

- Config loads from env and file.
- Event store has bounded retention.
- Health and readiness endpoints exist.
- Intake adapters can write event envelopes through a stable interface.
- Dashboard API can read redacted event envelopes.
- CI reporter can read validation results.

Failure examples:

- Protocol adapters write directly to UI-specific structures.
- Raw payload storage cannot be disabled.
- Config behavior differs between local and CI.

## G3: Protocol Intake

Pass criteria:

- RUM, logs, APM, and OTLP endpoints accept fixture payloads.
- Unsupported content types fail with useful errors.
- Decoding handles gzip where expected.
- Normalized fields are populated for each supported source.
- Fixture replay is deterministic.

Failure examples:

- A payload is accepted but cannot be inspected.
- Normalized trace IDs are inconsistent.
- Decoding errors are swallowed.

## G4: Product Usability

Pass criteria:

- Dashboard shows stream, detail, validation failure, and correlation views.
- A developer can identify why a payload failed validation.
- Raw payload and normalized view are both accessible in local mode.
- Production mode hides raw payload by default.
- Datadog search query hints are copyable.

Failure examples:

- UI shows payloads but not validation evidence.
- User cannot filter by user/account/workspace/case/trace.
- Dashboard hides the actual received payload in local mode.

## G5: CI Contract

Pass criteria:

- Headless command runs without dashboard.
- Fixture replay can fail on validation errors.
- JSON and Markdown reports are generated.
- Exit code policy is tested.
- Reports include enough evidence for a developer to fix instrumentation.

Failure examples:

- CI only checks that payloads arrived.
- Reports lack field paths or rule IDs.
- Redaction differs between dashboard and CI.

## G6: Forwarding Safety

Pass criteria:

- Forward mode records target, status, duration, and failure reason.
- Datadog API keys are never persisted or shown.
- Forwarding can be disabled without code changes.
- Retry and drop behavior is bounded.
- Forwarded payload mutation is explicit by mode.

Failure examples:

- Forward failure blocks local dashboard.
- Forwarding retries forever.
- Secret config appears in event detail.

## G7: Production Safety

Pass criteria:

- Raw production payload retention is disabled by default.
- Redaction runs before persistence.
- Sampling, queue limit, TTL, and max event count are configurable.
- Queue-full behavior is tested.
- Datadog unavailable behavior is tested.
- Dogtap emits its own health and operational metrics.
- Removal or bypass procedure is documented.

Failure examples:

- Production mode stores raw payloads by default.
- Dogtap can block application behavior unexpectedly.
- Backpressure behavior is implicit.

## G8: Release Candidate

Pass criteria:

- All previous gates pass or have accepted ADRs explaining scope deferral.
- Docker image builds reproducibly.
- Quickstart works on a clean machine.
- Public README reflects actual behavior.
- Release notes list supported endpoints and limitations.
- At least one realistic adoption profile validates successfully.

Failure examples:

- Docs promise endpoints that do not exist.
- Quickstart requires undocumented setup.
- Known production risks are not documented.

