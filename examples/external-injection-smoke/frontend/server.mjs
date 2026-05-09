import http from "node:http";

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

async function exercise(_req, res) {
  const frontendTelemetry = await emitRUM();
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
