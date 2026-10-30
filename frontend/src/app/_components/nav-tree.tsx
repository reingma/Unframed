import type { NavNode } from "../../lib/documents";
import { siteHref } from "../../lib/site";

type NavTreeProps = {
  root: NavNode;
  base: string;
  currentPath: string;
};

// NavTree renders the manifest tree in the order Go sorted it.
export function NavTree({ root, base, currentPath }: NavTreeProps) {
  return (
    <nav aria-label="Site" className="site-nav">
      <NavList nodes={root.children} base={base} currentPath={currentPath} />
    </nav>
  );
}

function NavList({
  nodes,
  base,
  currentPath,
}: {
  nodes: readonly NavNode[];
  base: string;
  currentPath: string;
}) {
  if (nodes.length === 0) return null;
  return (
    <ul>
      {nodes.map((node) => (
        <li key={node.urlPath}>
          <a
            href={siteHref(base, node.urlPath)}
            aria-current={node.urlPath === currentPath ? "page" : undefined}
          >
            {node.title}
          </a>
          <NavList
            nodes={node.children}
            base={base}
            currentPath={currentPath}
          />
        </li>
      ))}
    </ul>
  );
}
