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
- A build emits `article.published` and `validation.failed` events. Each
  event is self-contained and versioned (`schemaVersion`, a unique `id`, the
  `buildId` that also tags the build's log records, and a time), because the
  production path is S3 event → build → SNS or EventBridge → subscribers such
  as email Lambdas, and a bus delivers events one at a time, possibly more
  than once.
- The event contract (`event`) is separate from delivery (`publish`). A build
  hands its events to one `publish.Publisher` in a single call, which maps to
  SNS `PublishBatch` or EventBridge `PutEvents`. Reaching several consumers is
  the bus's job, not the build's: locally `publish.Local` delivers to
  in-process subscribers (logs and report files); in production a bus
  publisher replaces it and consumers decode the event contract in their own
  process. Reports must be outside the content root, so they can never become
  content.
- A publishing failure does not discard a build: the caller receives the
  result and an error wrapping `build.ErrPublish`, and decides the policy.

## Consequences

- Readers of the documents must check `schemaVersion` and `body.kind`.
- Raw HTML in Markdown is dropped; allowing it later is a renderer option and
  a security decision.
- Consumers must check the event `schemaVersion` and may use `id` to drop
  redelivered events.
- Events say what a build contains, not what changed since the previous
  build. Notifications that must fire once per new article need state.
- Rendering runs sequentially and is expected to dominate build cost.
  `BenchmarkBuild` measures it alongside `BenchmarkScan`. Optimizing it waits
  for the full rendering feature, which will change its cost.
- The [document contract](../specs/document-contract.md) is the canonical
  description of the documents.
