import { describe, expect, it, vi } from "vitest";

vi.mock("server-only", () => ({}));

import { getBackendHealth } from "./backend";

describe("getBackendHealth", () => {
  it("fetches and validates the configured backend without caching", async () => {
    vi.stubEnv("BACKEND_URL", "http://localhost:9000");
    const fetchMock = vi
      .fn()
      .mockResolvedValue(Response.json({ status: "ok" }));
    vi.stubGlobal("fetch", fetchMock);

    expect(await getBackendHealth()).toEqual({
      available: true,
      data: { status: "ok" },
    });
    expect(fetchMock).toHaveBeenCalledWith(
      new URL("http://localhost:9000/healthz"),
      { cache: "no-store", signal: expect.any(AbortSignal) },
    );
  });

  it("uses the local default when BACKEND_URL is unset", async () => {
    vi.stubEnv("BACKEND_URL", undefined);
    const fetchMock = vi
      .fn()
      .mockResolvedValue(Response.json({ status: "ok" }));
    vi.stubGlobal("fetch", fetchMock);
    await getBackendHealth();
    expect(fetchMock.mock.calls[0][0].href).toBe(
      "http://127.0.0.1:8080/healthz",
    );
  });

  it.each([
    {},
    { status: "unhealthy" },
    { status: 1 },
    { status: null },
    null,
    ["ok"],
  ])("rejects schema-invalid data: %j", async (body) => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json(body)));
    expect(await getBackendHealth()).toEqual({ available: false });
  });

  it("rejects malformed JSON", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("not JSON")));
    expect(await getBackendHealth()).toEqual({ available: false });
  });

  it("rejects non-success responses even if the body says ok", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(Response.json({ status: "ok" }, { status: 503 })),
    );
    expect(await getBackendHealth()).toEqual({ available: false });
  });

  it("handles connection failures", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new TypeError("fetch failed")),
    );
    expect(await getBackendHealth()).toEqual({ available: false });
  });

  it("handles an invalid backend address", async () => {
    vi.stubEnv("BACKEND_URL", "not a URL");
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    expect(await getBackendHealth()).toEqual({ available: false });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("aborts an unresponsive backend within the timeout", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        (_url, { signal }: RequestInit) =>
          new Promise((_resolve, reject) => {
            signal?.addEventListener("abort", () => reject(signal.reason), {
              once: true,
            });
          }),
      ),
    );
    expect(await getBackendHealth()).toEqual({ available: false });
  });
});
