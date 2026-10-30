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
// matches, keeping the path's suffix below it; the deepest match wins.
export function resolveRedirect(
  redirects: readonly Redirect[],
  urlPath: string,
): string | undefined {
  let match: Redirect | undefined;
  for (const redirect of redirects) {
    const matches =
      urlPath === redirect.from ||
      (redirect.prefix && urlPath.startsWith(redirect.from + "/"));
    if (matches && (!match || redirect.from.length > match.from.length)) {
      match = redirect;
    }
  }
  return match && match.to + urlPath.slice(match.from.length);
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
