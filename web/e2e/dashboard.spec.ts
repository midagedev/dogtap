import { expect, type Page, test } from "@playwright/test";

const events = [
  {
    id: "evt-rum-replay",
    receivedAt: "2026-05-08T08:15:00Z",
    source: "rum",
    payloadKind: "replay",
    endpoint: "/datadog-intake-proxy",
    method: "POST",
    decoded: {
      replay: {
        format: "multipart",
        contentType: "multipart/form-data",
        bytes: 512,
        recordCount: 3,
        segmentBytes: 280,
      },
      records: [
        {
          type: 4,
          timestamp: 1778206500000,
          data: {
            href: "http://localhost/cloud/cases/case-123",
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
                                "body{margin:0;font:16px system-ui;background:#f8fafc;color:#172026}.case{padding:28px}.toolbar{display:flex;gap:8px;margin-top:18px}.export{border:0;border-radius:6px;background:#2563eb;color:white;padding:10px 14px}",
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
                          attributes: { class: "case" },
                          childNodes: [
                            {
                              type: 2,
                              id: 9,
                              tagName: "h1",
                              attributes: {},
                              childNodes: [
                                { type: 3, id: 10, textContent: "Case #123" },
                              ],
                            },
                            {
                              type: 2,
                              id: 11,
                              tagName: "p",
                              attributes: {},
                              childNodes: [
                                {
                                  type: 3,
                                  id: 12,
                                  textContent: "Workspace replay preview",
                                },
                              ],
                            },
                            {
                              type: 2,
                              id: 13,
                              tagName: "div",
                              attributes: { class: "toolbar" },
                              childNodes: [
                                {
                                  type: 2,
                                  id: 14,
                                  tagName: "button",
                                  attributes: { class: "export" },
                                  childNodes: [
                                    { type: 3, id: 15, textContent: "Export" },
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
              ],
            },
            initialOffset: { top: 0, left: 0 },
          },
        },
        {
          type: 3,
          timestamp: 1778206500200,
          data: { source: 2, text: "export button click" },
        },
      ],
    },
    details: {
      replay: {
        format: "multipart",
        bytes: 512,
        recordCount: 3,
        segmentBytes: 280,
        sessionId: "session-123",
        viewId: "view-123",
      },
    },
    normalized: {
      source: "rum",
      service: "web-frontend",
      env: "local",
      version: "dev",
      sessionId: "session-123",
      viewId: "view-123",
      userId: "user-123",
      accountId: "account-123",
      workspaceId: "workspace-123",
      caseId: "case-123",
      route: "/cases/case-123",
    },
    validation: {
      status: "pass",
      summary: "passed",
      rules: [],
    },
  },
  {
    id: "evt-rum-missing",
    receivedAt: "2026-05-08T08:15:01Z",
    source: "rum",
    endpoint: "/rum",
    method: "POST",
    rawBody: '{"type":"view"}',
    decoded: { type: "view" },
    normalized: {
      source: "rum",
      service: "web-frontend",
      env: "local",
      version: "dev",
      route: "/cases/missing-context",
      viewId: "view-missing-context",
    },
    validation: {
      status: "fail",
      summary: "3 validation failures",
      rules: [
        {
          ruleId: "required.rum.user.id",
          severity: "error",
          status: "fail",
          message: "Missing required user context",
          fieldPath: "user.id",
        },
        {
          ruleId: "required.rum.workspace.id",
          severity: "error",
          status: "fail",
          message: "Missing required workspace context",
          fieldPath: "workspace.id",
        },
      ],
    },
  },
  {
    id: "evt-log-export",
    receivedAt: "2026-05-08T08:15:02Z",
    source: "logs",
    payloadKind: "log",
    endpoint: "/api/v2/logs",
    method: "POST",
    decoded: {
      status: "error",
      message: "case export failed",
      trace_id: "123456789",
      span_id: "987654321",
      route: "/api/cases/{caseId}/exports",
      status_code: 500,
      account_id: "account-123",
    },
    details: {
      logs: [
        {
          timestamp: "2026-05-08T08:15:02Z",
          level: "ERROR",
          message: "case export failed",
          traceId: "123456789",
        },
      ],
    },
    normalized: {
      source: "logs",
      service: "api-service",
      env: "local",
      version: "dev",
      traceId: "123456789",
      spanId: "987654321",
      userId: "user-123",
      workspaceId: "workspace-123",
      caseId: "case-123",
      route: "/api/cases/{caseId}/exports",
      statusCode: 500,
    },
    validation: {
      status: "pass",
      summary: "passed",
      rules: [],
    },
  },
  {
    id: "evt-rum-resource-export",
    receivedAt: "2026-05-08T08:15:02.500Z",
    source: "rum",
    payloadKind: "event",
    endpoint: "/datadog-intake-proxy",
    method: "POST",
    decoded: {
      type: "resource",
      _dd: { trace_id: "123456789", span_id: "555555555" },
      resource: {
        method: "POST",
        status_code: 500,
        url: "https://localhost:8080/api/cases/case-123/exports?token=redacted",
      },
    },
    normalized: {
      source: "rum",
      service: "web-frontend",
      env: "local",
      version: "dev",
      traceId: "123456789",
      spanId: "555555555",
      sessionId: "session-123",
      userId: "user-123",
      accountId: "account-123",
      workspaceId: "workspace-123",
      caseId: "case-123",
      route: "/api/cases/{caseId}/exports",
      method: "POST",
      statusCode: 500,
    },
    validation: {
      status: "pass",
      summary: "passed",
      rules: [],
    },
  },
  {
    id: "evt-apm-export",
    receivedAt: "2026-05-08T08:15:03Z",
    source: "apm",
    payloadKind: "trace",
    endpoint: "/v0.5/traces",
    method: "POST",
    decoded: [
      [
        {
          trace_id: "000000000000000000000000075bcd15",
          span_id: "987654321",
          parent_id: "00000000211d1ae3",
          name: "web.request",
          resource: "POST /api/cases/{caseId}/exports",
          service: "edge-gateway",
          duration: 32000000,
        },
        {
          trace_id: "000000000000000000000000075bcd15",
          span_id: "987654322",
          parent_id: "987654321",
          name: "case.export.render",
          resource: "render export",
          service: "api-service",
          duration: 12000000,
        },
      ],
    ],
    details: {
      trace: {
        traceId: "000000000000000000000000075bcd15",
        spans: [
          {
            traceId: "000000000000000000000000075bcd15",
            spanId: "987654321",
            parentSpanId: "00000000211d1ae3",
            name: "web.request",
            resource: "POST /api/cases/{caseId}/exports",
            service: "edge-gateway",
            durationMs: 32,
          },
          {
            traceId: "000000000000000000000000075bcd15",
            spanId: "987654322",
            parentSpanId: "987654321",
            name: "case.export.render",
            resource: "render export",
            service: "api-service",
            durationMs: 12,
          },
        ],
      },
    },
    normalized: {
      source: "apm",
      service: "api-service",
      env: "local",
      version: "dev",
      traceId: "000000000000000000000000075bcd15",
      spanId: "987654321",
      route: "POST /api/cases/{caseId}/exports",
    },
    validation: {
      status: "pass",
      summary: "passed",
      rules: [],
    },
  },
  {
    id: "evt-otlp-metric",
    receivedAt: "2026-05-08T08:15:04Z",
    source: "otlp",
    payloadKind: "metric",
    endpoint: "/v1/metrics",
    method: "POST",
    decoded: {
      resourceMetrics: [
        {
          scopeMetrics: [
            {
              metrics: [
                {
                  name: "http.server.request.duration",
                  unit: "ms",
                  gauge: {
                    dataPoints: [
                      {
                        asDouble: 48.5,
                        attributes: [
                          {
                            key: "http.route",
                            value: {
                              stringValue: "/api/cases/{caseId}/exports",
                            },
                          },
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
    },
    details: {
      metrics: [
        {
          name: "http.server.request.duration",
          service: "api-service",
          unit: "ms",
          value: 48.5,
          aggregation: "gauge",
          route: "/api/cases/{caseId}/exports",
          timestamp: "2026-05-08T08:15:04Z",
        },
        {
          name: "http.server.request.duration",
          service: "api-service",
          unit: "ms",
          value: 64.25,
          aggregation: "gauge",
          route: "/api/cases/{caseId}/exports",
          timestamp: "2026-05-08T08:15:05Z",
        },
      ],
    },
    normalized: {
      source: "otlp",
      service: "api-service",
      env: "local",
      version: "dev",
      route: "/api/cases/{caseId}/exports",
      statusCode: 200,
    },
    validation: {
      status: "pass",
      summary: "passed",
      rules: [],
    },
  },
  {
    id: "evt-rum-login",
    receivedAt: "2026-05-08T08:15:05Z",
    source: "rum",
    endpoint: "/rum",
    method: "POST",
    decoded: { type: "view" },
    normalized: {
      source: "rum",
      service: "web-frontend",
      env: "local",
      version: "dev",
      sessionId: "session-123",
      userId: "user-123",
      accountId: "account-123",
      workspaceId: "workspace-123",
      caseId: "case-123",
      route: "/cases/case-123",
    },
    validation: {
      status: "pass",
      summary: "passed",
      rules: [],
    },
  },
  {
    id: "evt-faro-workflow",
    receivedAt: "2026-05-08T08:15:06Z",
    source: "faro",
    payloadKind: "event",
    endpoint: "/collect/faro-smoke",
    method: "POST",
    decoded: {
      meta: {
        app: {
          name: "faro-smoke-frontend",
          version: "dev",
          environment: "local",
        },
        session: { id: "faro-session-1" },
      },
      events: [{ name: "faro.workflow.run" }],
    },
    normalized: {
      source: "faro",
      service: "faro-smoke-frontend",
      env: "local",
      version: "dev",
      sessionId: "faro-session-1",
      userId: "faro-user-1",
      accountId: "faro-account-1",
      workspaceId: "faro-workspace-1",
      caseId: "faro-case-1",
      route: "/faro",
    },
    validation: {
      status: "pass",
      summary: "passed",
      rules: [],
    },
  },
];

async function mockDashboardApi(page: Page, nextEvents = events) {
  const nextFailures = nextEvents.filter(
    (event) => event.validation.status === "fail",
  );
  await page.route("**/api/events?*", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(nextEvents),
    });
  });
  await page.route("**/api/validation/failures", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(nextFailures),
    });
  });
  await page.route("**/api/reports/latest", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        summary: {
          total: nextEvents.length,
          passed: nextEvents.length - nextFailures.length,
          failed: nextFailures.length,
          fatal: 0,
          warnings: 0,
        },
      }),
    });
  });
  await page.route("**/api/diagnostics", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        workflowContracts: [
          {
            name: "frontend-backend-readiness",
            description: "Checks browser and backend telemetry.",
            status: "fail",
            summary: { total: 5, passed: 3, failed: 2 },
            checks: [
              {
                id: "browser-session-context",
                type: "event",
                status: "pass",
                message: "Observed 1 matching event(s).",
                matched: 1,
                eventIds: ["evt-rum-login"],
              },
              {
                id: "browser-to-api-trace",
                type: "trace-correlation",
                status: "pass",
                message: "Observed 1 correlated trace event(s).",
                matched: 1,
                eventIds: ["evt-apm-export"],
                traceIds: ["000000000000000000000000075bcd15"],
              },
              {
                id: "workflow-metric",
                type: "metric",
                status: "pass",
                message: "Observed 1 matching metric event(s).",
                matched: 1,
                eventIds: ["evt-otlp-metric"],
              },
              {
                id: "missing-rum-context",
                type: "event",
                status: "fail",
                message: "Expected workflow telemetry was not observed.",
                matched: 0,
                hint: "Check Browser RUM context setters.",
                selectors: [
                  {
                    criteria: {
                      source: "rum",
                      route: "/cases/case-123",
                      fields: ["sessionId", "userId"],
                    },
                    matched: 0,
                    alternatives: [
                      {
                        eventId: "evt-rum-missing",
                        source: "rum",
                        payloadKind: "event",
                        service: "web-frontend",
                        route: "/cases/missing-context",
                        presentFields: ["sessionId"],
                        missingFields: ["userId"],
                        differences: [
                          "route expected /cases/case-123, saw /cases/missing-context",
                        ],
                      },
                    ],
                  },
                ],
              },
              {
                id: "no-sensitive-values",
                type: "no-sensitive-values",
                status: "fail",
                message: "Found 1 event(s) with obvious sensitive values.",
                matched: 1,
                eventIds: ["evt-rum-missing"],
                hint: "Review RUM context before forwarding telemetry.",
              },
            ],
          },
        ],
      }),
    });
  });
}

