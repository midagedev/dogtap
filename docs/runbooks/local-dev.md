# Local Development Runbook

This runbook covers working on Dogtap itself. For applying Dogtap to another
application, use `docs/runbooks/ADOPTING_DOGTAP.md`.

## Start Dogtap

From source:

```bash
make run
```

Or with a config file:

```bash
go run ./cmd/dogtap serve -config configs/generic-local.yaml
```

The generic local config stores recent events in `.dogtap/generic-events.json`,
so local sessions survive process restarts until TTL/count retention removes
them.

With Docker Compose:

```bash
docker compose up --build
```

The compose setup mounts a named `dogtap-data` volume at `/data` and writes `/data/events.json`.

Equivalent Docker shape:

```bash
docker run --rm \
  -p 8080:8080 \
  -p 8126:8126 \
  -p 4317:4317 \
  -p 4318:4318 \
  -e DOGTAP_STORAGE_KIND=file \
  -e DOGTAP_STORAGE_PATH=/data/events.json \
  -v dogtap-data:/data \
  dogtap/dogtap:latest
```

## Configure RUM

Set the Datadog RUM proxy option to:

```text
http://localhost:8080/datadog-intake-proxy
```

## Configure APM

```bash
export DD_AGENT_HOST=localhost
export DD_TRACE_AGENT_PORT=8126
export DD_ENV=local
export DD_SERVICE=api-service
export DD_VERSION=local
```

## Verify

Open:

```text
http://localhost:8080
```

Expected:

- RUM events appear after browser interaction.
- Traces appear after backend requests.
- Logs appear after log sender execution.
- Validation failures are visible if required fields are missing.

Replay bundled fixtures:

```bash
make replay
```

Run the generic adoption smoke:

```bash
make smoke-adoption
```
