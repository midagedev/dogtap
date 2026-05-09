# G6 Forwarding Safety Evidence

## Status

Passed for RUM and logs forwarding.

APM forwarding is explicitly deferred by
`docs/decisions/0004-forwarding-strategy.md` until real tracer fixture evidence
exists.

## Evidence

Implemented forwarding safety:

- Forwarding is disabled by default and controlled by configuration.
- RUM and logs payloads are forwarded through bounded HTTP attempts.
- Logs forwarding requires an API key; the key is sent outbound only and is not
  persisted or returned by the config API.
- Forwarding results record mode, target, status, status code, duration, retry
  count, and failure class/message.
- Retry attempts are bounded by configuration and a hard cap.
- Unsupported forwarding sources are recorded as unsupported instead of silently
  pretending to forward.
- `/metrics` exposes forwarding payload, attempt, retry, success, failure, and
  drop counters.

## Verification

```bash
go test ./internal/forwarding ./internal/server
go test ./...
```

## Remaining Risks

- Real RUM SDK and logs intake forwarding compatibility still needs to be
  checked with promoted G1 evidence.
- APM forwarding remains out of scope for the first forwarding slice.
