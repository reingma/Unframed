import { describe, expect, it } from "vitest";

import type { NavNode, Redirect } from "./documents";
import {
  findNode,
  isPreviewToken,
  pageTitle,
  previewBase,
  resolveRedirect,
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

describe("resolveRedirect", () => {
  const redirects: Redirect[] = [
    { from: "/old", to: "/new", prefix: true },
    { from: "/old/deep", to: "/elsewhere", prefix: true },
    { from: "/old/deep/page", to: "/page", prefix: false },
    { from: "/renamed", to: "/guides/alpha", prefix: false },
  ];

  it.each([
    ["/renamed", "/guides/alpha"],
    ["/old", "/new"],
    ["/old/a/b", "/new/a/b"],
    ["/old/deep", "/elsewhere"],
    ["/old/deep/x", "/elsewhere/x"],
    ["/old/deep/page", "/page"],
    ["/old/deep/page/x", "/elsewhere/page/x"],
  ])("redirects %s to %s", (urlPath, target) => {
    expect(resolveRedirect(redirects, urlPath)).toBe(target);
  });

  it.each(["/renamed/x", "/olden", "/ol", "/", "/new", "/guides/alpha"])(
    "does not redirect %s",
    (urlPath) => {
      expect(resolveRedirect(redirects, urlPath)).toBeUndefined();
    },
  );

  it.each([
    ["/older/a", "/new/a"],
    ["/older/deep/x", "/elsewhere/x"],
    ["/new/old/old/x", "/new/x"],
  ])("follows the chain from %s to %s in one result", (urlPath, target) => {
    const chained: Redirect[] = [
      ...redirects,
      { from: "/older", to: "/old", prefix: true },
      { from: "/new/old", to: "/new", prefix: true },
    ];
    expect(resolveRedirect(chained, urlPath)).toBe(target);
  });

  // Random redirect maps over a prefix-closed set of published paths, with
  // sources that are unpublished and targets that are published, as Go
  // builds them: every chain ends at a path no redirect matches.
  it("ends every chain at a path that does not redirect", () => {
    const segments = ["a", "b", "c"];
    let seed = 50;
    const random = () => (seed = (seed * 48271) % 2147483647) / 2147483647;
    const pick = <T>(items: readonly T[]) =>
      items[Math.floor(random() * items.length)];
    const randomPath = (depth: number) =>
      "/" + Array.from({ length: depth }, () => pick(segments)).join("/");
    for (let trial = 0; trial < 300; trial++) {
      const published = new Set<string>();
      for (let i = 0; i < 6; i++) {
        const parts = randomPath(1 + Math.floor(random() * 3)).split("/");
        for (let n = 2; n <= parts.length; n++) {
          published.add(parts.slice(0, n).join("/"));
        }
      }
      const targets = [...published];
      const chosen: Redirect[] = [];
      for (let i = 0; i < 8; i++) {
        const from = randomPath(1 + Math.floor(random() * 4));
        if (published.has(from) || chosen.some((r) => r.from === from)) {
          continue;
        }
        let ancestor = from;
        while (ancestor && !published.has(ancestor)) {
          ancestor = ancestor.slice(0, ancestor.lastIndexOf("/"));
        }
        if (ancestor) continue;
        chosen.push({ from, to: pick(targets), prefix: random() < 0.7 });
      }
      for (let i = 0; i < 10; i++) {
        const urlPath = randomPath(1 + Math.floor(random() * 6));
        const target = resolveRedirect(chosen, urlPath);
        if (target !== undefined) {
          expect(resolveRedirect(chosen, target)).toBeUndefined();
        }
      }
    }
  });
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
