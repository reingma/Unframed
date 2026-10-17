import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    include: ["src/**/*.test.{ts,tsx}", "perf/**/*.test.ts"],
    environment: "node",
    restoreMocks: true,
    unstubEnvs: true,
    unstubGlobals: true,
  },
});
