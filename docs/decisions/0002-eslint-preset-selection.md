# 0002 — ESLint preset selection

Status: records the existing ENG-14 implementation choice; no new tooling decision.

## Context

The initial toolchain was reviewed on 2026-10-13 using official releases,
engine and peer constraints, dependency audits and local compatibility checks.
Exact pins and dependency resolution remain in the package manifests and lockfiles.

At that review, the optional Next.js ESLint preset's React, import and accessibility
plugins excluded the selected ESLint major through peer constraints. The older
linter major was end-of-life, and the preset's fast-glob dependency pulled an
unpatched braces vulnerability:
[GHSA-vfj7-8cjw-p6xm](https://github.com/advisories/GHSA-vfj7-8cjw-p6xm).

## Decision

Use the core recommended, TypeScript recommended and React Hooks recommended
rules instead of the optional Next.js preset. Use compatible, supported plugins
without peer-dependency bypasses, an end-of-life linter or audit suppression.

## Consequences and revisit conditions

The preset's additional framework rules are not enabled. This initial tooling
choice does not change repository merge gates. Re-evaluate the optional preset
when its ecosystem supports the chosen linter and has patched dependencies.

## References

- [ENG-14](https://linear.app/unframed-blog/issue/ENG-14/implement-initial-go-and-nextjs-monorepo-skeleton)
- [ESLint configuration](../../frontend/eslint.config.mjs)
- [Package pins](../../frontend/package.json) and [resolved dependencies](../../frontend/package-lock.json)
- [Current architecture](../specs/architecture.md#tooling-sources)

This record relocates the rationale from the architecture specification during
[ENG-27](https://linear.app/unframed-blog/issue/ENG-27/consolidate-repository-documentation-remove-duplication-restore).
