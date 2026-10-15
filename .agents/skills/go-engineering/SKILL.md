---
name: go-engineering
description: Implement explicitly authorized Go changes with boundary-parsed domain types, meaningful interfaces, invariant-focused tests, early performance evidence, and correlated request observability. Use for Go implementation issues, not planning or independent review.
---

# Go engineering implementation

## Required context and loading

Follow the host's authority and delivery rules; skill activation grants no permission.
Stop if explicit implementation authorization or scope is missing.
Read the current issue and acceptance criteria, host instructions, architecture,
decisions, native tooling, relevant evidence and human feedback.

Read [shared principles](references/principles.md) before design. This is the
single implementation/review method. Load relevant sections of
[worked examples](references/examples.md) only when exploring types, null parsing,
lookup contracts, measurements or context propagation. Read `assets/` only to
inspect or adapt that exercise. Native checks do not require reading its source.
Do not routinely load examples, assets or token reports.

## Procedure

1. Establish inputs, invariants, failure and zero-value behavior, and checks that
   depend on external state. Distinguish requirements from assumptions.
2. Use the shared method to assess representations, interfaces and credible
   alternatives. Choose the smallest sufficient change and explain consequential costs.
3. Select meaningful performance evidence early and capture available baselines.
4. Implement with host patterns, preserving context and safe request correlation.
5. Run the appropriate checks and inspect their outputs, following the shared
   verification and measurement methods.
6. Prepare independent review. Passing tests alone does not establish correctness.

## Review handoff

Identify the authorized issue, revision, scope, intent and invariants. Explain
alternatives, costs and remaining uncertainty. Report actual commands and results;
for performance claims, include workload, environment, raw results and comparisons.
Report context and correlation checks, sensitive-data checks, limitations, and
approval or review gaps. Separate observations from hypotheses.

## Unframed adapter

Follow [AGENTS.md](../../../AGENTS.md) for host policy, including JSON API and
build requirements. Follow [workflow guidance](../../../docs/workflow.md) for
discovery and loading conventions. The synthetic exercise is not an application
feature or production performance evidence.
