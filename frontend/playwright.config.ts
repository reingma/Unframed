import { defineConfig } from "@playwright/test";

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
      url: "http://127.0.0.1:18080/healthz",
      env: { BACKEND_ADDR: "127.0.0.1:18080" },
      reuseExistingServer: false,
      gracefulShutdown: { signal: "SIGTERM", timeout: 5_000 },
    },
    {
      command: "npm run start -- --hostname 127.0.0.1 --port 13000",
      url: "http://127.0.0.1:13000",
      env: { BACKEND_URL: "http://127.0.0.1:18080" },
      reuseExistingServer: false,
    },
    {
      command: "npm run start -- --hostname 127.0.0.1 --port 13001",
      url: "http://127.0.0.1:13001",
      env: { BACKEND_URL: "http://127.0.0.1:18081" },
      reuseExistingServer: false,
    },
  ],
});
