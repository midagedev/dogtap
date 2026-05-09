import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 20_000,
  expect: {
    timeout: 5_000,
  },
  use: {
    baseURL: process.env.DOGTAP_E2E_BASE_URL ?? "http://localhost:8080",
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
    {
      name: "mobile",
      use: { ...devices["iPhone 15"] },
    },
  ],
});
