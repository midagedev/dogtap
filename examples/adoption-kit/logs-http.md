# Logs HTTP Intake

Dogtap accepts Datadog logs HTTP payloads at:

```text
http://localhost:8080/api/v2/logs
```

For a backend container in the same Compose project:

```text
http://dogtap:8080/api/v2/logs
```

Smoke payload:

```bash
curl -sS -X POST http://localhost:8080/api/v2/logs \
  -H 'Content-Type: application/json' \
  -d '{
    "service": "your-backend",
    "env": "local",
    "version": "local",
    "status": "info",
    "message": "dogtap log smoke",
    "trace_id": "123456789",
    "route": "GET /health"
  }'
```

For applications that already emit OTLP logs, prefer the OTLP templates instead
of adding a separate logs HTTP sender.
