import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { browserSample, bundleSample, type PageTimings } from "./metrics";

const complete: PageTimings = {
  ttfbMs: 12.5,
  fcpMs: 40,
  lcpMs: 44,
  htmlBytes: 2000,
  jsBytes: 500_000,
  cssBytes: 500,
};

describe("browserSample", () => {
  it("records every metric of a complete page load without an error", () => {
    expect(browserSample(200, complete)).toEqual({
      status: 200,
      metrics: {
        ttfb_ms: 12.5,
        fcp_ms: 40,
        lcp_ms: 44,
        html_bytes: 2000,
        js_bytes: 500_000,
        css_bytes: 500,
      },
    });
  });

  it("keeps measured values and names what is missing", () => {
    const sample = browserSample(200, { ...complete, lcpMs: null });
    expect(sample.metrics).not.toHaveProperty("lcp_ms");
    expect(sample.metrics.fcp_ms).toBe(40);
    expect(sample.error).toBe("no largest-contentful-paint");
  });

  it.each([0, 304, 404, 500])("marks status %i as an error", (status) => {
    const sample = browserSample(status, complete);
    expect(sample.status).toBe(status);
    expect(sample.error).toMatch(new RegExp(`^http status ${status}`));
    expect(sample.metrics.ttfb_ms).toBe(12.5);
  });
});

describe("bundleSample", () => {
  let dir = "";
  afterEach(() => rmSync(dir, { recursive: true, force: true }));

  function buildDir(files: Record<string, string>) {
    dir = mkdtempSync(path.join(tmpdir(), "next-"));
    for (const [name, content] of Object.entries(files)) {
      const file = path.join(dir, "static", name);
      mkdirSync(path.dirname(file), { recursive: true });
      writeFileSync(file, content);
    }
    return dir;
  }

  it("sums raw and gzip sizes of JavaScript and CSS only, deterministically", () => {
    const js = "console.log(1);".repeat(100);
    const build = buildDir({
      "chunks/a.js": js,
      "chunks/nested/b.js": js,
      "chunks/c.css": "body{}",
      "media/font.woff2": "binary",
    });
    const sample = bundleSample(build);
    expect(sample.metrics.js_bytes).toBe(2 * js.length);
    expect(sample.metrics.css_bytes).toBe(6);
    expect(sample.metrics.js_gzip_bytes).toBeGreaterThan(0);
    expect(sample.metrics.js_gzip_bytes).toBeLessThan(sample.metrics.js_bytes);
    expect(bundleSample(build)).toEqual(sample);
  });

  it("fails when the production build is missing", () => {
    dir = mkdtempSync(path.join(tmpdir(), "next-"));
    expect(() => bundleSample(dir)).toThrow(/production build/);
  });
});
