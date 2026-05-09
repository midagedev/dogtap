# OpenTelemetry Collector Tee

Use this pattern when an application already sends OTLP to an OpenTelemetry
Collector and Datadog should remain the primary telemetry destination.

The Collector receives application OTLP once and exports to two destinations:

```text
app OTLP SDK
  -> OpenTelemetry Collector
     -> Datadog exporter                 primary path
     -> OTLP HTTP exporter to Dogtap     sampled inspection path
```

## Files

- `otel-collector-tee.yaml`: Collector config with Datadog primary pipelines and
  Dogtap secondary pipelines
- `compose.otel-collector-tee.yaml`: Compose wrapper for local or staging
  experiments

## Run

Validate the Collector config without starting the pipeline:

```bash
docker run --rm \
  -v "$PWD/examples/adoption-kit/otel-collector-tee.yaml:/etc/otelcol-contrib/config.yaml:ro" \
  -e DD_API_KEY=dummy \
  -e DD_SITE=datadoghq.com \
  -e DOGTAP_OTLP_HTTP_ENDPOINT=http://dogtap:4318 \
  -e DOGTAP_TEE_TRACE_SAMPLING_PERCENTAGE=10 \
  otel/opentelemetry-collector-contrib:0.151.0 \
  validate --config=/etc/otelcol-contrib/config.yaml
```

Validate the Compose wrapper:

```bash
DD_API_KEY=dummy \
DD_SITE=datadoghq.com \
docker compose \
  -f examples/adoption-kit/compose.otel-collector-tee.yaml \
  config
```

Start the tee stack:

```bash
DD_API_KEY=... \
DD_SITE=datadoghq.com \
docker compose \
  -f examples/adoption-kit/compose.otel-collector-tee.yaml \
  up
```

Point applications at the Collector, not Dogtap:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
```

Dogtap is then removable by deleting the Dogtap exporter pipelines from the
Collector config or by returning to the previous Collector config file.

## Sampling And Safety

The sample keeps Datadog primary unsampled by this template. Only the Dogtap
trace pipeline uses `probabilistic_sampler/dogtap_traces`.

```bash
DOGTAP_TEE_TRACE_SAMPLING_PERCENTAGE=10
```

Dogtap also runs with bounded retention:

```bash
DOGTAP_MODE=tee
DOGTAP_SAMPLING_RATE=0.1
DOGTAP_ALLOW_RAW_PAYLOADS=false
```

The units differ deliberately: the Collector sampler uses a percentage
(`10` means 10%), while Dogtap's own sampling rate uses a fraction (`0.1` means
10%).

Keep these defaults unless a specific staging investigation needs higher local
retention. Do not use this as the only production telemetry route without a
separate safety review.

## Current Boundary

- This is an OpenTelemetry Collector bridge pattern, not a Datadog Agent
  replacement.
- DogStatsD and Datadog Agent integrations remain on the Datadog Agent path.
- Collector-level sampling in this template applies only to traces. Dogtap's
  own sampling and retention limits bound local storage for all received
  signals.
