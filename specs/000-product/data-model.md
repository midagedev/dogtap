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
  rawBody
  decoded
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
- `faro`
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
  metrics
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
  normalizedRef
```

```text
MetricEntry
  name
  service
  unit
  value
  aggregation
  route
  timestamp
```

`payloadKind` is used to distinguish source subtypes such as `rum`, `replay`,
`log`, `trace`, and `metric`. RUM Session Replay payloads may omit normal RUM
user/account/workspace context and are validated as replay segments rather than
workflow RUM events.

Faro SDK payloads use `source=faro` and may normalize as `event`, `log`,
`metric`, or trace-related telemetry depending on the SDK payload shape. Native
Faro intake is smoke-level; production-grade Faro routing should use Grafana
Alloy into OTLP.

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

## Diagnostics Snapshot

```text
Snapshot
  createdAt
  baseUrl
  limit
  filter
  healthz
  readyz
  events
  report
  debugBundle
  metrics
  assertions
  workflowContracts
```

```text
AssertionReport
  status
  summary
  observed
  expectations
  checks
  rootCauses
```

```text
RootCause
  id
  title
  evidence
  nextChecks
  relatedChecks
```

Diagnostics archives contain the same evidence as the API response, split into
agent-readable files such as `summary.md`, `assertions.json`, optional
`workflow-contracts.json`, `events.json`, `report.json`, `debug-bundle.json`,
`metrics.txt`, `healthz.json`, `readyz.json`, and `manifest.json`.

## Workflow Contract

```text
Definition
  schema
  name
  description
  labels
  checks
```

`schema` maps to optional `$schema` in YAML/JSON contract files and is used only
as an editor hint.

```text
CheckDefinition
  id
  type
  description
  source
  payloadKind
  service
  route
  routeRegex
  metric
  pattern
  fields
  from
  to
  hint
```

Supported check types:

- `event`
- `log-message`
- `metric`
- `trace-correlation`
- `no-sensitive-values`

```text
ContractResult
  name
  description
  status
  summary
  checks
```

```text
ContractCheckResult
  id
  type
  status
  message
  matched
  eventIds
  traceIds
  selectors
  description
  hint
```

```text
ContractSelectorResult
  label
  criteria
  pattern
  metric
  matched
  eventIds
  alternatives
```

Workflow contract failures are separate from diagnostics assertion failures
unless a caller explicitly opts into failing the CLI with
`-fail-on-workflow-contract`.

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

## Datadog Compatibility Response

```text
DatadogSearchResponse
  data
  meta
  links
```

`data[]` contains Datadog-style records for retained Dogtap telemetry:

- `type=log` for `POST /api/v2/logs/events/search`
- `type=rum` for `POST /api/v2/rum/events/search`
- `type=span` for `POST /api/v2/spans/events/search`

```text
DatadogMetricQueryResponse
  status
  res_type
  query
  from_date
  to_date
  series
```

The compatibility responses are projections of retained `EventEnvelope` data.
They do not add long-term retention, permissions, facets, indexes, cursor
pagination, or Datadog mutating APIs.

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
