import type { Metadata } from "next";
import { notFound } from "next/navigation";

import {
  SitePageView,
  sitePageMetadata,
  type SitePage,
} from "../_components/site-page";
import { getArticle, getManifest } from "../../lib/backend";
import { findNode, urlPathFromSlug } from "../../lib/site";

type Props = { params: Promise<{ slug?: string[] }> };

// resolve answers a public path from the manifest first, so paths Go does
// not publish are 404s without an envelope request.
async function resolve({ params }: Props): Promise<SitePage> {
  const urlPath = urlPathFromSlug((await params).slug);
  if (!urlPath) notFound();
  const manifest = await getManifest();
  const node = findNode(manifest.nav, urlPath);
  if (!node) notFound();
  if (node.kind === "directory" && !node.hasIndex) {
    return { kind: "listing", manifest, node };
  }
  const envelope = await getArticle(urlPath);
  if (!envelope) notFound();
  return { kind: "envelope", manifest, envelope };
}

export async function generateMetadata(props: Props): Promise<Metadata> {
  return sitePageMetadata(await resolve(props));
}

export default async function PublicPage(props: Props) {
  return <SitePageView page={await resolve(props)} base="" />;
}
