# 0010: Workflow Observability Contracts

Date: 2026-05-10

## Status

Accepted

## Context

Dogtap already verifies whether expected telemetry sources, services, sessions,
traces, routes, metrics, and endpoints were observed. That is enough for a
generic missing-signal check, but it does not tell a team whether a specific app
workflow is instrumented well enough to debug real failures.

The highest-value product lane is therefore not becoming a Datadog clone. It is
turning local and isolated E2E telemetry into a contract that can be asserted by
developers, CI, and coding agents.

## Decision

Dogtap will support additive workflow contracts evaluated against retained
events:

- contracts are YAML or JSON definitions with named checks
- checks cover event presence, log message presence, metric presence,
  browser-to-backend trace correlation, and obvious sensitive value leakage
- diagnostics API responses may include `workflowContracts`
- diagnostics archives may include `workflow-contracts.json`
- `dogtap diagnose` accepts repeatable `-workflow-contract` files
- workflow contract failures do not change existing diagnostics
  `assertions.status` unless the CLI caller passes `-fail-on-workflow-contract`

The dashboard evaluates a built-in frontend/backend readiness contract so local
users can immediately see whether RUM, Session Replay, backend logs, traces,
metrics, and basic privacy checks are present.

## Consequences

- Existing diagnostics clients keep their assertion semantics.
- Teams can add workflow-specific contracts such as login, checkout, case open,
  or report export without writing Dogtap-specific application SDK code.
- Contract failures remain agent-readable and point at the missing evidence.
- Dogtap stays an inspector and contract validator, not a production query
  engine or monitor system.
