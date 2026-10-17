import path from "node:path";
import { test, type Browser } from "@playwright/test";
import {
  detectEnvironment,
  readRegistry,
  repoRoot,
  runConfig,
  SCHEMA_VERSION,
  writeArtifact,
  type BrowserWorkload,
  type BuildWorkload,
  type Sample,
} from "./artifact";
import { browserSample, bundleSample, collectPageTimings } from "./metrics";

// Measures the registry's browser and build workloads into raw run artifacts.
// Workloads run serially so measurements do not compete for the machine.
const config = runConfig();
const workloads = readRegistry(config.registry);
const environment = detectEnvironment(config.machine);
const baseURL = "http://127.0.0.1:13090";
const paintTimeoutMs = 5_000;

test.describe.configure({ mode: "serial" });

// Each page load uses a fresh context: no cache, cookies or storage carry
// over, so every repetition measures a first visit.
async function measurePageLoad(browser: Browser, url: string) {
  const context = await browser.newContext();
  try {
    const page = await context.newPage();
    const response = await page.goto(url, { waitUntil: "load" });
    const timings = await page.evaluate(collectPageTimings, paintTimeoutMs);
    return browserSample(response?.status() ?? 0, timings);
  } finally {
    await context.close();
  }
}

function runBrowserWorkload(w: BrowserWorkload) {
  test(w.id, async ({ browser }) => {
    test.setTimeout(30_000 + w.repetitions * 15_000);
    const url = new URL(w.target, baseURL).toString();
    // One discarded load warms the server and browser processes.
    await measurePageLoad(browser, url);
    const startedAt = new Date();
    const samples: Sample[] = [];
    for (let i = 0; i < w.repetitions; i++) {
      samples.push(await measurePageLoad(browser, url));
    }
    writeArtifact(config.runDir, {
      version: SCHEMA_VERSION,
      workload_id: w.id,
      kind: "browser",
      started_at: startedAt.toISOString(),
      finished_at: new Date().toISOString(),
      environment,
      toolchain: { node: process.version, chromium: browser.version() },
      samples,
    });
  });
}

function runBuildWorkload(w: BuildWorkload) {
  test(w.id, async () => {
    const buildDir = path.resolve(repoRoot, w.target);
    if (path.relative(repoRoot, buildDir).startsWith("..")) {
      throw new Error(`build target ${w.target} is outside the repository`);
    }
    const startedAt = new Date();
    const sample = bundleSample(buildDir);
    writeArtifact(config.runDir, {
      version: SCHEMA_VERSION,
      workload_id: w.id,
      kind: "build",
      started_at: startedAt.toISOString(),
      finished_at: new Date().toISOString(),
      environment,
      toolchain: { node: process.version },
      samples: [sample],
    });
  });
}

for (const w of workloads) {
  if (w.kind === "browser") runBrowserWorkload(w);
  else if (w.kind === "build") runBuildWorkload(w);
}
