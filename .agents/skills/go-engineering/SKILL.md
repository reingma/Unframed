---
name: go-engineering
description: Implement authorized Go changes with parse-at-boundary domain types, meaningful interfaces, flexible invariant-focused tests, early performance measurements, and correlated request observability. Use when executing an implementation issue involving Go code; not as authorization to implement a plan or as a substitute for independent review.
---

# Go engineering implementation

## Activation and authority

Use during an explicitly authorized Go implementation issue. First establish the
human authorization, approved scope, and repository rules. Missing authorization
means stop before changing code; a skill invocation, issue status, or approved
plan does not supply it. This skill is guidance, not a permission mechanism.

## Inputs

- Current issue and acceptance criteria, authorization, and unresolved decisions.
- Repository instructions, architecture/decisions, toolchain and native commands.
- Changed boundaries, input sources, core invariants and credible alternatives.
- Relevant tests, workloads/baselines, telemetry conventions, and human feedback.

Read [the shared principles](references/principles.md) before designing the change.
That file is the common implementation/review source, not a separate review
checklist. Consult [worked examples](references/examples.md) for concrete choices,
commands, and an isolated executable exercise.

## Procedure

1. **Establish the contract.** Identify boundary inputs, promised behavior,
   failure semantics, zero-value behavior, and facts depending on authorization
   or mutable external state. Separate requirements from assumptions; ask about
   consequential gaps rather than silently choosing a new contract.
2. **Choose representations and boundaries.** Apply the shared principles to
   parsing/domain types and credible implementation alternatives. Explain the
   protected invariant, useful replaceability, and abstraction cost. Prefer the
   smallest change satisfying the issue, not wrappers/interfaces everywhere.
3. **Choose performance evidence early.** Identify a meaningful workload before
   implementing an expensive path. Capture a baseline when available, including
   revision, environment, input size/distribution, command and raw output. Avoid
   obviously poor complexity; compare credible choices without imposing an
   unapproved performance budget or mandatory library substitution.
4. **Implement and instrument.** Follow existing patterns and native tooling.
   Preserve context across boundaries; correlate structured logs/spans with the
   request path, without payload/secret leakage or speculative telemetry services.
5. **Verify behavior.** Write tests for contracts and failure paths. Choose unit,
   integration, property/fuzz and performance checks by the claim being tested.
   Ordinary substitutions should preserve contract tests. A deliberate contract
   change requires design reconsideration and human approval, not weaker tests.
   Run formatting/static checks, relevant tests/builds, and actual measurements;
   inspect outputs. Failed, unavailable or inconclusive evidence is not a pass.
6. **Prepare independent review.** Report the evidence and trade-offs below, so
   a reviewer can evaluate the same principles without adopting the author's
   assumptions. Tests and review complement one another. Do not declare the
   change verified solely because the test suite passes.

## Expected handoff evidence

- Issue/authorization, reviewed revision or working-tree scope, and change intent.
- Protected invariants, boundary/type/interface choices, alternatives and costs.
- Actual formatting/static/test/build commands and outcomes; failure-path and
  property/fuzz coverage where meaningful, plus what remains untested.
- Performance baseline/current workload, environment, raw results and limitations;
  distinguish observations from attribution hypotheses and accepted feature cost.
- Request-context propagation/correlation evidence, sensitive-data exclusions,
  and material logging/tracing overhead or gaps.
- Scope deviations, consequential decisions needing approval, and review gaps.

Do not invent performance thresholds, deployment facts, successful checks, or
coverage. Apply relevant human feedback to this issue; propose lasting changes
to the shared guidance separately for explicit agreement and reviewed changes.

## Repository adapter and limits

Use the host repository's authoritative instructions, architecture, native
commands, verification and delivery conventions rather than copying them here.
In Unframed, start with root `AGENTS.md` and `docs/workflow.md`; its application
Go commands require `GOEXPERIMENT=jsonv2`. The bundled exercise is not an
application feature, API design, performance budget, or production measurement.

No cloud provisioning, new tracing backend, automatic fixes/review pipeline,
global installation, agent profile/configuration change, security-gate bypass,
production load generation, merge, or auto-merge is authorized by this skill.
