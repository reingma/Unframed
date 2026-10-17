import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import {
  detectEnvironment,
  parseRegistry,
  readRegistry,
  repoRoot,
  runConfig,
  writeArtifact,
  type RunArtifact,
} from "./artifact";

describe("detectEnvironment", () => {
  it.each([
    ["linux", "x64", "linux", "amd64"],
    ["darwin", "arm64", "darwin", "arm64"],
    ["win32", "ia32", "windows", "386"],
  ] as const)("reports %s/%s as Go's %s/%s", (platform, arch, os, goArch) => {
    const env = detectEnvironment("workstation-1", platform, arch);
    expect(env).toMatchObject({ machine: "workstation-1", os, arch: goArch });
    expect(env.cpu).not.toBe("");
  });
});

describe("parseRegistry", () => {
  it("reads the committed registry's frontend workloads", () => {
    const workloads = readRegistry(path.join(repoRoot, "perf/workloads.json"));
    expect(workloads).toContainEqual(
      expect.objectContaining({ kind: "browser", target: "/" }),
    );
    expect(workloads).toContainEqual(
      expect.objectContaining({ kind: "build", target: "frontend/.next" }),
    );
  });

  it.each([
    ["an old version", { version: 1, workloads: [] }],
    [
      "a browser workload without repetitions",
      { version: 2, workloads: [{ id: "a", kind: "browser", target: "/" }] },
    ],
    [
      "an unknown kind",
      { version: 2, workloads: [{ id: "a", kind: "rpc", target: "/" }] },
    ],
  ])("rejects %s", (_, doc) => {
    expect(() => parseRegistry(JSON.stringify(doc))).toThrow();
  });
});

describe("runConfig", () => {
  it("requires a machine label and run directory", () => {
    expect(() => runConfig({ PERF_RUN_DIR: "perf/runs/x" })).toThrow(
      /PERF_MACHINE/,
    );
    expect(() => runConfig({ PERF_MACHINE: "m" })).toThrow(/PERF_RUN_DIR/);
  });

  it("resolves paths from the repository root", () => {
    const config = runConfig({
      PERF_MACHINE: "m",
      PERF_RUN_DIR: "perf/runs/x",
    });
    expect(config.runDir).toBe(path.join(repoRoot, "perf/runs/x"));
    expect(config.registry).toBe(path.join(repoRoot, "perf/workloads.json"));
  });
});

describe("writeArtifact", () => {
  let dir = "";
  afterEach(() => rmSync(dir, { recursive: true, force: true }));

  it("writes <id>.json once and refuses to overwrite it", () => {
    dir = mkdtempSync(path.join(tmpdir(), "perf-"));
    const artifact: RunArtifact = {
      version: 2,
      workload_id: "frontend-root",
      kind: "browser",
      started_at: "2026-10-17T00:00:00.000Z",
      finished_at: "2026-10-17T00:00:01.000Z",
      environment: detectEnvironment("m"),
      toolchain: { node: "v26.11.1" },
      samples: [{ metrics: { ttfb_ms: 1 }, status: 200 }],
    };
    const runDir = path.join(dir, "run");
    const file = writeArtifact(runDir, artifact);
    expect(JSON.parse(readFileSync(file, "utf8"))).toEqual(artifact);
    expect(() => writeArtifact(runDir, artifact)).toThrow();
  });
});
