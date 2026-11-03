import "server-only";

import { cache } from "react";
import type { z } from "zod";

import {
  decodeDocument,
  diagnosticsSchema,
  envelopeSchema,
  manifestSchema,
  previewMissSchema,
  type Diagnostics,
  type Envelope,
  type Manifest,
  type PreviewMiss,
} from "./documents";

export const REQUEST_ID_HEADER = "X-Request-ID";
export const BACKEND_TIMEOUT_MS = 2_000;

export type FailureReason =
  | "unavailable"
  | "http-status"
  | "invalid-json"
  | "unsupported-schema-version"
  | "invalid-document";

// BackendError is the defined failure for an unusable backend response. Its
// path never contains a preview token.
export class BackendError extends Error {
  constructor(
    readonly reason: FailureReason,
    readonly path: string,
  ) {
    super(`backend request failed: ${reason} ${path}`);
    this.name = "BackendError";
  }
}

// requestId is the page request's correlation ID, shared by every backend
// fetch the request makes and sent to Go as X-Request-ID.
export const requestId = cache((): string => crypto.randomUUID());

// logRedirect records an alias redirect with the request's correlation ID.
export function logRedirect(from: string, to: string): void {
  console.info(
    JSON.stringify({
      level: "info",
      msg: "redirect",
      request_id: requestId(),
      from,
      to,
    }),
  );
}

function fail(
  reason: FailureReason,
  logPath: string,
  id: string,
  detail?: string | number,
): BackendError {
  console.error(
    JSON.stringify({
      level: "error",
      msg: "backend request failed",
      request_id: id,
      path: logPath,
      reason,
      detail,
    }),
  );
  return new BackendError(reason, logPath);
}

type Fetched = { status: 200 | 404; body: unknown; id: string };

// fetchJSON fetches a Go route without caching. Only 200 and 404 are
// expected answers; anything else is a failure.
async function fetchJSON(path: string, logPath: string): Promise<Fetched> {
  const id = requestId();
  let response: Response;
  try {
    const baseUrl = process.env.BACKEND_URL || "http://127.0.0.1:8080";
    response = await fetch(new URL(path, baseUrl), {
      cache: "no-store",
      headers: { [REQUEST_ID_HEADER]: id },
      signal: AbortSignal.timeout(BACKEND_TIMEOUT_MS),
    });
  } catch (error) {
    throw fail("unavailable", logPath, id, (error as Error)?.name);
  }
  if (response.status !== 200 && response.status !== 404) {
    throw fail("http-status", logPath, id, response.status);
  }
  try {
    return { status: response.status, body: await response.json(), id };
  } catch {
    throw fail("invalid-json", logPath, id);
  }
}

function decode<T>(schema: z.ZodType<T>, fetched: Fetched, logPath: string): T {
  const decoded = decodeDocument(schema, fetched.body);
  if (!decoded.ok) throw fail(decoded.reason, logPath, fetched.id);
  return decoded.data;
}

function articleRoute(urlPath: string): string {
  return urlPath === "/" ? "/articles/" : `/articles${urlPath}`;
}

export const getManifest = cache(async (): Promise<Manifest> => {
  const fetched = await fetchJSON("/manifest", "/manifest");
  if (fetched.status !== 200) {
    throw fail("http-status", "/manifest", fetched.id, fetched.status);
  }
  return decode(manifestSchema, fetched, "/manifest");
});

// getArticle returns the published envelope at urlPath, or undefined when
// Go has none.
export const getArticle = cache(
  async (urlPath: string): Promise<Envelope | undefined> => {
    const path = articleRoute(urlPath);
    const fetched = await fetchJSON(path, path);
    return fetched.status === 404
      ? undefined
      : decode(envelopeSchema, fetched, path);
  },
);

export type PreviewResult =
  | { kind: "article"; envelope: Envelope }
  | { kind: "miss"; miss: PreviewMiss }
  | { kind: "denied" };

// getPreviewArticle passes a preview through to Go. A miss carries Go's
// diagnostics; a wrong or disabled token is denied with the public 404.
export const getPreviewArticle = cache(
  async (token: string, urlPath: string): Promise<PreviewResult> => {
    const route = articleRoute(urlPath);
    const logPath = `/preview/<redacted>${route}`;
    const fetched = await fetchJSON(
      `/preview/${encodeURIComponent(token)}${route}`,
      logPath,
    );
    if (fetched.status === 200) {
      return {
        kind: "article",
        envelope: decode(envelopeSchema, fetched, logPath),
      };
    }
    const miss = previewMissSchema.safeParse(fetched.body);
    return miss.success
      ? { kind: "miss", miss: miss.data }
      : { kind: "denied" };
  },
);

// getDiagnostics returns Go's build diagnostics, or undefined when the
// token is denied.
export const getDiagnostics = cache(
  async (token: string): Promise<Diagnostics | undefined> => {
    const logPath = "/preview/<redacted>/diagnostics";
    const fetched = await fetchJSON(
      `/preview/${encodeURIComponent(token)}/diagnostics`,
      logPath,
    );
    if (fetched.status === 404) return undefined;
    const parsed = diagnosticsSchema.safeParse(fetched.body);
    if (!parsed.success) throw fail("invalid-document", logPath, fetched.id);
    return parsed.data;
  },
);
