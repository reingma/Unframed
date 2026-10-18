# Document contract

Go turns the content tree into JSON documents for the frontend: one envelope
per article and one public manifest. This page is their canonical description;
[document.go](../../backend/internal/document/document.go) implements it and
[ADR 0005](../decisions/0005-document-envelope-rendering-and-fanout.md) records
the rationale. The [authoring schema](content-authoring.md) describes the input.
These documents are not served over HTTP yet.

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
    "kind": "directory", "urlPath": "/", "title": "Unframed", "description": "A blog",
    "hasIndex": false,
    "children": [
      { "kind": "article", "urlPath": "/on-martingales", "title": "On Martingales",
        "description": "Fair games", "hasIndex": false, "children": [] }
    ]
  }
}
```

- `site` is the root configuration.
- `nav` is the published navigation tree, rooted at the `/` directory. A node's
  `kind` is `directory` or `article`. A directory with a landing page has
  `hasIndex: true` and that page's envelope at its `urlPath`. Only directories
  have `children`, listed in navigation order; `children` is always an array.
- The manifest never contains drafts, invalid items or validation errors.
