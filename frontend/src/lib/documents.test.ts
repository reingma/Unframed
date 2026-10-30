import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";
import type { z } from "zod";

import {
  decodeDocument,
  envelopeSchema,
  manifestSchema,
  type NavNode,
  type Redirect,
} from "./documents";

// Go's golden documents keep the Zod schemas and the Go encoder in step.
function golden(name: string): Record<string, unknown> {
  const url = new URL(
    `../../../backend/internal/document/testdata/${name}`,
    import.meta.url,
  );
  return JSON.parse(readFileSync(url, "utf8"));
}

const schemas: [string, z.ZodType<unknown>, Record<string, unknown>][] = [
  ["envelope", envelopeSchema, golden("envelope.json")],
  ["manifest", manifestSchema, golden("manifest.json")],
];

describe.each(schemas)("%s schema", (_, schema, document) => {
  it("decodes Go's golden document", () => {
    expect(decodeDocument(schema, document)).toEqual({
      ok: true,
      data: document,
    });
  });

  it.each([undefined, 0, 2, "1", 1.5, null])(
    "rejects schemaVersion %j as unsupported",
    (schemaVersion) => {
      expect(decodeDocument(schema, { ...document, schemaVersion })).toEqual({
        ok: false,
        reason: "unsupported-schema-version",
      });
    },
  );

  it.each([null, [], "text", 1])("rejects non-object JSON %j", (json) => {
    expect(decodeDocument(schema, json)).toEqual({
      ok: false,
      reason: "invalid-document",
    });
  });

  it("drops unknown members", () => {
    const decoded = decodeDocument(schema, { ...document, extra: true });
    expect(decoded).toEqual({ ok: true, data: document });
  });
});

describe("envelope schema", () => {
  const envelope = golden("envelope.json");
  const withMember = (path: string[], value: unknown) => {
    const copy = structuredClone(envelope);
    let target = copy as Record<string, unknown>;
    for (const key of path.slice(0, -1))
      target = target[key] as Record<string, unknown>;
    target[path.at(-1)!] = value;
    return copy;
  };

  it.each([
    [["body", "kind"], "markdown"],
    [["body", "content"], undefined],
    [["metadata", "title"], " "],
    [["metadata", "urlPath"], "relative"],
    [["metadata", "draft"], "false"],
    [["resolvedConfig", "theme", "accent"], "red"],
    [["resolvedConfig", "theme", "text"], "#12345"],
    [["resolvedConfig", "theme", "background"], "#fff;}body{color:red"],
  ])("rejects %j = %j", (path, value) => {
    expect(decodeDocument(envelopeSchema, withMember(path, value))).toEqual({
      ok: false,
      reason: "invalid-document",
    });
  });
});

describe("manifest schema", () => {
  const manifest = golden("manifest.json") as {
    nav: NavNode;
    redirects: Redirect[];
  };
  const [redirect] = manifest.redirects;

  it("treats a manifest without redirects as having none", () => {
    const older: Partial<typeof manifest> = { ...manifest };
    delete older.redirects;
    expect(decodeDocument(manifestSchema, older)).toEqual({
      ok: true,
      data: { ...older, redirects: [] },
    });
  });

  it("drops unknown redirect members", () => {
    const redirects = [{ ...redirect, status: 302 }];
    expect(decodeDocument(manifestSchema, { ...manifest, redirects })).toEqual({
      ok: true,
      data: { ...manifest, redirects: [redirect] },
    });
  });

  it.each([
    ["an external source", [{ ...redirect, from: "https://evil.example" }]],
    ["a scheme-relative source", [{ ...redirect, from: "//evil.example" }]],
    ["a root source", [{ ...redirect, from: "/" }]],
    ["an uppercase source", [{ ...redirect, from: "/Old" }]],
    ["a relative target", [{ ...redirect, to: "mathematics" }]],
    ["a missing prefix flag", [{ from: redirect.from, to: redirect.to }]],
    ["a repeated source", [redirect, redirect]],
    ["redirects that are not a list", {}],
  ])("rejects %s", (_, redirects) => {
    expect(decodeDocument(manifestSchema, { ...manifest, redirects })).toEqual({
      ok: false,
      reason: "invalid-document",
    });
  });

  function withNav(nav: unknown) {
    return { ...manifest, nav };
  }

  // tree builds a nested directory chain, checking recursion at any depth.
  function tree(depth: number, leaf: NavNode): NavNode {
    let node = leaf;
    for (let i = depth; i > 0; i--) {
      node = {
        kind: "directory",
        urlPath: i === 0 ? "/" : `/d${i}`,
        title: `D${i}`,
        description: "",
        hasIndex: false,
        children: [node],
      };
    }
    return { ...node, urlPath: "/" };
  }

  const article: NavNode = {
    kind: "article",
    urlPath: "/a",
    title: "A",
    description: "",
    hasIndex: false,
    children: [],
  };

  it.each([1, 2, 8])("accepts a tree %i levels deep", (depth) => {
    expect(
      decodeDocument(manifestSchema, withNav(tree(depth, article))).ok,
    ).toBe(true);
  });

  it.each([
    ["an article root", { ...manifest.nav, kind: "article", children: [] }],
    ["a root that is not /", { ...manifest.nav, urlPath: "/blog" }],
    ["an article with children", tree(2, { ...article, children: [article] })],
    ["an article with an index", tree(2, { ...article, hasIndex: true })],
    ["an unknown node kind", tree(2, { ...article, kind: "asset" } as never)],
    ["a node without a title", tree(3, { ...article, title: "" })],
  ])("rejects %s", (_, nav) => {
    expect(decodeDocument(manifestSchema, withNav(nav))).toEqual({
      ok: false,
      reason: "invalid-document",
    });
  });
});
