# Agent Orchestration Plan

## Purpose

Dogtap should be developed with parallel agents because the product naturally decomposes into protocol adapters, validation logic, dashboard UX, CI tooling, and production safety. The orchestration plan defines how to split work, what each agent owns, and which gates must pass before integration.

## Operating Model

Use short-lived implementation agents with clear ownership and small merge surfaces. Agents should work from specs, produce patches, and include verification evidence. The lead agent integrates results and owns cross-cutting decisions.

Every agent must read:

1. `.specify/memory/constitution.md`
2. `docs/FINAL_GOAL.md`
3. `specs/000-product/spec.md`
4. `specs/000-product/gates.md`
5. This orchestration plan

## Lead Responsibilities

The lead agent owns:

- product scope
- architecture consistency
- work slicing
- integration order
- gate enforcement
- conflict resolution
- release readiness

The lead agent should avoid implementing every slice directly. It should focus on interfaces, contracts, review, and integration.

## Agent Roles

| Agent | Ownership | Primary outputs | Must not touch |
| --- | --- | --- | --- |
| Spec Lead | specs, gates, decisions | updated specs, ADRs, acceptance criteria | runtime implementation |
| Runtime Core Agent | server skeleton, config, event store | process lifecycle, config, store APIs | protocol-specific parsing beyond contracts |
| RUM Agent | browser RUM intake and fixtures | RUM endpoint, decoder, fixtures, tests | dashboard layout |
| Logs Agent | logs intake and fixtures | logs endpoint, gzip/text/JSON decoding, tests | RUM or APM parsers |
| APM Agent | Datadog trace intake | trace endpoint, span normalization, fixtures, tests | OTLP receiver |
| OTLP Agent | OTLP HTTP/gRPC | OTLP receivers, protobuf mapping, fixtures, tests | Datadog private endpoints |
| Validation Agent | rule engine and policy | rule schema, validators, redaction tests | UI styling |
| Dashboard Agent | UI and API consumption | stream/detail/failure/correlation views | core protocol parsing |
| CI Agent | headless mode and reports | validate command, replay command, reports | production forwarding |
| Workflow Contract Agent | contract evaluator, templates, diagnostics contract artifacts | contract schema, contract tests, CI recipes | protocol intake parsing |
| Forwarding Agent | Datadog forwarding | forwarder, retry/drop accounting, config | UI except status API needs |
| Production Safety Agent | limits, sampling, retention | queue limits, TTL, redaction-before-persistence, fault tests | product copy |
| Adoption Integration Agent | adoption profile | validation profile and runbooks | generic product defaults |
| Release Agent | packaging and docs | Docker image, examples, release checklist | feature semantics |

## Parallel Work Waves

### Wave 0: Scope Lock

Goal: prevent agents from building different products.

Parallel work:

- Spec Lead finalizes MVP cut and gate definitions.
- Runtime Core Agent proposes module boundaries.
- Dashboard Agent sketches information architecture.
- Validation Agent proposes rule schema.

Integration gate:

- G0 Spec Readiness

### Wave 1: Fixture and Protocol Evidence

Goal: collect enough real payloads to avoid designing from guesses.

Parallel work:

- RUM Agent captures browser RUM fixtures.
- Logs Agent captures JSON, text, and gzip log fixtures.
- APM Agent captures Datadog Java tracer fixtures and evaluates `dd-apm-test-agent`.
- OTLP Agent captures OTLP HTTP/gRPC fixtures.

Integration gate:

- G1 Fixture Evidence

### Wave 2: Runtime Skeleton and Contracts

Goal: create the integration surface that protocol agents can plug into.

Parallel work:

- Runtime Core Agent implements config, lifecycle, event store, health endpoints.
- Validation Agent implements rule interfaces and base validators.
- Dashboard Agent implements UI shell against mocked API.
- CI Agent implements report schema and command shape.

Integration gate:

- G2 Runtime Contract

### Wave 3: Intake Adapters

Goal: make real telemetry appear as normalized events.

Parallel work:

- RUM Agent implements RUM intake.
- Logs Agent implements logs intake.
- APM Agent implements trace intake.
- OTLP Agent implements OTLP intake.

Integration gate:

- G3 Protocol Intake

### Wave 4: Validation, UI, and CI

Goal: turn payload visibility into product value.

Parallel work:

- Validation Agent implements required context, PII, token, query, and cardinality rules.
- Dashboard Agent implements stream, detail, failure inbox, and correlation view.
- CI Agent implements fixture replay and validation exit codes.
- Adoption Integration Agent writes a realistic validation profile.

Integration gate:

- G4 Product Usability
- G5 CI Contract

### Wave 5: Forwarding and Production Safety

Goal: make staging and limited production use credible.

Parallel work:

- Forwarding Agent implements forward modes.
- Production Safety Agent implements queue, sampling, TTL, redaction-before-persistence, and fault tests.
- Dashboard Agent adds forwarding and safety status.
- Release Agent prepares Docker image and deployment examples.

Integration gate:

- G6 Forwarding Safety
- G7 Production Safety

### Wave 6: Release Candidate

Goal: publish a coherent first release.

Parallel work:

- Release Agent finalizes image, examples, and changelog.
- Spec Lead verifies docs match behavior.
- Adoption Integration Agent validates one real adoption path.
- Workflow Contract Agent verifies template and CI recipe consistency.
- Lead Agent runs full gate checklist.

Integration gate:

- G8 Release Candidate

## Handoff Format

Each agent returns:

```text
Summary
- What changed

Files changed
- path

Verification
- command or evidence

Open risks
- risk and suggested next step

Gate status
- passed / failed / not applicable
```

## Integration Rules

- One agent owns one write surface per wave.
- Shared interfaces must be changed by the lead or by a coordinated patch.
- Protocol adapters must add fixtures before implementation is considered complete.
- Dashboard changes must include at least one fixture-backed screen state.
- Production-related changes must include failure behavior tests.
- If a gate fails, do not proceed to the next wave except for independent research tasks.

## Agent Prompt Template

```text
You are working on Dogtap.

Read:
- .specify/memory/constitution.md
- docs/FINAL_GOAL.md
- docs/AGENT_ORCHESTRATION.md
- specs/000-product/spec.md
- specs/000-product/gates.md

Your ownership:
- <owned files/modules>

Your task:
- <specific task>

Constraints:
- Do not edit files outside your ownership unless required for a compile/test fix.
- Do not broaden product scope.
- Add or update fixtures/tests for behavior changes.
- Report gate status in your final response.
```
