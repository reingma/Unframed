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

  it("matches the deepest alias for every path, in any order", () => {
    const segments = ["a", "b"];
    const paths = ["/a", "/b", "/a/a", "/a/b", "/b/a", "/a/a/b", "/a/b/a"];
    const all = paths.map((from, i) => ({
      from,
      to: `/t${i}`,
      prefix: i % 3 !== 0,
    }));
    let seed = 50;
    const random = () => (seed = (seed * 48271) % 2147483647) / 2147483647;
    for (let trial = 0; trial < 200; trial++) {
      const chosen = all
        .filter(() => random() < 0.5)
        .sort(() => random() - 0.5);
      for (const depth of [1, 2, 3, 4]) {
        const parts = Array.from(
          { length: depth },
          () => segments[Math.floor(random() * segments.length)],
        );
        const urlPath = "/" + parts.join("/");
        const target = resolveRedirect(chosen, urlPath);
        const deepest = deepestMatch(chosen, urlPath);
        expect(target).toBe(
          deepest && deepest.to + urlPath.slice(deepest.from.length),
        );
      }
    }
  });

  // Random redirect maps built under Go's rules: a prefix-closed set of
  // published paths, sources anywhere unpublished (including below published
  // directories, where chains form) and published targets. Every chain ends
  // at a path that does not redirect, within one hop per segment of the
  // requested path plus one.
  it("follows every chain to its end within the hop bound", () => {
    const segments = ["a", "b", "c"];
    let seed = 50;
    const random = () => (seed = (seed * 48271) % 2147483647) / 2147483647;
    const pick = <T>(items: readonly T[]) =>
      items[Math.floor(random() * items.length)];
    const randomPath = (maxDepth: number) =>
      "/" +
      Array.from({ length: 1 + Math.floor(random() * maxDepth) }, () =>
        pick(segments),
      ).join("/");
    let chains = 0;
    for (let trial = 0; trial < 500; trial++) {
      const published = new Set<string>();
      for (const path of [randomPath(3), randomPath(3)]) {
        for (let at = path; at; at = at.slice(0, at.lastIndexOf("/"))) {
          published.add(at);
        }
      }
      const targets = [...published];
      const bySource = new Map<string, Redirect>();
      for (let i = 0; i < 12; i++) {
        const from = randomPath(4);
        if (!published.has(from)) {
          bySource.set(from, {
            from,
            to: pick(targets),
            prefix: random() < 0.7,
          });
        }
      }
      const chosen = [...bySource.values()];
      for (let i = 0; i < 20; i++) {
        const urlPath = randomPath(6);
        const bound = urlPath.split("/").length;
        let hops = 0;
        let current = urlPath;
        for (
          let match = deepestMatch(chosen, current);
          match && hops <= bound;
          match = deepestMatch(chosen, current)
        ) {
          current = match.to + current.slice(match.from.length);
          hops++;
        }
        expect(hops).toBeLessThanOrEqual(bound);
        expect(resolveRedirect(chosen, urlPath)).toBe(
          hops ? current : undefined,
        );
        if (hops > 1) chains++;
      }
    }
    expect(chains).toBeGreaterThan(100);
  });
});

// deepestMatch is the reference for one redirect hop: the exact or prefix
// redirect with the longest source matching urlPath.
function deepestMatch(
  redirects: readonly Redirect[],
  urlPath: string,
): Redirect | undefined {
  return redirects
    .filter(
      (r) =>
        urlPath === r.from || (r.prefix && urlPath.startsWith(r.from + "/")),
    )
    .sort((x, y) => y.from.length - x.from.length)[0];
}

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
