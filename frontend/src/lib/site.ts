import type { CSSProperties } from "react";

import type { NavNode, Redirect, Theme } from "./documents";

const slugPattern = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const tokenPattern = /^[A-Za-z0-9_-]{32,256}$/;

// urlPathFromSlug turns catch-all route segments into a document URL path,
// or undefined when a segment is not a lowercase slug.
export function urlPathFromSlug(
  slug: readonly string[] = [],
): string | undefined {
  if (!slug.every((segment) => slugPattern.test(segment))) return undefined;
  return "/" + slug.join("/");
}

// isPreviewToken reports whether token has the shape Go accepts, so other
// values never reach the backend.
export function isPreviewToken(token: string): boolean {
  return tokenPattern.test(token);
}

export function findNode(root: NavNode, urlPath: string): NavNode | undefined {
  if (root.urlPath === urlPath) return root;
  for (const child of root.children) {
    if (urlPath === child.urlPath || urlPath.startsWith(child.urlPath + "/")) {
      const found = findNode(child, urlPath);
      if (found) return found;
    }
  }
  return undefined;
}

// resolveRedirect returns where a path that is not published redirects, or
// undefined. An exact alias or a directory alias at or above the path
// matches, keeping the path's suffix below it; the deepest match wins. The
// result is followed through further aliases to its final path. No target is
// at or below a source, so each hop consumes a segment of the remaining
// suffix and the chain ends.
export function resolveRedirect(
  redirects: readonly Redirect[],
  urlPath: string,
): string | undefined {
  const bySource = redirectIndex(redirects);
  let target: string | undefined;
  for (let current = urlPath; ;) {
    const match = deepestRedirect(bySource, current);
    if (!match) return target;
    current = target = match.to + current.slice(match.from.length);
  }
}

// The manifest is fetched once per request, so its page and metadata share
// one index of its redirects.
const indexes = new WeakMap<readonly Redirect[], Map<string, Redirect>>();

function redirectIndex(
  redirects: readonly Redirect[],
): ReadonlyMap<string, Redirect> {
  let index = indexes.get(redirects);
  if (!index) {
    index = new Map(redirects.map((r) => [r.from, r]));
    indexes.set(redirects, index);
  }
  return index;
}

function deepestRedirect(
  bySource: ReadonlyMap<string, Redirect>,
  urlPath: string,
): Redirect | undefined {
  for (const source of ancestorPaths(urlPath)) {
    const redirect = bySource.get(source);
    if (redirect && (redirect.prefix || source === urlPath)) return redirect;
  }
  return undefined;
}

// ancestorPaths yields urlPath and each ancestor above it, deepest first,
// excluding the root.
function* ancestorPaths(urlPath: string): Generator<string> {
  for (let at = urlPath; at.length > 1; at = at.slice(0, at.lastIndexOf("/"))) {
    yield at;
  }
}

// A base prefixes every site link: "" for the public site, or the preview
// articles route, whose root has no trailing slash.
export function previewBase(token: string): string {
  return `/preview/${token}/articles`;
}

export function siteHref(base: string, urlPath: string): string {
  if (urlPath === "/") return base || "/";
  return base + urlPath;
}

export function pageTitle(siteTitle: string, title: string): string {
  return title === siteTitle ? siteTitle : `${title} | ${siteTitle}`;
}

// themeStyle exposes a validated theme as CSS custom properties, so the
// stylesheet stays static and only these values change with configuration.
export function themeStyle(theme: Theme): CSSProperties {
  return {
    "--accent": theme.accent,
    "--background": theme.background,
    "--text": theme.text,
  } as CSSProperties;
}
