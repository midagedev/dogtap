# Production Deployment Runbook

## Default Stance

Dogtap should be introduced in production only as a bounded, removable
telemetry sidecar. The first production shape should be tee mode, not forward
mode, unless the owning team explicitly accepts the added operational risk.

## Required Configuration

- `DOGTAP_MODE=tee` or `DOGTAP_MODE=forward`
- `DOGTAP_FORWARDING_ENABLED=true` only when outbound forwarding is intended
- `DOGTAP_FORWARDING_SITE=<datadog site>`
- `DOGTAP_FORWARDING_API_KEY` or `DD_API_KEY` supplied from the secret manager
- `DOGTAP_ALLOW_RAW_PAYLOADS=false`
- `DOGTAP_STORAGE_MAX_EVENTS` sized to a short diagnostic window
- `DOGTAP_STORAGE_TTL` set to a short retention period
- `DOGTAP_STORAGE_KIND=sqlite` only when bounded restart persistence is
  required; keep `memory` for the smallest production-facing diagnostic tap
- `DOGTAP_SAMPLING_RATE` set explicitly for the production diagnostic window
- `DOGTAP_QUEUE_MAX_IN_FLIGHT` sized to bound concurrent Dogtap work
- `DOGTAP_BACKPRESSURE_POLICY=drop-newest`

Do not put API keys in YAML files. Keep them in process environment or the
deployment secret mechanism.

## Safety Checks

Before enabling production traffic:

1. Confirm `/healthz`, `/readyz`, and `/metrics` are scraped.
2. Confirm `/api/config` does not reveal secret material.
3. Send a synthetic log containing authorization, token, and email-like values,
   then verify persisted events contain only redacted values.
4. Confirm forwarding failures increment `dogtap_forwarding_failures_total` and
   `dogtap_forwarding_drops_total`.
5. Confirm payload size limits reject oversized payloads.
6. Confirm `DOGTAP_SAMPLING_RATE=0` accepts telemetry but increments
   `dogtap_intake_sample_drops_total`, stores no event, and still runs
   configured forwarding.
7. Confirm a queue-full test returns accepted in production modes with
   `reason=queue_full` and increments
   `dogtap_intake_backpressure_drops_total`.
8. Confirm storage failures in production modes return accepted with
   `reason=storage_error` and increment `dogtap_intake_storage_drops_total`.

## Failure Behavior

- Dashboard failure must not block application traffic.
- In local and CI modes, storage failure returns an intake error so the problem
  is visible.
- In production forward, tee, and redact-only modes, storage failure drops the
  Dogtap copy and returns accepted.
- Queue-full in `tee`, `forward`, and `redact-only` modes drops the Dogtap copy
  and returns accepted so telemetry clients are not blocked by Dogtap
  backpressure.
- Datadog forwarding failures are recorded in event forwarding metadata and
  metrics.
- Retry attempts are bounded; Dogtap must not retry indefinitely.

## Bypass Procedure

To remove Dogtap from the path:

1. Disable SDK proxy or agent endpoint overrides in the application deployment.
2. Point RUM, logs, APM, or OTLP exporters back to their previous Datadog or
   collector endpoints.
3. Set `DOGTAP_FORWARDING_ENABLED=false`.
4. Keep the Dogtap process online long enough to export final debug bundles if
   needed.
5. Stop Dogtap and retain only reviewed redacted artifacts.

## Temporary Raw Capture

Raw production payload capture is disabled by default. If it must be enabled
for a short incident window, record:

- owner
- exact reason
- start and end time
- affected service
- cleanup confirmation

Disable raw capture immediately after the incident window and delete unreviewed
raw artifacts.
