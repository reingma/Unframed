# 0005 — Document envelope, Markdown rendering and event fanout

Status: accepted in [ENG-33](https://linear.app/unframed-blog/issue/ENG-33) planning;
implemented under [ENG-35](https://linear.app/unframed-blog/issue/ENG-35).

## Context

[ADR 0004](0004-content-source-and-authoring-format.md) turns the content tree
into a validated manifest. The frontend needs a stable document per article
and a navigation document, and the body representation is expected to grow
from HTML to structured blocks. Builds also produce facts, such as published
articles and validation errors, that local tooling and later notification
services consume.

## Decision

- Every document is a JSON object with an integer `schemaVersion`. Decoders
  reject missing or unknown versions; within a version, members are additive.
- An article envelope separates non-inherited `metadata`, the cascaded
  `resolvedConfig` and a `body` tagged by `kind`. Version 1 has only
  `{kind: "html", content}`; structured blocks become another kind.
- Markdown is rendered by `github.com/yuin/goldmark` as CommonMark with GFM
  extensions. It is mature, dependency-free and extensible for later heading
  IDs and math. Raw HTML is not passed through, so rendered output stays safe
  if content ever comes from a less trusted source.
- The scan keeps each article's Markdown body from its single read, so
  frontmatter and body always come from the same file contents.
- A build publishes one batch of events through an `event.Publisher`:
  `article.published` and `validation.failed`, all carrying the build ID that
  also tags its log records. Local publishers log events and write report
  files; `event.Fanout` combines publishers. Reports must be outside the
  content root, so they can never become content.
- A publishing failure does not discard a build: the caller receives the
  result and an error wrapping `build.ErrPublish`, and decides the policy.

## Consequences

- Readers of the documents must check `schemaVersion` and `body.kind`.
- Raw HTML in Markdown is dropped; allowing it later is a renderer option and
  a security decision.
- Events say what a build contains, not what changed since the previous
  build. Notifications that must fire once per new article need state.
- Building 1,000 small articles from memory takes about 20 ms on the ENG-35
  development VM, against about 3 ms to scan them; rendering dominates.
  Rendering runs sequentially and could be parallelized for large trees.
- The [document contract](../specs/document-contract.md) is the canonical
  description of the documents.
