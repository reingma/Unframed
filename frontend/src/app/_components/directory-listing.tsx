import type { NavNode } from "../../lib/documents";
import { siteHref } from "../../lib/site";

// DirectoryListing is the generated page of a directory without index.md.
export function DirectoryListing({
  node,
  base,
}: {
  node: NavNode;
  base: string;
}) {
  return (
    <section className="listing">
      <h1>{node.title}</h1>
      {node.description && <p>{node.description}</p>}
      {node.children.length === 0 ? (
        <p>Nothing is published here yet.</p>
      ) : (
        <ul>
          {node.children.map((child) => (
            <li key={child.urlPath}>
              <a href={siteHref(base, child.urlPath)}>{child.title}</a>
              {child.description && <p>{child.description}</p>}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
