import type { Metadata } from "next";
import { notFound, permanentRedirect } from "next/navigation";

import {
  SitePageView,
  sitePageMetadata,
  type SitePage,
} from "../_components/site-page";
import { getArticle, getManifest, logRedirect } from "../../lib/backend";
import { findNode, resolveRedirect, urlPathFromSlug } from "../../lib/site";

type Props = { params: Promise<{ slug?: string[] }> };

// resolve answers a public path from the manifest first, so paths Go does
// not publish are 404s without an envelope request, unless they are aliases
// of published paths, which redirect permanently. Metadata and the page both
// resolve a request, so only the page logs the redirect.
async function resolve({ params }: Props, logHit: boolean): Promise<SitePage> {
  const urlPath = urlPathFromSlug((await params).slug);
  if (!urlPath) notFound();
  const manifest = await getManifest();
  const node = findNode(manifest.nav, urlPath);
  if (!node) {
    const target = resolveRedirect(manifest.redirects, urlPath);
    if (!target) notFound();
    if (logHit) logRedirect(urlPath, target);
    permanentRedirect(target);
  }
  if (node.kind === "directory" && !node.hasIndex) {
    return { kind: "listing", manifest, node };
  }
  const envelope = await getArticle(urlPath);
  if (!envelope) notFound();
  return { kind: "envelope", manifest, envelope };
}

export async function generateMetadata(props: Props): Promise<Metadata> {
  return sitePageMetadata(await resolve(props, false));
}

export default async function PublicPage(props: Props) {
  return <SitePageView page={await resolve(props, true)} base="" />;
}
