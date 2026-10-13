# Architectural decisions

Record enduring choices and rationale here; keep implementation acceptance criteria
in the authoritative specification. Use numbered Markdown records with status,
context, decision, consequences, and links to the work that approved the choice.
Supersede a decision explicitly rather than silently rewriting its history.

- [0001: monorepo and rendering boundary](0001-monorepo-and-rendering-boundary.md)

Engineering resources in Linear also govern collaboration:

- [Operating Model — Unframed](https://linear.app/unframed-blog/document/operating-model-unframed-e5167f7747a9)
- [Engineering Principles — Unframed](https://linear.app/unframed-blog/document/engineering-principles-unframed-e94d1836af7e)
- [PR Naming Convention](https://linear.app/unframed-blog/document/pr-naming-convention-fd814085c80c):
  branch `eng-123-implement-markdown-renderer`, PR title
  `[ENG-123] Implement Markdown renderer`, and `Fixes`/`Refs` issue linkage.
- [Commit Naming Schema](https://linear.app/unframed-blog/document/architecture-decision-commit-naming-schema-39b96b2fcbdf):
  commit messages use `type(area): message`. The document labels this as branch
  naming, but the dedicated PR Naming Convention supplies Git-valid branch names.

The resources remain authoritative for their policies; this index is not a second
copy of those policies.
