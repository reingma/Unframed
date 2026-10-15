# Security practices for review

The practices a security review checks changed code against. Apply those relevant
to the change. This is a lens for reading code, not a complete vulnerability
checklist; a review that finds nothing here has not shown the code is secure.

## Untrusted input at boundaries

Parse request data, environment, files, external responses and CI event fields
once at the boundary into types that encode what is allowed, following the shared
[boundary-parsing principle](../../go-engineering/references/principles.md#boundary-parsing-and-meaningful-types).
Suspect any unparsed value that reaches a decision, a path, a query or output.
Remember decoding: a router may unescape `%2F` into a value after path cleaning.

## Injection and path access

Values must not change the structure of SQL, shell commands, templates, HTML,
headers, URLs or file paths. Prefer parameterized APIs and allow-lists. A joined
path must stay under its intended root; `filepath.Join` does not guarantee that.
Rendering user content as HTML needs escaping or an allow-list sanitizer.

## Authentication and authorization

Every privileged action checks the caller on the server for that resource, not
only in the UI or by an unguessable identifier. Check new routes and methods,
including those added beside protected ones.

## Secret and data disclosure

Secrets and sensitive data stay out of logs, error responses, client bundles,
URLs and artifacts. In Next.js, a `NEXT_PUBLIC_` variable or a value returned to a
client component ships to the browser. Errors returned to callers should not
reveal internal paths, origins or stack traces.

## Outbound requests

A request whose host, scheme or path comes from input can reach internal services
(SSRF). Fix the origin in server-only configuration and validate responses before use.

## Resource exhaustion

Bound request bodies, uploads, decoded sizes, recursion, loops over input and
fan-out, and set timeouts on outbound calls and servers. Cost should not grow
without limit with attacker-controlled input.

## Dependencies and supply chain

New or upgraded dependencies need exact pins, a lockfile change that matches,
a maintained source and a reason. Check install scripts and unexpected transitive
additions. In CI, pin third-party actions to commit SHAs, keep workflow
`permissions` minimal, and never run untrusted pull-request code with secrets or a
write token, as `pull_request_target` or `workflow_run` with a head checkout can.
