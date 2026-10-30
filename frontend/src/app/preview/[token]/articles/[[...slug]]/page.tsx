import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { DiagnosticList } from "../../../../_components/diagnostic-list";
import { SiteShell } from "../../../../_components/site-shell";
import {
  SitePageView,
  sitePageMetadata,
  type SitePage,
} from "../../../../_components/site-page";
import { getManifest, getPreviewArticle } from "../../../../../lib/backend";
import type { Manifest, PreviewMiss } from "../../../../../lib/documents";
import {
  findNode,
  isPreviewToken,
  pageTitle,
  previewBase,
  urlPathFromSlug,
} from "../../../../../lib/site";

type Props = { params: Promise<{ token: string; slug?: string[] }> };

type PreviewPage =
  SitePage | { kind: "miss"; manifest: Manifest; miss: PreviewMiss };

// resolve asks Go first, so the token gates every preview page. A miss is a
// directory listing when the path is one, and otherwise shows Go's
// diagnostics for the path.
async function resolve({
  params,
}: Props): Promise<{ token: string; page: PreviewPage }> {
  const { token, slug } = await params;
  const urlPath = urlPathFromSlug(slug);
  if (!isPreviewToken(token) || !urlPath) notFound();
  const [result, manifest] = await Promise.all([
    getPreviewArticle(token, urlPath),
    getManifest(),
  ]);
  if (result.kind === "denied") notFound();
  if (result.kind === "article") {
    return {
      token,
      page: { kind: "envelope", manifest, envelope: result.envelope },
    };
  }
  const node = findNode(manifest.nav, urlPath);
  if (node?.kind === "directory" && !node.hasIndex) {
    return { token, page: { kind: "listing", manifest, node } };
  }
  return { token, page: { kind: "miss", manifest, miss: result.miss } };
}

export async function generateMetadata(props: Props): Promise<Metadata> {
  const { page } = await resolve(props);
  if (page.kind !== "miss") return sitePageMetadata(page);
  return { title: pageTitle(page.manifest.site.title, "Not found") };
}

export default async function PreviewArticlePage(props: Props) {
  const { token, page } = await resolve(props);
  const base = previewBase(token);
  if (page.kind !== "miss")
    return <SitePageView page={page} base={base} preview />;
  return (
    <SiteShell
      manifest={page.manifest}
      theme={page.manifest.site.theme}
      base={base}
      currentPath={page.miss.urlPath}
      preview
    >
      <section>
        <h1>Nothing to preview at {page.miss.urlPath}</h1>
        <p>Build {page.miss.buildId} has no draft or published article here.</p>
        <DiagnosticList diagnostics={page.miss.diagnostics} />
      </section>
    </SiteShell>
  );
}
