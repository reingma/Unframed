# 0006 — Snapshot serving and rebuilds

Status: accepted in [ENG-36](https://linear.app/unframed-blog/issue/ENG-36) planning;
implemented under ENG-36.

## Context

[ADR 0005](0005-document-envelope-rendering-and-fanout.md) defines the build
output. Next.js fetches it on every page request without caching, so Go must
serve fresh documents cheaply, never a half-built site, and keep serving when
an author saves a broken file.

## Decision

- Go watches the content tree instead of checking freshness per request.
  Requests read only the current snapshot.
- Each successful build becomes an immutable snapshot, swapped atomically.
  Handlers read it through a small server-owned interface, the seam that a
  separate builder with a shared store would replace when serving scales out.
- The manifest and envelopes are encoded once per build and served as bytes,
  with a content hash per document for later caching headers. Encoded
  documents are decoded back, so a document the decoders reject is never served.
- One builder goroutine runs full rebuilds. Triggers only signal it, so builds
  never overlap and work is bounded by the build time, not the event or reload
  volume. Incremental builds wait until content is more than Markdown.
- The watcher uses `github.com/fsnotify/fsnotify` v1.9.0 behind a
  consumer-owned notifier interface; S3 event triggers are the production
  equivalent. v1.9.0 was chosen over v1.10.x under the maturity policy: v1.10
  was published in May 2026 and nothing here needs it.
- Startup fails on an invalid root, so a misconfigured server never listens.
  Afterwards the last good snapshot keeps serving.
- Reload is a synchronous, token-gated `POST` instead of a signal, so its
  caller learns which build it got. Preview uses a separate, shareable token.

## Consequences

- Every server instance builds and watches its own tree and holds the whole
  rendered site in memory.
- Filesystems without change notifications (NFS, SMB, FUSE) need a manual
  reload.
