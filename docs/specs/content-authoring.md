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
ignored. Other files that are not `.md` are assets. The manifest lists them
separately from the navigation tree, at their tree path. Assets inside an
excluded directory are excluded with it.

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
configuration is empty. The [document contract](document-contract.md) describes
how the body is rendered.

These keys are recognized; every other key is ignored:

| Key                | Type                                 | Root     | Directory, article | Landing page |
| ------------------ | ------------------------------------ | -------- | ------------------ | ------------ |
| `title`            | text of at most 120 characters       | required | optional           | optional     |
| `description`      | text of at most 300 characters       | optional | optional           | optional     |
| `slug`             | lowercase slug, such as `my-section` | rejected | optional           | rejected     |
| `order`            | integer from 0 to 9999               | rejected | optional           | rejected     |
| `aliases`          | array of at most 32 distinct paths   | rejected | optional           | rejected     |
| `theme.accent`     | `#RRGGBB` color                      | required | optional           | optional     |
| `theme.background` | `#RRGGBB` color                      | required | optional           | optional     |
| `theme.text`       | `#RRGGBB` color                      | required | optional           | optional     |

Text is trimmed of surrounding whitespace and must be a non-empty single line
without control characters. Colors accept hexadecimal digits in either case.

`title` replaces the derived display title, `slug` the derived URL slug, and
`order` sorts the node before unordered siblings. `title`, `description`,
`slug`, `order` and `aliases` describe their node only and are never inherited.

`aliases` lists the node's former URL paths, so links survive a rename or a
move. Each alias is `/` followed by lowercase slugs joined by `/`, such as
`/essays/old-title`; URLs, schemes, relative paths and trailing slashes are
rejected. A published article's aliases redirect permanently to it. A
published directory's aliases also redirect every path below them to the
same path below the directory, so `/old-section/post` reaches
`/new-section/post`. Chains of aliases resolve to their final target. A
draft's aliases are checked but never published.

The `[theme]` colors cascade from the root through each directory to its
articles: each color a node sets replaces the inherited one. Because the root
sets every color, every node has a complete theme. For example:

```toml
# .configuration
title = "Unframed"

[theme]
accent = "#336699"
background = "#FFFFFF"
text = "#111111"
```

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
- Invalid TOML, a missing closing `+++`, or a recognized key that breaks the
  rules above. Each error names the file, the key path such as
  `theme.accent`, its line and column, the broken rule and the value found.
  Every broken rule in a file is reported at once.
- An unreadable file.
- Siblings that resolve to the same URL. Every sibling in the collision is
  excluded. A draft and a published article only collide with siblings of the
  same kind.
- An alias that equals its own node's URL, is claimed by more than one
  published node, equals a published URL, or redirects back into itself,
  such as two articles aliasing each other's paths or a directory alias below
  the directory. Only that alias is dropped; its node stays published. Remove
  stale aliases that a published path has taken over.

An invalid directory name or directory `.configuration` excludes the whole
directory. A root `.configuration` that is missing, unreadable or breaks the
rules, for example by lacking a required key, is fatal: no manifest is
produced.
