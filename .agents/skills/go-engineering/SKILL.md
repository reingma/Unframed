---
name: go-engineering
description: Implement explicitly authorized Go changes with boundary-parsed domain types, meaningful interfaces, invariant-focused tests, early performance evidence, and correlated request observability. Use for Go implementation issues, not planning or independent review.
---

# Go engineering implementation

## Authority and required context

Obtain explicit human implementation authorization and scope; stop if missing.
Skill invocation, issue status and approved planning are not permission.
Read the current issue/acceptance criteria, host instructions/architecture/decisions,
native tooling, relevant evidence and human feedback. Resolve consequential gaps.

Read [shared principles](references/principles.md) before design: the single
implementation/review method source. [Worked examples](references/examples.md)
are optional; consult only relevant sections for types/null parsing, replaceable
lookup contracts, measurements or context propagation. Read `assets/` only when
inspecting/adapting that exercise; native checks can run without reading its source.
Do not load examples, assets or token reports by default.

## Workflow

1. Establish inputs, invariants, failure/zero-value semantics and external-state
   checks; separate requirements from assumptions.
2. Apply shared principles to types/interfaces and credible alternatives; choose
   the smallest sufficient change and explain consequential costs.
3. Select meaningful performance evidence early; capture available baselines.
4. Implement using host patterns, preserving context and safe request correlation.
5. Verify contracts/failure paths with appropriate tests, static checks, builds
   and measurements; run commands and inspect outputs. Failed, unavailable or
   inconclusive evidence is not a pass. Contract changes require approval, not
   weaker tests.
6. Prepare independent review; passing tests alone does not establish correctness.

## Handoff

Report authorization/issue, revision/scope, intent, invariants, alternatives/costs,
actual commands/results, performance workload/environment/raw comparisons,
context/correlation and sensitive-data checks, limitations and approval/review gaps.
Separate measured facts from hypotheses; invent no coverage, budgets or deployment
facts. Durable feedback refinements require explicit agreement and reviewed changes.

## Host adapter and limits

Follow host rules and approved library preferences. In Unframed, use `AGENTS.md`
and `docs/workflow.md`; application/tests/examples prefer `encoding/json/v2` and
require `GOEXPERIMENT=jsonv2`. The exercise is not an application feature or
production performance evidence.

This skill authorizes no provisioning, new tracing backend, automatic fixes/review
pipeline, global installation, agent/profile configuration, security-gate bypass,
production load, merge or auto-merge.