test("dashboard renders stream detail, failure inbox, correlation, and query builder", async ({
  page,
}) => {
  await mockDashboardApi(page);
  await page.goto("/");

  await expect(
    page.getByRole("heading", { name: "Dogtap", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByLabel("Validation summary").getByText("Received"),
  ).toBeVisible();
  await expect(page.getByLabel("Apply Dogtap to an app")).toHaveCount(0);
  await expect(
    page.getByRole("heading", { name: "Service Map" }),
  ).toBeVisible();
  await expect(page.getByRole("heading", { name: "Traffic" })).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Metrics Snapshot" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Intake Health" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Session Timeline" }),
  ).toBeVisible();
  await expect(
    page.getByLabel("Intake sources").getByText("faro"),
  ).toBeVisible();
  await expect(
    page.getByLabel("Intake endpoints").getByText("/collect/faro-smoke"),
  ).toBeVisible();
  await expect(
    page
      .getByLabel("Browser sessions")
      .getByRole("button", { name: /session-123/ }),
  ).toBeVisible();
  await expect(
    page.getByLabel("Browser session timeline").getByText("case export failed"),
  ).toBeVisible();
  await expect(
    page.getByLabel("Service edges").getByText("edge-gateway").first(),
  ).toBeVisible();
  await expect(
    page.getByLabel("Service edges").getByText("api-service").first(),
  ).toBeVisible();
  await expect(
    page.getByLabel("Service edges").getByText("web-frontend").first(),
  ).toBeVisible();
  await expect(
    page.getByLabel("Metric samples").getByText("http.server.request.duration"),
  ).toBeVisible();
  await expect(
    page.getByRole("img", {
      name: "http.server.request.duration retained metric chart",
    }),
  ).toBeVisible();
  await expect(page.getByLabel("Metric samples").getByText("2 samples")).toBeVisible();
  const workflowContracts = page.getByLabel("Workflow contract diagnostics");
  await expect(
    workflowContracts.getByText("browser-session-context", { exact: true }),
  ).toBeVisible();
  await expect(
    workflowContracts.getByText("browser-to-api-trace", { exact: true }),
  ).toBeVisible();
  await expect(
    workflowContracts.getByText("no-sensitive-values", { exact: true }),
  ).toBeVisible();
  await expect(
    workflowContracts.getByText("missing-rum-context", { exact: true }),
  ).toBeVisible();
  await expect(workflowContracts.getByText("Evaluated selector")).toBeVisible();
  await expect(workflowContracts.getByText("source: rum")).toBeVisible();
  await expect(
    workflowContracts.getByText("Closest alternatives"),
  ).toBeVisible();
  await expect(
    workflowContracts
      .locator(".workflow-alternative-list")
      .getByRole("button", { name: /evt-rum-missing/ }),
  ).toBeVisible();
  await expect(
    workflowContracts.getByRole("button", { name: "evt-rum-login" }),
  ).toBeVisible();
  await workflowContracts.getByRole("button", { name: "evt-apm-export" }).click();
  await expect(
    page.getByRole("heading", { name: "Trace Spans" }),
  ).toBeVisible();
  await expect(
    page
      .locator(".event-list")
      .getByRole("button")
      .filter({ hasText: "/cases/missing-context" }),
  ).toBeVisible();
  await expect(page.getByText("Payload")).toBeVisible();

  await page
    .getByRole("button")
    .filter({ hasText: "/cases/case-123" })
    .first()
    .click();
  await expect(
    page.getByRole("heading", { name: "Session Replay" }),
  ).toBeVisible();
  await expect(page.getByText("DOM replay")).toBeVisible();
  await expect(page.getByLabel("DOM replay position")).toBeVisible();
  const replayFrame = page.frameLocator(".replay-dom-stage iframe").first();
  await expect(replayFrame.getByText("Case #123")).toBeVisible();
  await expect(
    replayFrame.getByRole("button", { name: "Export" }),
  ).toBeVisible();

  await page.getByRole("tab", { name: /Failures/ }).click();
  await expect(
    page.getByLabel("Validation failure inbox filters"),
  ).toBeVisible();
  await page.getByLabel("Failure rule").selectOption("required.rum.user.id");
  await expect(
    page.getByRole("button").filter({ hasText: "required.rum.user.id" }),
  ).toBeVisible();

  await page.getByRole("tab", { name: /Events/ }).click();
  await page.getByPlaceholder("Filter payloads").fill("case export failed");
  await page.locator(".event-row").first().click();
  await expect(
    page.getByRole("heading", { name: "Correlation" }),
  ).toBeVisible();
  await expect(page.getByText("trace spans 3 sources")).toBeVisible();
  await expect(page.getByText("2 peers").first()).toBeVisible();
  await expect(page.getByRole("heading", { name: "Log Viewer" })).toBeVisible();
  await expect(
    page.locator(".log-viewer").getByText("case export failed"),
  ).toBeVisible();
  const logFields = page.getByLabel("Structured log fields");
  await expect(logFields.getByText("route")).toBeVisible();
  await expect(
    logFields.getByText("/api/cases/{caseId}/exports"),
  ).toBeVisible();
  await expect(logFields.getByText("500")).toBeVisible();
  await expect(logFields.getByText("987654321")).toBeVisible();

  const query = page.getByLabel("Datadog search query");
  await expect(query).toHaveValue(/service:api-service/);
  await expect(query).toHaveValue(/trace_id:123456789/);
  await expect(query).toHaveValue(/@workspace.id:workspace-123/);

  await page.getByRole("button", { name: "Copy" }).click();
  await expect(page.getByRole("button", { name: "Copied" })).toBeVisible();

  await page
    .getByPlaceholder("Filter payloads")
    .fill("POST /api/cases/{caseId}/exports");
  await page.locator(".event-row").first().click();
  await expect(
    page.getByRole("heading", { name: "Trace Spans" }),
  ).toBeVisible();
  await expect(
    page.locator(".trace-viewer").getByText("case.export.render"),
  ).toBeVisible();

  await page
    .getByPlaceholder("Filter payloads")
    .fill("http.server.request.duration");
  await page.locator(".event-row").first().click();
  await expect(
    page.getByRole("heading", { name: "Metric Viewer" }),
  ).toBeVisible();
  await expect(page.getByLabel("Metric summary")).toBeVisible();
  await expect(
    page.locator(".metric-detail-list").getByText("48.5 ms"),
  ).toBeVisible();
  await expect(
    page.locator(".metric-detail-list").getByText("64.25 ms"),
  ).toBeVisible();
});

test("empty dashboard shows generic adoption targets", async ({ page }) => {
  await mockDashboardApi(page, []);
  await page.goto("/");

  const setup = page.getByLabel("Apply Dogtap to an app");
  await expect(
    setup.getByRole("heading", { name: "Apply Dogtap" }),
  ).toBeVisible();
  await expect(setup.getByText("Browser RUM")).toBeVisible();
  await expect(setup.getByText(/datadog-intake-proxy/)).toBeVisible();
  await expect(setup.getByText("APM")).toBeVisible();
  await expect(setup.getByText(/DD_TRACE_AGENT_PORT=8126/)).toBeVisible();
  await expect(setup.getByText("OTLP HTTP")).toBeVisible();
  await expect(setup.getByText(/4318/)).toBeVisible();
  await expect(page.getByText("No telemetry received yet.")).toBeVisible();
});
