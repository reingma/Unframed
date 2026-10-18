# Content authoring schema

A content tree is a directory of Markdown articles. Its layout is the site's
navigation: directories become sections and articles become pages.
[backend/internal/content](../../backend/internal/content) implements this schema.
[ADR 0004](../decisions/0004-content-source-and-authoring-format.md) records why.

```text
content/
├── .configuration                 # required root configuration
├── Mathematics/
│   ├── .configuration             # optional directory configuration
│   ├── index.md                   # landing page of /mathematics
│   ├── On-Martingales.md          # /mathematics/on-martingales
│   └── On-Martingales.draft.md    # draft: indexed, never listed
└── Essays/
    └── Hello-World.md             # /essays/hello-world
```

## Names

Directory names and article names, without `.md` or `.draft.md`, are ASCII
letters and digits in segments joined by single hyphens, such as
`On-Martingales` or `ENG-34-Notes`. Spaces, underscores, dots and leading,
trailing or repeated hyphens are invalid.

- The display title replaces hyphens with spaces: `On Martingales`.
- The URL slug is the lowercased name: `on-martingales`. A node's URL is its
  ancestors' slugs followed by its own, such as `/mathematics/on-martingales`.

Dot-prefixed files and directories, such as `.git`, are ignored, except
`.configuration` files. Symbolic links and other non-regular files are
ignored. Files that are not `.md` are not indexed; asset handling is not
implemented yet.

## Configuration

Configuration is TOML. The root and any directory may contain a
`.configuration` file. An article may start with TOML frontmatter between two
lines that contain exactly `+++`:

```markdown
+++
title = "On Martingales and Stopping"
order = 2
+++
# Body
```

Without an opening `+++` line, the whole file is the article body and its
configuration is empty.

Three keys describe the node itself and are never inherited:

| Key     | Type                                  | Effect                                      |
| ------- | ------------------------------------- | ------------------------------------------- |
| `title` | non-empty string                      | Replaces the derived display title.         |
| `slug`  | lowercase slug, such as `my-section`  | Replaces the derived URL slug.              |
| `order` | integer                               | Sorts the node before unordered siblings.   |

The root may set `title` but not `slug` or `order`. A landing page may set
`title` but not `slug` or `order`.

Every other key is preserved and cascades from the root through each
directory to its articles. A child's value replaces an inherited value, except
that tables merge key by key at every depth. Arrays are replaced, not
concatenated. For example, with a root `[theme]` of `accent = "blue"` and
`font = "serif"` and an article `[theme]` of `font = "mono"`, the article's
theme is `accent = "blue"` and `font = "mono"`.

## Navigation and drafts

- `index.md` is its directory's landing page. It shares the directory's URL
  and title unless it sets `title`, and it is not listed as a child.
- `X.draft.md` is a draft of `X`. Drafts are in the article index with a draft
  flag, but never in the navigation tree.
- Directories without published articles or landing pages are omitted from
  the navigation tree.
- Siblings are listed by `order` ascending, then unordered siblings by
  case-insensitive title, then by slug.

## Validation

A scan reports each invalid item with its path and excludes it; valid items
are unaffected. These are per-item errors:

- An invalid directory or article name.
- Invalid TOML, a missing closing `+++`, a wrongly typed `title`, `slug` or
  `order`, or `slug`/`order` on a landing page.
- An unreadable file.
- Siblings that resolve to the same URL. Every sibling in the collision is
  excluded. A draft and a published article only collide with siblings of the
  same kind.

An invalid directory name or directory `.configuration` excludes the whole
directory. A missing, unreadable or invalid root `.configuration` is fatal:
no manifest is produced.
