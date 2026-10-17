import { readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import { gzipSync } from "node:zlib";
import type { Sample } from "./artifact";

// Raw values read from the browser's Performance APIs after one page load.
// Fields are null when the browser did not report the entry.
export type PageTimings = {
  ttfbMs: number | null;
  fcpMs: number | null;
  lcpMs: number | null;
  htmlBytes: number | null;
  jsBytes: number;
  cssBytes: number;
};

// Runs inside the page, so it must not reference anything outside its body.
// It waits for buffered paint entries, bounded by timeoutMs, and sums the
// decoded sizes of the scripts and stylesheets loaded so far.
export function collectPageTimings(timeoutMs: number): Promise<PageTimings> {
  const firstEntry = (type: string, name?: string) =>
    new Promise<PerformanceEntry | null>((resolve) => {
      const observer = new PerformanceObserver((list) => {
        const entries = list
          .getEntries()
          .filter((e) => name === undefined || e.name === name);
        if (entries.length > 0) {
          clearTimeout(timer);
          observer.disconnect();
          resolve(entries[entries.length - 1]);
        }
      });
      const timer = setTimeout(() => {
        observer.disconnect();
        resolve(null);
      }, timeoutMs);
      observer.observe({ type, buffered: true });
    });

  return Promise.all([
    firstEntry("paint", "first-contentful-paint"),
    firstEntry("largest-contentful-paint"),
  ]).then(([fcp, lcp]) => {
    const nav = performance.getEntriesByType("navigation")[0] as
      PerformanceNavigationTiming | undefined;
    const sizes = new Map<string, number>();
    for (const entry of performance.getEntriesByType("resource")) {
      const resource = entry as PerformanceResourceTiming;
      sizes.set(resource.name, resource.decodedBodySize);
    }
    let jsBytes = 0;
    let cssBytes = 0;
    for (const [name, size] of sizes) {
      const pathname = new URL(name).pathname;
      if (pathname.endsWith(".js")) jsBytes += size;
      else if (pathname.endsWith(".css")) cssBytes += size;
    }
    return {
      ttfbMs: nav ? nav.responseStart - nav.startTime : null,
      fcpMs: fcp ? fcp.startTime : null,
      lcpMs: lcp ? lcp.startTime : null,
      htmlBytes: nav ? nav.decodedBodySize : null,
      jsBytes,
      cssBytes,
    };
  });
}

// Converts one page load into a run-artifact sample. Missing entries and
// non-2xx documents are recorded as sample errors; the values that were
// measured still count, as for HTTP samples.
export function browserSample(status: number, t: PageTimings): Sample {
  const metrics: Record<string, number> = {};
  const errors: string[] = [];
  const record = (metric: string, value: number | null, entry: string) => {
    if (value === null || !Number.isFinite(value)) errors.push(`no ${entry}`);
    else metrics[metric] = value;
  };
  record("ttfb_ms", t.ttfbMs, "navigation timing");
  record("fcp_ms", t.fcpMs, "first-contentful-paint");
  record("lcp_ms", t.lcpMs, "largest-contentful-paint");
  if (t.htmlBytes !== null) metrics.html_bytes = t.htmlBytes;
  metrics.js_bytes = t.jsBytes;
  metrics.css_bytes = t.cssBytes;
  if (status < 200 || status > 299) errors.unshift(`http status ${status}`);
  const sample: Sample = { metrics, status };
  if (errors.length > 0) sample.error = errors.join("; ");
  return sample;
}

function filesUnder(dir: string): string[] {
  return readdirSync(dir, { recursive: true, encoding: "utf8" })
    .map((name) => path.join(dir, name))
    .filter((file) => statSync(file).isFile())
    .sort();
}

// Measures the client assets of a Next.js production build: raw and gzip
// bytes of every JavaScript and CSS file under <build dir>/static. Sizes are
// deterministic for a given build, so one sample is the whole measurement.
export function bundleSample(buildDir: string): Sample {
  const staticDir = path.join(buildDir, "static");
  if (!statSync(staticDir, { throwIfNoEntry: false })?.isDirectory()) {
    throw new Error(`${staticDir} not found; run the production build first`);
  }
  const metrics = {
    js_bytes: 0,
    js_gzip_bytes: 0,
    css_bytes: 0,
    css_gzip_bytes: 0,
  };
  for (const file of filesUnder(staticDir)) {
    const ext = path.extname(file);
    if (ext !== ".js" && ext !== ".css") continue;
    const data = readFileSync(file);
    const kind = ext === ".js" ? "js" : "css";
    metrics[`${kind}_bytes`] += data.length;
    metrics[`${kind}_gzip_bytes`] += gzipSync(data).length;
  }
  return { metrics };
}
