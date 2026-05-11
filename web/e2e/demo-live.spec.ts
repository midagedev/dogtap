import { expect, test } from "@playwright/test";

test.skip(
  process.env.DOGTAP_LIVE_E2E !== "1",
  "live demo visual check starts Dogtap and seeds telemetry first",
);

test("seeded demo dashboard shows the public telemetry workflow", async ({
  page,
}, testInfo) => {
  await page.goto("/");

  await expect(
    page.getByRole("heading", { name: "Dogtap", exact: true }),
  ).toBeVisible();
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
    page
      .getByLabel("Browser sessions")
      .getByRole("button", { name: /session-123/ }),
  ).toBeVisible();

  const serviceMap = page.getByLabel("Interactive service map");
  await expect(
    serviceMap
      .locator(".service-graph-node")
      .filter({ hasText: "web-frontend" }),
  ).toBeVisible();
  await serviceMap
    .locator(".service-graph-node")
    .filter({ hasText: "api-service" })
    .click();
  await expect(
    page.getByLabel("Selected service details").getByText("api-service").first(),
  ).toBeVisible();
  await expect(
    page.getByLabel("Service edges").getByText("edge-gateway"),
  ).toBeVisible();
  await expect(
    page.getByLabel("Service edges").getByText("api-service"),
  ).toBeVisible();
  await expect(
    page.getByLabel("Metric samples").getByText("http.server.request.duration"),
  ).toBeVisible();

  await page.getByPlaceholder("Filter payloads").fill("replay");
  await page.locator(".event-row").filter({ hasText: "replay" }).first().click();
  await expect(
    page.getByRole("heading", { name: "Session Replay" }),
  ).toBeVisible();
  await expect(page.getByText("DOM replay")).toBeVisible();
  const replayFrame = page.frameLocator(".replay-dom-stage iframe").first();
  await expect(replayFrame.getByText("Case #123")).toBeVisible();
  await expect(replayFrame.getByRole("button", { name: "Export" })).toBeVisible();
  await page.screenshot({
    path: testInfo.outputPath(`dogtap-demo-replay-${testInfo.project.name}.png`),
    fullPage: true,
  });

  await page.getByPlaceholder("Filter payloads").fill("case export failed");
  await page.locator(".event-row").filter({ hasText: "log" }).first().click();
  await expect(page.getByRole("heading", { name: "Log Viewer" })).toBeVisible();
  await expect(page.locator(".log-viewer").getByText("case export failed")).toBeVisible();

  await page.getByPlaceholder("Filter payloads").fill("case.export.render");
  await page.locator(".event-row").filter({ hasText: "trace" }).first().click();
  await expect(page.getByRole("heading", { name: "Trace Spans" })).toBeVisible();
  await expect(
    page.locator(".trace-viewer").getByText("case.export.render"),
  ).toBeVisible();

  await page.getByPlaceholder("Filter payloads").fill("http.server.request.duration");
  await page.locator(".event-row").first().click();
  await expect(page.getByRole("heading", { name: "Metric Viewer" })).toBeVisible();
  await expect(page.locator(".metric-detail-list").getByText("42.5 ms")).toBeVisible();

  await page.getByPlaceholder("Filter payloads").fill("");
  await page.getByRole("tab", { name: /Failures/ }).click();
  await page.getByLabel("Failure rule").selectOption("required.rum.userId");
  const failureRow = page
    .locator(".event-row")
    .filter({ hasText: "required.rum.userId" })
    .first();
  await expect(failureRow).toBeVisible();
  await failureRow.click();
  await expect(
    page.getByRole("button").filter({ hasText: "required.rum.userId" }),
  ).toBeVisible();
  await expect(page.getByRole("heading", { name: "RUM detail" })).toBeVisible();
  await expect(
    page.locator(".rules").getByText("required.rum.workspaceId"),
  ).toBeVisible();

  await page.screenshot({
    path: testInfo.outputPath(`dogtap-demo-${testInfo.project.name}.png`),
    fullPage: true,
  });
});
