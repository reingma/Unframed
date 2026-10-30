import { z } from "zod";

// Schemas for the versioned documents Go serves. The document contract
// (docs/specs/document-contract.md) is their canonical description. Unknown
// members are dropped; a missing or unknown schemaVersion is rejected before
// the rest of the document is read.

export const SCHEMA_VERSION = 1;

const color = z.string().regex(/^#[0-9a-f]{6}$/);
const title = z.string().trim().min(1);
const urlPath = z.string().startsWith("/");

export const themeSchema = z.object({
  accent: color,
  background: color,
  text: color,
});
export type Theme = z.infer<typeof themeSchema>;

export const envelopeSchema = z.object({
  schemaVersion: z.literal(SCHEMA_VERSION),
  metadata: z.object({
    urlPath,
    title,
    description: z.string(),
    draft: z.boolean(),
    landing: z.boolean(),
  }),
  resolvedConfig: z.object({ theme: themeSchema }),
  body: z.object({ kind: z.literal("html"), content: z.string() }),
});
export type Envelope = z.infer<typeof envelopeSchema>;

export type NavNode = {
  kind: "directory" | "article";
  urlPath: string;
  title: string;
  description: string;
  hasIndex: boolean;
  children: NavNode[];
};

export const navNodeSchema: z.ZodType<NavNode> = z
  .object({
    kind: z.enum(["directory", "article"]),
    urlPath,
    title,
    description: z.string(),
    hasIndex: z.boolean(),
    get children() {
      return z.array(navNodeSchema);
    },
  })
  .refine(
    (node) =>
      node.kind === "directory" ||
      (!node.hasIndex && node.children.length === 0),
    "an article has no index or children",
  );

export const manifestSchema = z.object({
  schemaVersion: z.literal(SCHEMA_VERSION),
  site: z.object({ title, description: z.string(), theme: themeSchema }),
  nav: navNodeSchema.refine(
    (nav) => nav.kind === "directory" && nav.urlPath === "/",
    'nav must be the root directory at "/"',
  ),
});
export type Manifest = z.infer<typeof manifestSchema>;

export type DecodeFailure = "unsupported-schema-version" | "invalid-document";

export type Decoded<T> =
  { ok: true; data: T } | { ok: false; reason: DecodeFailure };

// decodeDocument checks the schema version first, so a document of another
// version fails as unsupported rather than as malformed.
export function decodeDocument<T>(
  schema: z.ZodType<T>,
  json: unknown,
): Decoded<T> {
  if (typeof json !== "object" || json === null || Array.isArray(json)) {
    return { ok: false, reason: "invalid-document" };
  }
  if ((json as { schemaVersion?: unknown }).schemaVersion !== SCHEMA_VERSION) {
    return { ok: false, reason: "unsupported-schema-version" };
  }
  const parsed = schema.safeParse(json);
  return parsed.success
    ? { ok: true, data: parsed.data }
    : { ok: false, reason: "invalid-document" };
}

// Preview responses are author diagnostics, not versioned documents.

export const diagnosticSchema = z.object({
  path: z.string(),
  urlPath: z.string(),
  message: z.string(),
});
export type Diagnostic = z.infer<typeof diagnosticSchema>;

export const previewMissSchema = z.object({
  error: z.literal("not found"),
  urlPath,
  buildId: z.string(),
  diagnostics: z.array(diagnosticSchema),
});
export type PreviewMiss = z.infer<typeof previewMissSchema>;

export const diagnosticsSchema = z.object({
  serving: z.object({
    buildId: z.string(),
    builtAt: z.string().optional(),
    envelopes: z.number().int(),
    validationErrors: z.number().int(),
  }),
  diagnostics: z.array(diagnosticSchema),
  lastBuild: z.object({
    trigger: z.string(),
    signals: z.number().int(),
    started: z.string(),
    durationMs: z.number(),
    outcome: z.enum(["succeeded", "failed"]),
    buildId: z.string().optional(),
    envelopes: z.number().int(),
    validationErrors: z.number().int(),
    error: z.string().optional(),
  }),
});
export type Diagnostics = z.infer<typeof diagnosticsSchema>;
