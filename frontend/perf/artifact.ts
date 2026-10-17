import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { cpus } from "node:os";
import path from "node:path";
import { z } from "zod";

// The run-artifact contract is perf/schema/run-artifact.schema.json; the Go
// perf tool parses, validates and compresses what this module writes.
export const SCHEMA_VERSION = 2;

export const repoRoot = path.resolve(import.meta.dirname, "../..");

const workloadId = z.string().regex(/^[a-z0-9][a-z0-9-]*$/);

const workloadSchema = z.discriminatedUnion("kind", [
  z.object({ id: workloadId, kind: z.literal("http") }),
  z.object({
    id: workloadId,
    kind: z.literal("browser"),
    target: z.string().min(1),
    repetitions: z.number().int().positive(),
  }),
  z.object({ id: workloadId, kind: z.literal("build"), target: z.string() }),
]);

const registrySchema = z.object({
  version: z.literal(SCHEMA_VERSION),
  workloads: z.array(workloadSchema).min(1),
});

export type Workload = z.infer<typeof workloadSchema>;
export type BrowserWorkload = Extract<Workload, { kind: "browser" }>;
export type BuildWorkload = Extract<Workload, { kind: "build" }>;

// The Go perf tool owns full registry validation (`perf validate`); this
// parse keeps only what the frontend runner needs.
export function parseRegistry(data: string): Workload[] {
  return registrySchema.parse(JSON.parse(data)).workloads;
}

export function readRegistry(file: string): Workload[] {
  return parseRegistry(readFileSync(file, "utf8"));
}

export type Environment = {
  machine: string;
  os: string;
  arch: string;
  cpu: string;
};

export type Toolchain = Record<string, string>;

export type Sample = {
  metrics: Record<string, number>;
  status?: number;
  error?: string;
};

export type RunArtifact = {
  version: number;
  workload_id: string;
  kind: "browser" | "build";
  started_at: string;
  finished_at: string;
  environment: Environment;
  toolchain: Toolchain;
  samples: Sample[];
};

// Records compare environments by equality across producers, so Node's
// platform and architecture names are reported as Go's GOOS and GOARCH.
const goOS: Partial<Record<NodeJS.Platform, string>> = {
  win32: "windows",
  sunos: "solaris",
};
const goArch: Record<string, string> = { x64: "amd64", ia32: "386" };

export function detectEnvironment(
  machine: string,
  platform: NodeJS.Platform = process.platform,
  arch: string = process.arch,
): Environment {
  return {
    machine,
    os: goOS[platform] ?? platform,
    arch: goArch[arch] ?? arch,
    cpu: cpus()[0]?.model.trim() || "unknown",
  };
}

export type RunConfig = { runDir: string; machine: string; registry: string };

export function runConfig(
  env: Record<string, string | undefined> = process.env,
): RunConfig {
  const machine = env.PERF_MACHINE;
  const runDir = env.PERF_RUN_DIR;
  if (!machine) {
    throw new Error(
      "PERF_MACHINE is required (an opaque label, not a hostname)",
    );
  }
  if (!runDir) {
    throw new Error("PERF_RUN_DIR is required (the shared run directory)");
  }
  return {
    machine,
    runDir: path.resolve(repoRoot, runDir),
    registry: path.resolve(
      repoRoot,
      env.PERF_REGISTRY ?? "perf/workloads.json",
    ),
  };
}

// Writes the artifact as <run dir>/<workload id>.json, refusing to
// overwrite an existing artifact of the same run.
export function writeArtifact(runDir: string, artifact: RunArtifact): string {
  mkdirSync(runDir, { recursive: true });
  const file = path.join(runDir, `${artifact.workload_id}.json`);
  writeFileSync(file, `${JSON.stringify(artifact, null, 2)}\n`, {
    flag: "wx",
  });
  return file;
}
