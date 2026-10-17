import { defineConfig } from "@playwright/test";

// Performance measurement, not a test suite: one worker, no retries, and
// production servers on ports separate from the integration suite. Make
// targets in the repository root build both applications first.
export default defineConfig({
  testDir: "./perf",
  testMatch: "*.perf.ts",
  workers: 1,
  retries: 0,
  reporter: "list",
  use: {
    browserName: "chromium",
    trace: "off",
  },
  webServer: [
    {
      command: "../backend/bin/server",
      url: "http://127.0.0.1:18090/healthcheck",
      env: { BACKEND_ADDR: "127.0.0.1:18090" },
      reuseExistingServer: false,
      gracefulShutdown: { signal: "SIGTERM", timeout: 5_000 },
    },
    {
      command: "npm run start -- --hostname 127.0.0.1 --port 13090",
      url: "http://127.0.0.1:13090",
      env: { BACKEND_URL: "http://127.0.0.1:18090" },
      reuseExistingServer: false,
    },
  ],
});
