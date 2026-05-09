#!/usr/bin/env bash
set -euo pipefail

base_url="${DOGTAP_DEMO_BASE_URL:-http://127.0.0.1:8080}"
apm_url="${DOGTAP_DEMO_APM_URL:-http://127.0.0.1:8126}"
otlp_url="${DOGTAP_DEMO_OTLP_URL:-http://127.0.0.1:4318}"

post_json() {
  local url="$1"
  curl -fsS -X POST "${url}" \
    -H "Content-Type: application/json" \
    --data-binary @- >/dev/null
}

post_text_json() {
  local url="$1"
  curl -fsS -X POST "${url}" \
    -H "Content-Type: text/plain;charset=UTF-8" \
    --data-binary @- >/dev/null
}

wait_for_dashboard() {
  local attempt
  for attempt in $(seq 1 60); do
    if curl -fsS "${base_url}/healthz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  echo "Dogtap did not become healthy at ${base_url}." >&2
  exit 3
}

seed_rum() {
  post_json "${base_url}/datadog-intake-proxy?ddforward=/api/v2/rum" <<'JSON'
{
  "service": "web-frontend",
  "env": "local",
  "version": "demo",
  "type": "view",
  "usr": {
    "id": "user-123"
  },
  "context": {
    "account": {
      "id": "account-123"
    },
    "workspace": {
      "id": "workspace-123"
    },
    "case": {
      "id": "case-123"
    }
  },
  "session": {
    "id": "session-123"
  },
  "view": {
    "id": "view-123",
    "url_path": "/cases/case-123"
  }
}
JSON
}

seed_failure() {
  post_json "${base_url}/datadog-intake-proxy?ddforward=/api/v2/rum" <<'JSON'
{
  "service": "web-frontend",
  "env": "local",
  "version": "demo",
  "type": "view",
  "session": {
    "id": "session-missing-context"
  },
  "view": {
    "id": "view-missing-context",
    "url_path": "/cases/missing-context"
  }
}
JSON
}

seed_replay() {
  post_text_json "${base_url}/datadog-intake-proxy?ddforward=/api/v2/replay&ddtags=env:local,service:web-frontend,version:demo" <<'JSON'
{
  "session": {
    "id": "session-123"
  },
  "view": {
    "id": "view-123"
  },
  "records": [
    {
      "type": 4,
      "timestamp": 1778206500000,
      "data": {
        "href": "http://localhost/cases/case-123"
      }
    },
    {
      "type": 2,
      "timestamp": 1778206500100,
      "data": {
        "node": {
          "type": 0,
          "childNodes": []
        }
      }
    },
    {
      "type": 5,
      "timestamp": 1778206500200,
      "data": {
        "tagName": "export-button"
      }
    }
  ]
}
JSON
}

seed_log() {
  post_json "${base_url}/api/v2/logs" <<'JSON'
{
  "service": "api-service",
  "env": "local",
  "version": "demo",
  "status": "error",
  "message": "case export failed",
  "trace_id": "123456789",
  "span_id": "987654321",
  "userId": "user-123",
  "workspaceId": "workspace-123",
  "caseId": "case-123",
  "route": "/api/cases/{caseId}/exports"
}
JSON
}

seed_trace() {
  post_json "${apm_url}/v0.5/traces" <<'JSON'
[
  [
    {
      "service": "edge-gateway",
      "env": "local",
      "version": "demo",
      "name": "web.request",
      "resource": "POST /api/cases/{caseId}/exports",
      "trace_id": "123456789",
      "span_id": "987654321",
      "parent_id": "0",
      "duration": 32000000
    },
    {
      "service": "api-service",
      "env": "local",
      "version": "demo",
      "name": "case.export.render",
      "resource": "render export",
      "trace_id": "123456789",
      "span_id": "987654322",
      "parent_id": "987654321",
      "duration": 12000000
    }
  ]
]
JSON
}

seed_metric() {
  post_json "${otlp_url}/v1/metrics" <<'JSON'
{
  "resourceMetrics": [
    {
      "resource": {
        "attributes": [
          {
            "key": "service.name",
            "value": {
              "stringValue": "api-service"
            }
          },
          {
            "key": "deployment.environment",
            "value": {
              "stringValue": "local"
            }
          },
          {
            "key": "service.version",
            "value": {
              "stringValue": "demo"
            }
          }
        ]
      },
      "scopeMetrics": [
        {
          "metrics": [
            {
              "name": "http.server.request.duration",
              "unit": "ms",
              "gauge": {
                "dataPoints": [
                  {
                    "asDouble": 42.5,
                    "attributes": [
                      {
                        "key": "http.route",
                        "value": {
                          "stringValue": "/api/cases/{caseId}/exports"
                        }
                      }
                    ]
                  }
                ]
              }
            }
          ]
        }
      ]
    }
  ]
}
JSON
}

assert_seeded() {
  local events
  events="$(curl -fsS "${base_url}/api/events?limit=100")"

  for expected in \
    '"source":"rum"' \
    '"payloadKind":"replay"' \
    '"source":"logs"' \
    '"source":"apm"' \
    '"payloadKind":"metric"' \
    '"validation":{"status":"fail"'; do
    if ! printf '%s' "${events}" | grep -q "${expected}"; then
      echo "Missing expected demo marker: ${expected}" >&2
      printf '%s\n' "${events}" >&2
      exit 1
    fi
  done
}

main() {
  wait_for_dashboard
  seed_rum
  seed_failure
  seed_replay
  seed_log
  seed_trace
  seed_metric
  assert_seeded

  echo "Dogtap demo telemetry seeded."
  echo "Dashboard: ${base_url}"
}

main "$@"
