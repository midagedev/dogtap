import http from "node:http";

const port = Number(process.env.PORT || 3001);
const service = process.env.DD_SERVICE || process.env.SERVICE_NAME || "external-smoke-backend";
const env = process.env.DD_ENV || process.env.ENVIRONMENT || "ci";
const version = process.env.DD_VERSION || process.env.VERSION || "base";

// Datadog APM JSON traces use numeric IDs; OTLP below uses fixed-width hex.
const traceId = "123456789";
const spanId = "987654321";

function json(res, status, body) {
  const encoded = JSON.stringify(body);
  res.writeHead(status, {
    "content-type": "application/json",
    "content-length": Buffer.byteLength(encoded),
  });
  res.end(encoded);
}

async function postJSON(url, body) {
  const response = await fetch(url, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    throw new Error(`${url} returned ${response.status}`);
  }
}

async function emitLog() {
  if (!process.env.DOGTAP_LOGS_URL) return false;
  await postJSON(process.env.DOGTAP_LOGS_URL, {
    service,
    env,
    version,
    status: "info",
    message: "external injection backend handled workflow",
    trace_id: traceId,
    span_id: spanId,
    route: "GET /api/workflow",
  });
  return true;
}

async function emitAPMTrace() {
  if (!process.env.DD_TRACE_AGENT_URL) return false;
  await postJSON(`${process.env.DD_TRACE_AGENT_URL}/v0.5/traces`, [
    [
      {
        service,
        env,
        version,
        name: "web.request",
        resource: "GET /api/workflow",
        trace_id: Number(traceId),
        span_id: Number(spanId),
        parent_id: 0,
        meta: {
          "http.method": "GET",
          "http.route": "/api/workflow",
        },
      },
    ],
  ]);
  return true;
}

async function emitOTLPTrace() {
  if (!process.env.OTEL_EXPORTER_OTLP_TRACES_ENDPOINT) return false;
  await postJSON(process.env.OTEL_EXPORTER_OTLP_TRACES_ENDPOINT, {
    resourceSpans: [
      {
        resource: {
          attributes: [
            { key: "service.name", value: { stringValue: service } },
            { key: "deployment.environment", value: { stringValue: env } },
            { key: "service.version", value: { stringValue: version } },
          ],
        },
        scopeSpans: [
          {
            spans: [
              {
                traceId: "000000000000000000000000075bcd15",
                spanId: "000000003ade68b1",
                name: "external-injection.backend.workflow",
                attributes: [
                  { key: "http.route", value: { stringValue: "GET /api/workflow" } },
                ],
              },
            ],
          },
        ],
      },
    ],
  });
  return true;
}

async function emitOTLPMetric() {
  if (!process.env.OTEL_EXPORTER_OTLP_METRICS_ENDPOINT) return false;
  await postJSON(process.env.OTEL_EXPORTER_OTLP_METRICS_ENDPOINT, {
    resourceMetrics: [
      {
        resource: {
          attributes: [
            { key: "service.name", value: { stringValue: service } },
            { key: "deployment.environment", value: { stringValue: env } },
            { key: "service.version", value: { stringValue: version } },
          ],
        },
        scopeMetrics: [
          {
            metrics: [
              {
                name: "external_injection.workflow.duration",
                unit: "ms",
                gauge: {
                  dataPoints: [
                    {
                      asDouble: 24.5,
                      attributes: [
                        { key: "http.route", value: { stringValue: "GET /api/workflow" } },
                      ],
                    },
                  ],
                },
              },
            ],
          },
        ],
      },
    ],
  });
  return true;
}

async function handleWorkflow(_req, res) {
  console.log(JSON.stringify({
    service,
    env,
    version,
    status: "info",
    message: "external injection workflow reached backend",
    trace_id: traceId,
  }));

  const emitted = {
    log: await emitLog(),
    apm: await emitAPMTrace(),
    otlpTrace: await emitOTLPTrace(),
    otlpMetric: await emitOTLPMetric(),
  };

  json(res, 200, {
    service,
    env,
    version,
    telemetryEnabled: Object.values(emitted).some(Boolean),
    emitted,
  });
}

const server = http.createServer((req, res) => {
  if (req.url === "/healthz") {
    json(res, 200, { status: "ok", service });
    return;
  }
  if (req.url === "/api/workflow") {
    handleWorkflow(req, res).catch((err) => {
      json(res, 500, { error: err.message });
    });
    return;
  }
  json(res, 404, { error: "not found" });
});

server.listen(port, "0.0.0.0", () => {
  console.log(`backend listening on ${port}`);
});
