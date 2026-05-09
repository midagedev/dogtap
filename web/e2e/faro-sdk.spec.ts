import { expect, test } from "@playwright/test";

test.skip(
  process.env.DOGTAP_FARO_E2E !== "1",
  "Faro SDK smoke starts Dogtap and the integration frontend first",
);

const frontendURL =
  process.env.DOGTAP_FARO_FRONTEND_URL ?? "http://127.0.0.1:3000";
const dogtapURL =
  process.env.DOGTAP_FARO_DOGTAP_URL ?? "http://127.0.0.1:8080";

test("integration frontend sends Faro SDK telemetry to Dogtap", async ({
  page,
}) => {
  await page.goto(`${frontendURL}/faro`);

  await expect(
    page.getByRole("heading", { name: "Faro SDK workflow" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Run Faro workflow" }).click();
  await expect(page.getByText("Faro SDK telemetry sent")).toBeVisible();

  await expect
    .poll(async () => {
      const response = await page.request.get(
        `${dogtapURL}/api/events?source=faro&limit=100`,
      );
      if (!response.ok()) return "";
      const events = (await response.json()) as Array<{
        source?: string;
        payloadKind?: string;
        normalized?: {
          service?: string;
          env?: string;
          version?: string;
          userId?: string;
          accountId?: string;
          workspaceId?: string;
          caseId?: string;
          sessionId?: string;
          route?: string;
        };
        details?: {
          logs?: Array<{ message?: string }>;
          metrics?: Array<{ name?: string; value?: number }>;
        };
      }>;
      const event = events.find(
        (item) =>
          item.payloadKind === "event" &&
          item.normalized?.service === "faro-smoke-frontend" &&
          item.normalized?.env === "local" &&
          item.normalized?.version === "dev" &&
          item.normalized?.userId === "faro-user-1" &&
          item.normalized?.accountId === "faro-account-1" &&
          item.normalized?.workspaceId === "faro-workspace-1" &&
          item.normalized?.caseId === "faro-case-1" &&
          item.normalized?.sessionId === "faro-session-1" &&
          item.normalized?.route === "/faro",
      );
      const log = events.find(
        (item) => item.details?.logs?.[0]?.message === "Faro workflow log",
      );
      const metric = events.find((item) =>
        item.details?.metrics?.some(
          (metric) =>
            metric.name === "faro.workflow.duration" && metric.value === 42.5,
        ),
      );
      return event && log && metric ? "ready" : "waiting";
    })
    .toBe("ready");
});
