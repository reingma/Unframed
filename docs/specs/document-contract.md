# Document contract

Go turns the content tree into JSON documents for the frontend: one envelope
per article and one public manifest. A build also emits [events](#build-events)
for subscribers. This page is their canonical description;
[document.go](../../backend/internal/document/document.go) implements it and
[ADR 0005](../decisions/0005-document-envelope-rendering-and-fanout.md) records
the rationale. The [authoring schema](content-authoring.md) describes the input.
Go serves them over the [JSON API](architecture.md#http-and-rendering-contract),
and the frontend validates them with [matching Zod schemas](../../frontend/src/lib/documents.ts).

## Versioning

Every document has an integer `schemaVersion`, currently `1`. A decoder
rejects a document whose version is missing or unknown instead of guessing.
Within a version, members may be added and decoders ignore members they do not
know. Removing, renaming or changing the meaning of a member requires a new
version. Member names are case-sensitive and may not repeat.

## Article envelope

```json
{
  "schemaVersion": 1,
  "metadata": {
    "urlPath": "/mathematics/on-martingales",
    "title": "On Martingales",
    "description": "Fair games",
    "draft": false,
    "landing": false
  },
  "resolvedConfig": {
    "theme": { "accent": "#336699", "background": "#ffffff", "text": "#111111" }
  },
  "body": { "kind": "html", "content": "<h1>On Martingales</h1>\n" }
}
```

- `metadata` identifies and describes the article and is never inherited.
  `urlPath` starts with `/`; `title` is never empty. A landing page has its
  directory's `urlPath`. A draft has the `urlPath` of the article it previews,
  so a published article and its draft are told apart by `draft`.
- `resolvedConfig` holds the configuration after the root → directory →
  article cascade. Theme colors are lowercase `#rrggbb`.
- `body.kind` names the body representation. Version 1 defines only `html`:
  `content` is an HTML fragment rendered from the Markdown body as CommonMark
  with the GitHub Flavored Markdown tables, strikethrough, autolinks and task
  lists. Raw HTML in the Markdown is replaced with an HTML comment, and links
  with unsafe schemes such as `javascript:` are emptied. Headings have no IDs.

Source paths, ordering and validation errors are internal and never appear in
an envelope.

## Public manifest

```json
{
  "schemaVersion": 1,
  "site": {
    "title": "Unframed",
    "description": "A blog",
    "theme": { "accent": "#336699", "background": "#ffffff", "text": "#111111" }
  },
  "nav": {
    "kind": "directory",
    "urlPath": "/",
    "title": "Unframed",
    "description": "A blog",
    "hasIndex": false,
    "children": [
      {
        "kind": "article",
        "urlPath": "/on-martingales",
        "title": "On Martingales",
        "description": "Fair games",
        "hasIndex": false,
        "children": []
      }
    ]
  },
  "redirects": [
    { "from": "/martingales", "to": "/on-martingales", "prefix": false }
  ]
}
```

- `site` is the root configuration.
- `nav` is the published navigation tree, rooted at the `/` directory. A node's
  `kind` is `directory` or `article`. A directory with a landing page has
  `hasIndex: true` and that page's envelope at its `urlPath`. Only directories
  have `children`, listed in navigation order; `children` is always an array.
- `redirects` lists the valid aliases of published nodes, ordered by `from`,
  with chains resolved so `to` is always published. `from` is an internal path
  of lowercase slugs, never published and never repeated. A `prefix` redirect,
  from a directory alias, also sends every path below `from` to the same suffix
  below `to`; the deepest matching `from` wins. `redirects` was added within
  version 1, so a manifest without it has no redirects.
- The manifest never contains drafts, invalid items or validation errors.

## Build events

A build emits one event per published article and one per validation error;
[event.go](../../backend/internal/event/event.go) implements them. Events
follow the [versioning](#versioning) rules with their own `schemaVersion`,
currently `1`. Each event is self-contained, since a bus delivers events one
at a time.

```json
{
  "schemaVersion": 1,
  "id": "Q2X7…-1",
  "buildId": "Q2X7…",
  "time": "2026-10-18T03:00:00Z",
  "kind": "article.published",
  "path": "Mathematics/On-Martingales.md",
  "urlPath": "/mathematics/on-martingales"
}
```

- `kind` is `article.published` or `validation.failed`.
- `id` is unique per event; a consumer may receive an event more than once and
  can use `id` to drop duplicates.
- `buildId` and `time` are shared by every event of a build.
- `path` is the item's slash-separated path in the content source. `urlPath`
  is present for `article.published`, `message` (the validation error) for
  `validation.failed`.
- Events report what a build contains, not what changed since the last build.
