# Datadog References

These are the first references to use while implementing Dogtap compatibility.

## RUM Proxy

Datadog Browser RUM supports proxying browser intake data through a custom endpoint.

- https://docs.datadoghq.com/real_user_monitoring/guide/proxy-rum-data/

## APM Agent API

Datadog traces are sent to the local Agent through HTTP APIs, commonly on port `8126`.

- https://docs.datadoghq.com/tracing/guide/send_traces_to_agent_by_api/

## Logs HTTP Intake

Datadog logs support HTTP intake endpoints for JSON, text, gzip, and other formats.

- https://docs.datadoghq.com/api/latest/logs/

## OTLP Intake

Datadog supports direct OpenTelemetry protocol intake.

- https://docs.datadoghq.com/opentelemetry/setup/otlp_ingest/

## Datadog Agent

The Datadog Agent is open source and includes several components.

- https://opensource.datadoghq.com/projects/agent-integrations-tracers/
- https://github.com/DataDog/datadog-agent

## dd-apm-test-agent

Datadog's APM test agent emulates APM endpoints and includes an optional Web UI.

- https://github.com/DataDog/dd-apm-test-agent

