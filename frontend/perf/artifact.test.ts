import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import {
  detectEnvironment,
  parseRegistry,
  readRegistry,
  repoRoot,
  newArtifact,
  runConfig,
  SCHEMA_VERSION,
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

const contentHash = "2a66ce5990e1d520265d13ed0b712dacc43cb7b8";
const runEnv = {
  PERF_MACHINE: "m",
  PERF_RUN_DIR: "perf/runs/x",
  PERF_CONTENT_HASH: contentHash,
};

describe("runConfig", () => {
  it("requires a machine label, run directory and content hash", () => {
    expect(() => runConfig({ ...runEnv, PERF_MACHINE: undefined })).toThrow(
      /PERF_MACHINE/,
    );
    expect(() => runConfig({ ...runEnv, PERF_RUN_DIR: undefined })).toThrow(
      /PERF_RUN_DIR/,
    );
    for (const hash of [
      undefined,
      "",
      "HEAD",
      contentHash.slice(1),
      contentHash.toUpperCase(),
    ]) {
      expect(() => runConfig({ ...runEnv, PERF_CONTENT_HASH: hash })).toThrow(
        /PERF_CONTENT_HASH/,
      );
    }
  });

  it("accepts SHA-256 content hashes", () => {
    const sha256 = "a".repeat(64);
    expect(
      runConfig({ ...runEnv, PERF_CONTENT_HASH: sha256 }).contentHash,
    ).toBe(sha256);
  });

  it("resolves paths from the repository root", () => {
    const config = runConfig(runEnv);
    expect(config.runDir).toBe(path.join(repoRoot, "perf/runs/x"));
    expect(config.registry).toBe(path.join(repoRoot, "perf/workloads.json"));
  });
});

describe("writeArtifact", () => {
  let dir = "";
  afterEach(() => rmSync(dir, { recursive: true, force: true }));

  it("writes <id>.json once, with the run's content hash, and refuses to overwrite it", () => {
    dir = mkdtempSync(path.join(tmpdir(), "perf-"));
    const environment = detectEnvironment("m");
    const artifact: RunArtifact = newArtifact(runConfig(runEnv), environment, {
      workload_id: "frontend-root",
      kind: "browser",
      started_at: "2026-10-17T00:00:00.000Z",
      finished_at: "2026-10-17T00:00:01.000Z",
      toolchain: { node: "v26.11.1" },
      samples: [{ metrics: { ttfb_ms: 1 }, status: 200 }],
    });
    const runDir = path.join(dir, "run");
    const file = writeArtifact(runDir, artifact);
    const written = JSON.parse(readFileSync(file, "utf8"));
    expect(written).toEqual(artifact);
    expect(written).toMatchObject({
      version: SCHEMA_VERSION,
      content_hash: contentHash,
      environment,
    });
    expect(() => writeArtifact(runDir, artifact)).toThrow();
  });
});
