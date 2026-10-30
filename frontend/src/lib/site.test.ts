import { describe, expect, it } from "vitest";

import type { NavNode } from "./documents";
import {
  findNode,
  isPreviewToken,
  pageTitle,
  previewBase,
  siteHref,
  themeStyle,
  urlPathFromSlug,
} from "./site";

function node(urlPath: string, children: NavNode[] = []): NavNode {
  return {
    kind: children.length ? "directory" : "article",
    urlPath,
    title: urlPath,
    description: "",
    hasIndex: false,
    children,
  };
}

const root = node("/", [
  node("/guides", [node("/guides/alpha"), node("/guides/zeta")]),
  node("/guide"),
]);

describe("urlPathFromSlug", () => {
  it.each([
    [undefined, "/"],
    [[], "/"],
    [["guides"], "/guides"],
    [["guides", "a-1"], "/guides/a-1"],
  ])("maps %j to %s", (slug, urlPath) => {
    expect(urlPathFromSlug(slug)).toBe(urlPath);
  });

  it.each([
    ["Guides"],
    ["a", ""],
    [".."],
    ["a/b"],
    ["-a"],
    ["a--b"],
    ["a_b"],
    ["%2e"],
  ])("rejects %j", (...slug) => {
    expect(urlPathFromSlug(slug)).toBeUndefined();
  });
});

describe("findNode", () => {
  it("finds every node of the tree by its URL path", () => {
    const visit = (n: NavNode): void => {
      expect(findNode(root, n.urlPath)).toBe(n);
      n.children.forEach(visit);
    };
    visit(root);
  });

  it.each(["/missing", "/guides/missing", "/guide/alpha", "/guides/"])(
    "does not find %s",
    (urlPath) => {
      expect(findNode(root, urlPath)).toBeUndefined();
    },
  );
});

describe("links", () => {
  it("keeps public links at their URL path", () => {
    expect(siteHref("", "/")).toBe("/");
    expect(siteHref("", "/guides/alpha")).toBe("/guides/alpha");
  });

  it("keeps preview links under the preview route", () => {
    const base = previewBase("t".repeat(32));
    expect(siteHref(base, "/")).toBe(`/preview/${"t".repeat(32)}/articles`);
    expect(siteHref(base, "/guides")).toBe(
      `/preview/${"t".repeat(32)}/articles/guides`,
    );
  });

  it("suffixes the site title except on the page that has it", () => {
    expect(pageTitle("Site", "Alpha")).toBe("Alpha | Site");
    expect(pageTitle("Site", "Site")).toBe("Site");
  });
});

describe("isPreviewToken", () => {
  it.each(["a".repeat(32), "A-z_0".repeat(51).slice(0, 256)])(
    "accepts %s",
    (token) => {
      expect(isPreviewToken(token)).toBe(true);
    },
  );

  it.each([
    "a".repeat(31),
    "a".repeat(257),
    "a".repeat(31) + "/",
    "a".repeat(31) + ".",
  ])("rejects %s", (token) => {
    expect(isPreviewToken(token)).toBe(false);
  });
});

describe("themeStyle", () => {
  it("sets each theme color as a CSS custom property", () => {
    expect(
      themeStyle({ accent: "#aa3300", background: "#ffffff", text: "#111111" }),
    ).toEqual({
      "--accent": "#aa3300",
      "--background": "#ffffff",
      "--text": "#111111",
    });
  });
});
