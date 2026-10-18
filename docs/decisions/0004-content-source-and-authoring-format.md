# 0004 — Content source and authoring format

Status: accepted in [ENG-33](https://linear.app/unframed-blog/issue/ENG-33) planning;
implemented under [ENG-34](https://linear.app/unframed-blog/issue/ENG-34).

## Context

Unframed is a self-hosted Markdown blog whose content directory is the source
of truth. Authors need to organize and configure content without a database
or admin interface, and the backend must later run against object storage as
well as a local directory. Rendering, versioned documents and event fanout
follow in later work.

## Decision

- Go serves from a validated manifest derived entirely from the content tree,
  not from a separately maintained index.
- Content is read through a narrow `content.Source` interface that lists and
  reads files. The filesystem implementation comes first and is confined to
  its root with `os.Root`; S3 can implement the same interface later.
- The tree is the navigation. Directory and article names are slug-form, so
  titles and URLs derive from them without per-file configuration.
- Configuration is TOML: `.configuration` files for the root and directories
  and `+++` frontmatter for articles. It has a fixed, typed schema validated
  per field with the file, key, position and rule of each error. Only the
  theme cascades root → directory → article; the root must define the title
  and a complete theme. TOML is parsed with `github.com/pelletier/go-toml/v2`,
  whose `unstable` parser locates keys for error positions.
- Drafts use an `X.draft.md` name: indexed for previews, never listed.
- Invalid items are excluded and reported without blocking valid content.
  Only an invalid root configuration is fatal, because every node inherits it.

## Consequences

- Moving or renaming a file changes its URL; there is no stable ID yet.
- Unknown configuration keys are ignored, so files may carry settings for
  later features, but a misspelled key is silently ignored rather than
  rejected. New settings need a schema change.
- The `unstable` go-toml API is outside that library's compatibility
  guarantee; it is pinned and only used to locate keys.
- Scanning reads every article. ENG-34's benchmark scans 1,000 articles from a
  local filesystem in about 20 ms; very large trees or remote sources may need
  concurrent reads or incremental scans.
- The [authoring schema](../specs/content-authoring.md) is the canonical
  description of the format.
