import type { ReactNode } from "react";

import type { Manifest, Theme } from "../../lib/documents";
import { siteHref, themeStyle } from "../../lib/site";

import { NavTree } from "./nav-tree";

type SiteShellProps = {
  manifest: Manifest;
  theme: Theme;
  base: string;
  currentPath: string;
  preview?: boolean;
  children: ReactNode;
};

// SiteShell is the page frame: site header, navigation and the page theme.
export function SiteShell({
  manifest,
  theme,
  base,
  currentPath,
  preview = false,
  children,
}: SiteShellProps) {
  return (
    <div className="site" style={themeStyle(theme)}>
      <header className="site-header">
        <a href={siteHref(base, "/")} className="site-title">
          {manifest.site.title}
        </a>
        {preview && <span className="badge">Preview</span>}
      </header>
      <NavTree root={manifest.nav} base={base} currentPath={currentPath} />
      <main>{children}</main>
    </div>
  );
}
