import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { DiagnosticList } from "../../../_components/diagnostic-list";
import { SiteShell } from "../../../_components/site-shell";
import { getDiagnostics, getManifest } from "../../../../lib/backend";
import { isPreviewToken, pageTitle, previewBase } from "../../../../lib/site";

type Props = { params: Promise<{ token: string }> };

export async function generateMetadata(): Promise<Metadata> {
  const manifest = await getManifest();
  return { title: pageTitle(manifest.site.title, "Diagnostics") };
}

export default async function DiagnosticsPage({ params }: Props) {
  const { token } = await params;
  if (!isPreviewToken(token)) notFound();
  const [diagnostics, manifest] = await Promise.all([
    getDiagnostics(token),
    getManifest(),
  ]);
  if (!diagnostics) notFound();
  const { serving, lastBuild } = diagnostics;
  return (
    <SiteShell
      manifest={manifest}
      theme={manifest.site.theme}
      base={previewBase(token)}
      currentPath=""
      preview
    >
      <section>
        <h1>Diagnostics</h1>
        <p>
          Serving build {serving.buildId}: {serving.envelopes} envelopes,{" "}
          {serving.validationErrors} validation errors.
        </p>
        <p>
          Last build ({lastBuild.trigger}) {lastBuild.outcome}
          {lastBuild.error && `: ${lastBuild.error}`}
        </p>
        <DiagnosticList diagnostics={diagnostics.diagnostics} />
      </section>
    </SiteShell>
  );
}
