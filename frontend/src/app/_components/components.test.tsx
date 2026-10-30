// @vitest-environment jsdom

import "@testing-library/jest-dom/vitest";
import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import type { Manifest, NavNode } from "../../lib/documents";
import { DirectoryListing } from "./directory-listing";
import { NavTree } from "./nav-tree";
import { SiteShell } from "./site-shell";

afterEach(cleanup);

function article(urlPath: string, title: string, description = ""): NavNode {
  return {
    kind: "article",
    urlPath,
    title,
    description,
    hasIndex: false,
    children: [],
  };
}

const guides: NavNode = {
  kind: "directory",
  urlPath: "/guides",
  title: "Guides",
  description: "How-to guides",
  hasIndex: false,
  children: [
    article("/guides/zeta", "Zeta", "Ordered first"),
    article("/guides/alpha", "Alpha"),
  ],
};

const root: NavNode = {
  kind: "directory",
  urlPath: "/",
  title: "Site",
  description: "",
  hasIndex: true,
  children: [guides, article("/about", "About")],
};

const manifest: Manifest = {
  schemaVersion: 1,
  site: {
    title: "Site",
    description: "",
    theme: { accent: "#336699", background: "#ffffff", text: "#111111" },
  },
  nav: root,
};

const links = (container: HTMLElement) =>
  within(container)
    .getAllByRole("link")
    .map((a) => [a.textContent, a.getAttribute("href")]);

describe("NavTree", () => {
  it("lists every node below the root in manifest order, nested", () => {
    render(<NavTree root={root} base="" currentPath="/" />);
    const nav = screen.getByRole("navigation", { name: "Site" });
    expect(links(nav)).toEqual([
      ["Guides", "/guides"],
      ["Zeta", "/guides/zeta"],
      ["Alpha", "/guides/alpha"],
      ["About", "/about"],
    ]);
    const guidesItem = within(nav)
      .getByRole("link", { name: "Guides" })
      .closest("li")!;
    expect(within(guidesItem).getAllByRole("link")).toHaveLength(3);
  });

  it("marks only the current page", () => {
    render(<NavTree root={root} base="" currentPath="/guides/alpha" />);
    expect(screen.getByRole("link", { name: "Alpha" })).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(
      screen.getAllByRole("link").filter((a) => a.hasAttribute("aria-current")),
    ).toHaveLength(1);
  });

  it("keeps links under the base", () => {
    render(<NavTree root={root} base="/preview/t/articles" currentPath="/" />);
    for (const [, href] of links(screen.getByRole("navigation"))) {
      expect(href).toMatch(/^\/preview\/t\/articles\//);
    }
  });
});

describe("DirectoryListing", () => {
  it("lists the directory's children with descriptions, in order", () => {
    render(<DirectoryListing node={guides} base="" />);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(
      "Guides",
    );
    expect(screen.getByText("How-to guides")).toBeInTheDocument();
    const items = screen.getAllByRole("listitem");
    expect(items.map((li) => li.textContent)).toEqual([
      "ZetaOrdered first",
      "Alpha",
    ]);
    expect(links(document.body)).toEqual([
      ["Zeta", "/guides/zeta"],
      ["Alpha", "/guides/alpha"],
    ]);
  });

  it("says when nothing is published", () => {
    render(<DirectoryListing node={{ ...guides, children: [] }} base="" />);
    expect(
      screen.getByText("Nothing is published here yet."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("list")).not.toBeInTheDocument();
  });
});

describe("SiteShell", () => {
  it("applies the page theme as CSS variables on the site frame", () => {
    const theme = { accent: "#aa3300", background: "#fffdf8", text: "#1d1d1d" };
    const { container } = render(
      <SiteShell manifest={manifest} theme={theme} base="" currentPath="/">
        <p>Body</p>
      </SiteShell>,
    );
    const site = container.querySelector(".site") as HTMLElement;
    expect(site.style.getPropertyValue("--accent")).toBe("#aa3300");
    expect(site.style.getPropertyValue("--background")).toBe("#fffdf8");
    expect(site.style.getPropertyValue("--text")).toBe("#1d1d1d");
    expect(screen.getByRole("link", { name: "Site" })).toHaveAttribute(
      "href",
      "/",
    );
    expect(screen.queryByText("Preview")).not.toBeInTheDocument();
  });

  it("labels preview pages", () => {
    render(
      <SiteShell
        manifest={manifest}
        theme={manifest.site.theme}
        base="/preview/t/articles"
        currentPath="/"
        preview
      >
        <p>Body</p>
      </SiteShell>,
    );
    expect(screen.getByText("Preview")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Site" })).toHaveAttribute(
      "href",
      "/preview/t/articles",
    );
  });
});
