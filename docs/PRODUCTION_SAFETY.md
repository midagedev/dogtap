# Production Safety

## Principle

Dogtap must never become a critical application dependency by accident.

Production modes are allowed only when failure behavior, retention, redaction, and forwarding policy are explicit.

## Allowed Production Modes

### Tee mode

Preferred first production shape. Datadog remains the primary telemetry destination. Dogtap receives sampled copies or metadata.

### Forward mode

Allowed only when Dogtap is deployed redundantly and failure behavior is fail-open or explicitly accepted.

### Redact-only mode

Allowed when the organization wants to enforce payload policy before Datadog ingestion. This mode has the highest operational risk and must be treated like infrastructure.

## Default Production Rules

- Raw payload persistence disabled
- Redacted metadata only
- Bounded event count
- Bounded TTL
- Sampling enabled (`0.1` default in `forward`, `tee`, and `redact-only`
  modes unless `DOGTAP_SAMPLING_RATE` is set)
- Intake queue bounded by `DOGTAP_QUEUE_MAX_IN_FLIGHT`
- Secret headers masked
- Query strings removed or redacted
- Backpressure behavior documented
- Forwarding failures counted and visible

## Sampling

Sampling controls whether Dogtap keeps an inspected local copy. It does not
change application behavior and does not suppress configured Datadog forwarding.

- `local` and `ci` default to `1.0` so tests and developer sessions are
  deterministic.
- `forward`, `tee`, and `redact-only` default to `0.1`.
- `DOGTAP_SAMPLING_RATE=0` accepts telemetry requests but stores no Dogtap
  event; configured forwarding still runs before the Dogtap copy is sampled
  out.
- Sample drops increment `dogtap_intake_sample_drops_total`.

## Queue and Backpressure

Dogtap admits only `DOGTAP_QUEUE_MAX_IN_FLIGHT` intake requests at a time. The
only supported policy is `DOGTAP_BACKPRESSURE_POLICY=drop-newest`.

- `local` and `ci`: queue-full returns a controlled `503` to make the pressure
  visible during development and CI.
- `forward`, `tee`, and `redact-only`: queue-full returns accepted with
  `reason=queue_full`, drops the Dogtap copy, and increments
  `dogtap_intake_backpressure_drops_total`.
- Queue-full drops do not persist events and do not retry later.

## Failure Behavior

Dogtap must define behavior for:

- Datadog unavailable
- Dogtap storage unavailable
- Dogtap queue full
- invalid payload
- redaction engine failure
- dashboard unavailable
- config reload failure

Recommended default:

- local: keep payload and show error
- ci: fail validation
- staging forward: forward if safe, record error
- production tee: drop Dogtap copy, do not block Datadog
- production forward: fail-open if policy allows, otherwise return controlled 5xx only for telemetry clients

Datadog unavailable:

- forwarding attempts are bounded by `DOGTAP_FORWARDING_MAX_ATTEMPTS`
- intake still returns accepted after bounded failure in production forward or
  tee mode
- event forwarding metadata records `status=dropped`,
  `errorClass=upstream_status` or `request_error`
- `dogtap_forwarding_failures_total` and `dogtap_forwarding_drops_total`
  increment

Dogtap storage unavailable:

- local and CI modes return a controlled intake error so developers see the
  local tool failure
- production forward, tee, and redact-only modes return accepted with
  `reason=storage_error`
- `dogtap_intake_storage_drops_total` increments
- no retry queue is created; the Dogtap copy is dropped

## Data Retention

Local:

- raw payload allowed
- developer-controlled

CI:

- redacted artifacts by default
- raw artifacts opt-in

Production:

- raw payload off by default
- redacted metadata only
- short TTL
- explicit audit trail if raw capture is temporarily enabled

## Security Requirements

- API keys must never be displayed.
- Authorization headers must be masked.
- Cookies must be masked.
- Email-like values must be configurable as warning or error.
- Access tokens and refresh tokens must be fatal validation failures by default.
- Debug bundles must include a redaction report.
