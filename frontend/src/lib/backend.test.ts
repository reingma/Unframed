import { readFileSync } from "node:fs";

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("server-only", () => ({}));

import {
  BackendError,
  getArticle,
  getDiagnostics,
  getManifest,
  getPreviewArticle,
  REQUEST_ID_HEADER,
} from "./backend";

function golden(name: string): Record<string, unknown> {
  const url = new URL(
    `../../../backend/internal/document/testdata/${name}`,
    import.meta.url,
  );
  return JSON.parse(readFileSync(url, "utf8"));
}

const envelope = golden("envelope.json");
const manifest = golden("manifest.json");
const token = "preview-token-".padEnd(40, "x");
const goRequestId = /^[A-Za-z0-9._-]{1,128}$/;

let fetchMock: ReturnType<typeof vi.fn>;
let errorLog: ReturnType<typeof vi.spyOn>;

beforeEach(() => {
  vi.stubEnv("BACKEND_URL", "http://go.test:9000");
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
  errorLog = vi.spyOn(console, "error").mockImplementation(() => {});
});

afterEach(() => {
  for (const [message] of errorLog.mock.calls) {
    expect(String(message)).not.toContain(token);
  }
});

function lastRequest(): { url: URL; init: RequestInit } {
  const [url, init] = fetchMock.mock.lastCall!;
  return { url, init };
}

function loggedFailure(): Record<string, unknown> {
  return JSON.parse(String(errorLog.mock.lastCall![0]));
}

describe("backend requests", () => {
  it("fetches uncached with a timeout and a correlation ID Go keeps", async () => {
    fetchMock.mockImplementation(async () => Response.json(manifest));
    await getManifest();
    const { url, init } = lastRequest();
    expect(url).toEqual(new URL("http://go.test:9000/manifest"));
    expect(init.cache).toBe("no-store");
    expect(init.signal).toBeInstanceOf(AbortSignal);
    const id = new Headers(init.headers).get(REQUEST_ID_HEADER);
    expect(id).toMatch(goRequestId);
  });

  it("uses the local default when BACKEND_URL is unset", async () => {
    vi.stubEnv("BACKEND_URL", undefined);
    fetchMock.mockImplementation(async () => Response.json(manifest));
    await getManifest();
    expect(lastRequest().url).toEqual(
      new URL("http://127.0.0.1:8080/manifest"),
    );
  });

  it.each([
    ["/", "/articles/"],
    ["/mathematics/on-martingales", "/articles/mathematics/on-martingales"],
  ])("maps article %s to the Go route %s", async (urlPath, route) => {
    fetchMock.mockImplementation(async () => Response.json(envelope));
    expect(await getArticle(urlPath)).toEqual(envelope);
    expect(lastRequest().url.pathname).toBe(route);
  });

  it("returns no article for a Go 404", async () => {
    fetchMock.mockImplementation(async () =>
      Response.json({ error: "not found" }, { status: 404 }),
    );
    expect(await getArticle("/missing")).toBeUndefined();
  });
});

describe("backend failures", () => {
  const failures: [string, () => Promise<Response>, string][] = [
    [
      "a connection failure",
      () => Promise.reject(new TypeError("fetch failed")),
      "unavailable",
    ],
    [
      "a timeout",
      () => Promise.reject(new DOMException("timed out", "TimeoutError")),
      "unavailable",
    ],
    [
      "a 500",
      async () => Response.json({ error: "x" }, { status: 500 }),
      "http-status",
    ],
    [
      "a 401",
      async () => Response.json({ error: "x" }, { status: 401 }),
      "http-status",
    ],
    [
      "malformed JSON",
      async () => new Response("{", { status: 200 }),
      "invalid-json",
    ],
    [
      "an unknown version",
      async () => Response.json({ ...envelope, schemaVersion: 2 }),
      "unsupported-schema-version",
    ],
    [
      "an invalid document",
      async () => Response.json({ ...envelope, body: {} }),
      "invalid-document",
    ],
  ];

  it.each(failures)(
    "rejects %s as %s and logs it",
    async (_, respond, reason) => {
      fetchMock.mockImplementation(respond);
      const error = await getArticle("/a").catch((e: unknown) => e);
      expect(error).toBeInstanceOf(BackendError);
      expect(error).toMatchObject({ reason, path: "/articles/a" });
      const log = loggedFailure();
      expect(log).toMatchObject({
        level: "error",
        reason,
        path: "/articles/a",
      });
      expect(log.request_id).toBe(
        new Headers(lastRequest().init.headers).get(REQUEST_ID_HEADER),
      );
    },
  );

  it("rejects a manifest 404", async () => {
    fetchMock.mockImplementation(async () =>
      Response.json({ error: "not found" }, { status: 404 }),
    );
    await expect(getManifest()).rejects.toMatchObject({
      reason: "http-status",
    });
  });
});

describe("preview requests", () => {
  const miss = {
    error: "not found",
    urlPath: "/a",
    buildId: "b1",
    diagnostics: [],
  };

  it("passes the token to Go's preview route", async () => {
    fetchMock.mockImplementation(async () =>
      Response.json({
        ...envelope,
        metadata: { ...(envelope.metadata as object), draft: true },
      }),
    );
    const result = await getPreviewArticle(token, "/guides/setup");
    expect(result).toMatchObject({
      kind: "article",
      envelope: { metadata: { draft: true } },
    });
    expect(lastRequest().url.pathname).toBe(
      `/preview/${token}/articles/guides/setup`,
    );
  });

  it("returns Go's diagnostics for a miss", async () => {
    fetchMock.mockImplementation(async () =>
      Response.json(miss, { status: 404 }),
    );
    expect(await getPreviewArticle(token, "/a")).toEqual({
      kind: "miss",
      miss,
    });
  });

  it("denies a 404 without diagnostics, as for a wrong token", async () => {
    fetchMock.mockImplementation(async () =>
      Response.json({ error: "not found" }, { status: 404 }),
    );
    expect(await getPreviewArticle(token, "/a")).toEqual({ kind: "denied" });
    expect(await getDiagnostics(token)).toBeUndefined();
  });

  it("never logs the token", async () => {
    fetchMock.mockImplementation(async () =>
      Response.json({ ...envelope, schemaVersion: 9 }),
    );
    await expect(getPreviewArticle(token, "/a")).rejects.toMatchObject({
      path: "/preview/<redacted>/articles/a",
    });
    fetchMock.mockImplementation(async () => Response.json({ serving: {} }));
    await expect(getDiagnostics(token)).rejects.toMatchObject({
      reason: "invalid-document",
      path: "/preview/<redacted>/diagnostics",
    });
    expect(errorLog).toHaveBeenCalledTimes(2);
  });
});
