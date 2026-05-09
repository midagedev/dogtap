# Dogtap Agent Instructions

Dogtap uses spec-driven development. Do not start implementation before checking the relevant spec artifacts.

## Required Reading Order

1. `.specify/memory/constitution.md`
2. `specs/000-product/spec.md`
3. `specs/000-product/plan.md`
4. `specs/000-product/tasks.md`
5. `specs/000-product/gates.md`
6. `docs/AGENT_ORCHESTRATION.md`
7. Relevant docs under `docs/`

## Development Rules

- Specs drive implementation. If code and spec disagree, update the spec first or explicitly record a decision.
- Keep the first implementation narrow. Do not build a Datadog clone.
- Production path features must be fail-open or explicitly bounded by configuration.
- Never persist raw production telemetry by default.
- Redaction and sampling must be testable.
- All external endpoints must have contract tests or fixture-based replay tests.
- Agent-driven work must declare ownership, expected gate, and verification evidence.
- Do not move to a later wave when the current gate has failed unless the later work is independent research.

## Documentation Rules

- Record durable architecture decisions in `docs/decisions/`.
- Keep `docs/ROADMAP.md` aligned with completed milestones.
- Every feature spec should include validation criteria that can be converted into tests.

## Agent Handoff

Every agent handoff should include:

- Summary
- Files changed
- Verification
- Open risks
- Gate status
