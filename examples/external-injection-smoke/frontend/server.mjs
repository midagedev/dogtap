import http from "node:http";
import { deflateSync } from "node:zlib";

const port = Number(process.env.PORT || 3000);
const service = process.env.DD_SERVICE || process.env.SERVICE_NAME || "external-smoke-frontend";
const env = process.env.DD_ENV || process.env.ENVIRONMENT || "ci";
const version = process.env.DD_VERSION || process.env.VERSION || "base";
const backendURL = process.env.BACKEND_URL || "http://backend:3001";

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
    headers: {
      "content-type": "application/json",
      "x-datadog-origin": "rum",
    },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    throw new Error(`${url} returned ${response.status}`);
  }
}

async function postMultipartReplay(url, metadata, records) {
  const encodedRecords = Buffer.from(JSON.stringify(records));
  const compressedSegment = deflateSync(encodedRecords);
  const form = new FormData();

  form.append(
    "event",
    JSON.stringify({
      ...metadata,
      records_count: records.length,
      raw_segment_size: encodedRecords.byteLength,
      compressed_segment_size: compressedSegment.byteLength,
    }),
  );
  form.append(
    "segment",
    new Blob([compressedSegment], { type: "application/octet-stream" }),
    `${metadata.session.id}-${metadata.start}`,
  );

  const response = await fetch(url, {
    method: "POST",
    headers: {
      "x-datadog-origin": "rum",
    },
    body: form,
  });
  if (!response.ok) {
    throw new Error(`${url} returned ${response.status}`);
  }
}

async function emitRUM() {
  const proxy = process.env.DATADOG_RUM_PROXY_URL;
  if (!proxy) return false;

  await postJSON(`${proxy}?ddforward=/api/v2/rum`, {
    service,
    env,
    version,
    type: "view",
    usr: {
      id: "external-user-1",
    },
    context: {
      account: { id: "external-account-1" },
      workspace: { id: "external-workspace-1" },
      case: { id: "external-case-1" },
    },
    session: {
      id: "external-session-1",
    },
    view: {
      id: "external-view-1",
      url_path: "/external-injection",
    },
    action: {
      target: {
        name: "Run external injection workflow",
      },
    },
  });
  return true;
}

async function emitReplay() {
  const proxy = process.env.DATADOG_RUM_PROXY_URL;
  if (!proxy) return false;

  const records = [
    {
      type: 4,
      timestamp: 1778206500000,
      data: {
        href: "http://external-smoke.local/external-injection",
        width: 1024,
        height: 720,
      },
    },
    {
      type: 2,
      timestamp: 1778206500100,
      data: {
        node: {
          type: 0,
          id: 1,
          childNodes: [
            { type: 1, id: 2, name: "html", publicId: "", systemId: "" },
            {
              type: 2,
              id: 3,
              tagName: "html",
              attributes: {},
              childNodes: [
                {
                  type: 2,
                  id: 4,
                  tagName: "head",
                  attributes: {},
                  childNodes: [
                    {
                      type: 2,
                      id: 5,
                      tagName: "style",
                      attributes: {},
                      childNodes: [
                        {
                          type: 3,
                          id: 6,
                          textContent:
                            "body{margin:0;font:16px system-ui;background:#f8fafc;color:#172026}.workflow{padding:28px}.run{border:0;border-radius:6px;background:#2563eb;color:white;padding:10px 14px}",
                        },
                      ],
                    },
                  ],
                },
                {
                  type: 2,
                  id: 7,
                  tagName: "body",
                  attributes: {},
                  childNodes: [
                    {
                      type: 2,
                      id: 8,
                      tagName: "main",
                      attributes: { class: "workflow" },
                      childNodes: [
                        {
                          type: 2,
                          id: 9,
                          tagName: "h1",
                          attributes: {},
                          childNodes: [
                            { type: 3, id: 10, textContent: "External Injection Workflow" },
                          ],
                        },
                        {
                          type: 2,
                          id: 11,
                          tagName: "button",
                          attributes: { class: "run" },
                          childNodes: [
                            { type: 3, id: 12, textContent: "Run workflow" },
                          ],
                        },
                      ],
                    },
                  ],
                },
              ],
            },
          ],
        },
        initialOffset: { top: 0, left: 0 },
      },
    },
    {
      type: 5,
      timestamp: 1778206500200,
      data: {
        tagName: "run-workflow-button",
      },
    },
  ];

  await postMultipartReplay(
    `${proxy}?ddforward=/api/v2/replay`,
    {
      service,
      env,
      version,
      session: { id: "external-session-1" },
      view: { id: "external-view-1" },
      start: "1778206500000",
      end: "1778206500200",
    },
    records,
  );
  return true;
}

async function emitMissingContextRUM() {
  const proxy = process.env.DATADOG_RUM_PROXY_URL;
  if (!proxy) return false;

  await postJSON(`${proxy}?ddforward=/api/v2/rum`, {
    service,
    env,
    version,
    type: "view",
    session: {
      id: "external-session-missing-context",
    },
    view: {
      id: "external-view-missing-context",
      url_path: "/external-injection/missing-context",
    },
  });
  return true;
}

async function exercise(_req, res) {
  const frontendTelemetry = {
    rum: await emitRUM(),
    replay: await emitReplay(),
    missingContext: await emitMissingContextRUM(),
  };
  const backendResponse = await fetch(`${backendURL}/api/workflow`);
  const backend = await backendResponse.json();
  if (!backendResponse.ok) {
    json(res, 500, { frontendTelemetry, backend });
    return;
  }
  json(res, 200, {
    service,
    env,
    version,
    frontendTelemetry,
    frontendTelemetryEnabled: Object.values(frontendTelemetry).some(Boolean),
    backend,
  });
}

const server = http.createServer((req, res) => {
  if (req.url === "/healthz") {
    json(res, 200, { status: "ok", service });
    return;
  }
  if (req.url === "/exercise") {
    exercise(req, res).catch((err) => {
      json(res, 500, { error: err.message });
    });
    return;
  }
  if (req.url === "/") {
    const html = `<!doctype html>
<html>
  <head><title>Dogtap external injection smoke</title></head>
  <body>
    <h1>Dogtap external injection smoke</h1>
    <button id="run">Run workflow</button>
    <pre id="result"></pre>
    <script>
      document.getElementById("run").onclick = async () => {
        const response = await fetch("/exercise");
        document.getElementById("result").textContent = await response.text();
      };
    </script>
  </body>
</html>`;
    res.writeHead(200, {
      "content-type": "text/html",
      "content-length": Buffer.byteLength(html),
    });
    res.end(html);
    return;
  }
  json(res, 404, { error: "not found" });
});

server.listen(port, "0.0.0.0", () => {
  console.log(`frontend listening on ${port}`);
});
