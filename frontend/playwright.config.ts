import { defineConfig } from "@playwright/test";

import {
  BACKEND,
  PREVIEW_TOKEN,
  SITE,
  SITE_WITHOUT_BACKEND,
} from "./e2e/servers";

const port = (url: string) => new URL(url).port;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? "github" : "list",
  use: {
    browserName: "chromium",
    trace: "retain-on-failure",
  },
  webServer: [
    {
      command: "../backend/bin/server",
      url: `${BACKEND}/healthcheck`,
      env: {
        BACKEND_ADDR: new URL(BACKEND).host,
        CONTENT_ROOT: "./e2e/content",
        PREVIEW_TOKEN,
      },
      reuseExistingServer: false,
      gracefulShutdown: { signal: "SIGTERM", timeout: 5_000 },
    },
    {
      command: `npm run start -- --hostname 127.0.0.1 --port ${port(SITE)}`,
      url: SITE,
      env: { BACKEND_URL: BACKEND },
      reuseExistingServer: false,
    },
    {
      // Every page of this server fails, so readiness is its startup line.
      command: `npm run start -- --hostname 127.0.0.1 --port ${port(SITE_WITHOUT_BACKEND)}`,
      wait: { stdout: /Ready in/ },
      env: { BACKEND_URL: "http://127.0.0.1:18081" },
      reuseExistingServer: false,
    },
  ],
});
