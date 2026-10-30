import type { Diagnostic } from "../../lib/documents";

export function DiagnosticList({
  diagnostics,
}: {
  diagnostics: readonly Diagnostic[];
}) {
  if (diagnostics.length === 0) return <p>No validation errors.</p>;
  return (
    <ul className="diagnostics">
      {diagnostics.map((d) => (
        <li key={`${d.path}\n${d.message}`}>
          <code>{d.path}</code> ({d.urlPath}): {d.message}
        </li>
      ))}
    </ul>
  );
}
