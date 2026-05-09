# Data Model

## Event Envelope

Every received payload is represented as an event envelope.

```text
EventEnvelope
  id
  receivedAt
  source
  payloadKind
  endpoint
  method
  headers
  contentType
  contentEncoding
  bodySizeBytes
  decodedSizeBytes
  rawBodyRef
  details
  normalized
  validation
  forwarding
```

## Source

Allowed initial values:

- `rum`
- `apm`
- `logs`
- `otlp`
- `unknown`

## Normalized Fields

```text
NormalizedTelemetry
  service
  env
  version
  host
  source
  timestamp
  traceId
  spanId
  parentSpanId
  sessionId
  viewId
  userId
  accountId
  workspaceId
  caseId
  route
  method
  statusCode
  durationMs
  errorType
  errorMessage
  tags
```

## Source-Specific Details

Dogtap keeps the generic decoded payload, but the dashboard and reports use
typed details when a source has a stable inspection shape.

```text
TelemetryDetails
  replay
  logs
  trace
```

```text
ReplayDetail
  format
  contentType
  bytes
  recordCount
  segmentBytes
  segmentContentType
  segmentFilename
  sessionId
  viewId
  start
  end
```

```text
LogEntry
  timestamp
  level
  message
  traceId
```

```text
TraceDetail
  traceId
  spans
```

```text
SpanDetail
  traceId
  spanId
  parentSpanId
  name
  resource
  service
  start
  durationMs
  error
```

`payloadKind` is used to distinguish source subtypes such as `rum`, `replay`,
`log`, `trace`, and `metric`. RUM Session Replay payloads may omit normal RUM
user/account/workspace context and are validated as replay segments rather than
workflow RUM events.

## Validation Result

```text
ValidationResult
  status
  rules
  summary
```

```text
ValidationRuleResult
  ruleId
  severity
  status
  message
  fieldPath
  evidence
```

Severity:

- `info`
- `warning`
- `error`
- `fatal`

Status:

- `pass`
- `fail`
- `skipped`

## Forwarding Result

```text
ForwardingResult
  mode
  attempted
  target
  status
  statusCode
  durationMs
  retryCount
  errorClass
  errorMessage
```

## Debug Bundle

```text
DebugBundle
  bundleId
  createdAt
  filter
  summary
  events
  validationFailures
  datadogQueries
  redactionReport
```

## Storage Rules

Local mode:

- Raw decoded payload may be stored temporarily.
- Snapshots are allowed.

CI mode:

- Raw decoded payload may be included in artifacts if explicitly configured.
- Default artifact should include redacted payload.

Production modes:

- Raw payload is disabled by default.
- Headers must be redacted before storage.
- Query strings must be redacted or removed before storage.
- Secret-like values must never be persisted unmasked.
