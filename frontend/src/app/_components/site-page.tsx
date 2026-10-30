import type { Metadata } from "next";

import type { Envelope, Manifest, NavNode } from "../../lib/documents";
import { pageTitle } from "../../lib/site";

import { ArticleView } from "./article-view";
import { DirectoryListing } from "./directory-listing";
import { SiteShell } from "./site-shell";

// A SitePage is an article or landing envelope, or else the generated
// listing of a directory without index.md.
export type SitePage =
  | { kind: "envelope"; manifest: Manifest; envelope: Envelope }
  | { kind: "listing"; manifest: Manifest; node: NavNode };

export function sitePageMetadata(page: SitePage): Metadata {
  const { title, description } =
    page.kind === "envelope" ? page.envelope.metadata : page.node;
  return {
    title: pageTitle(page.manifest.site.title, title),
    description: description || page.manifest.site.description,
  };
}

export function SitePageView({
  page,
  base,
  preview,
}: {
  page: SitePage;
  base: string;
  preview?: boolean;
}) {
  const envelope = page.kind === "envelope" ? page.envelope : undefined;
  return (
    <SiteShell
      manifest={page.manifest}
      theme={envelope?.resolvedConfig.theme ?? page.manifest.site.theme}
      base={base}
      currentPath={
        envelope?.metadata.urlPath ??
        (page.kind === "listing" ? page.node.urlPath : "/")
      }
      preview={preview}
    >
      {page.kind === "envelope" ? (
        <ArticleView envelope={page.envelope} />
      ) : (
        <DirectoryListing node={page.node} base={base} />
      )}
    </SiteShell>
  );
}
