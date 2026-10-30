# 0007 — Next.js page resolution and preview routes

Status: accepted in [ENG-37](https://linear.app/unframed-blog/issue/ENG-37)
implementation; implemented under ENG-37.

## Context

[ADR 0001](0001-monorepo-and-rendering-boundary.md) gives Next.js the
rendering and Go the content. Go serves a manifest, one envelope per article
and token-gated previews ([ADR 0006](0006-snapshot-serving-and-rebuilds.md)).
Next.js must turn a page URL into those requests without exposing Go, drafts
or the preview token.

## Decision

- Public pages resolve their path in the manifest before asking for an
  envelope. Unknown paths are 404s without an envelope request, and a
  directory without `index.md` gets a listing generated from its manifest
  children, because Go has no envelope for it.
- Preview pages mirror Go's routes: `/preview/<token>/articles/<slug…>` and
  `/preview/<token>/diagnostics`. Go is asked first, so the token gates every
  preview page, listings included. A miss shows Go's validation errors, so an
  author sees why an item is missing. The cost is that a top-level item with
  the slug `preview` is shadowed on the public site.
- Responses are validated before any rendering, and a bad response fails the
  whole page with HTTP 500 rather than rendering partial data. Pages await
  their data before rendering, so 404 and 500 statuses are real HTTP statuses,
  not streamed soft errors.
- Themes are applied as CSS custom properties on the page frame, so the
  stylesheet and client bundle do not change with configuration.

## Consequences

- Every public page fetches the manifest; landing and article pages also fetch
  their envelope. React `cache` shares both between metadata and page rendering.
- Go should reject a top-level `preview` slug in a later change.
- Preview pages render with HTTP 200 even for a miss; they are `noindex`.
