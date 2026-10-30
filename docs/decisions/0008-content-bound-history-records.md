# 0008 — Content-bound history records

Status: accepted in [ENG-47](https://linear.app/unframed-blog/issue/ENG-47)
planning (2026-10-30); implemented under ENG-47. Amends the record binding of
[ADR 0003](0003-performance-history-records-and-measurement.md).

## Context

Each `perf/history/` record named the commit it measured in `revision`, and
the merge gate required that commit to resolve and differ from the PR head only
by record appends. A squash or rebase merge drops the PR's commits from `main`,
so most committed records named commits that no longer exist there. A new PR
then failed the gate until it added its own record, and nothing on `main` led
from a record back to the code it measured. Merge commits were the only
working strategy.

## Decision

- Records bind to content, not to a commit. Schema version 3 adds
  `content_hash`: the git tree ID of the measured commit's tree with
  `perf/history/` removed. Appending records leaves it unchanged, and any
  other change alters it, so it keeps the previous drift rule while surviving
  squash, rebase and merge-commit merges. It is a genuine tree ID that git
  tools reproduce, and computing it writes no objects.
- The hash is captured when a run starts. `perf run` and `perf content-hash`
  refuse a worktree with uncommitted or untracked changes outside
  `perf/history/` and stamp the hash into every artifact. `compress` takes it
  from the artifacts, refuses a run whose artifacts disagree, and refuses a
  `revision` whose content differs. Compress-time hashing could not prove what
  the run measured.
- The gate compares the newest record's hash with head's and no longer
  resolves `revision`, which stays informational. Every added record must
  carry a hash. Version 1 and 2 records stay unchanged on disk.
- `perf locate` walks `main`'s first-parent history for the oldest commit
  whose content matches a record, and `compare`/`report` show that commit.
  The walk is on demand only, never in CI.
- Rejected: fetching `refs/pull/*/head` in CI, which is GitHub-only and leaves
  no lineage on `main` or in clones; and a post-merge fix-up commit, which
  breaks record immutability and needs an automatic push pipeline.

## Consequences

- Any push outside `perf/history/` after measuring still needs a fresh record,
  docs-only pushes included. Hashing only measurement inputs is a separate
  decision ([ENG-53](https://linear.app/unframed-blog/issue/ENG-53)).
- A PR squashed while `main` has moved on lands content that nobody measured,
  and `locate` finds no commit for its record. Requiring up-to-date branches
  would close that gap; it is a repository protection decision
  ([ENG-52](https://linear.app/unframed-blog/issue/ENG-52)).
- The gate proves a record names the merged content, not that the measurement
  ran on `workstation-1`.
- `locate` costs one tree computation per first-parent commit walked.
