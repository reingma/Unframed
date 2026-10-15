---
name: implementation-planning
description: Plan implementation interactively, one meaningful decision at a time, covering security, early performance evidence, telemetry, verification, libraries and meaningful interfaces. Use for planning tasks before design is settled; produce a traceable discussion record and bounded follow-up issue proposals, not implementation authorization.
---

# Implementation planning

## Activation, authority and inputs

Use for an implementation-planning task, including a loosely specified initial
issue. Planning is discussion, not permission to implement. Obtain human kickoff
separately before changing code/settings, installing tools, opening implementation
PRs or executing experiments. Issue status and planning approval do not supply it.
Never merge or bypass security gates; this skill grants no tools or permissions.

Read the originating issue and prior discussion, approved intent and constraints,
host instructions/architecture/decisions, dependency/toolchain information, known
workloads and relevant evidence. Identify unknown facts rather than inventing them.
Read [the planning method](references/method.md) before conducting the discussion.
It owns the detailed method; host rules own authorization and delivery policy.

## Interactive procedure

1. Establish intent, invariants, scope, dependencies, uncertainties and credible
   alternatives. Ask the next consequential question, not an entire questionnaire.
2. Discuss inputs/trust boundaries, deployment/exposure assumptions, sensitive
   data, abuse and mitigations. Unverified hosting/security facts stay open.
3. Discuss workload, expensive choices, an initial baseline and early benchmarks
   or experiments. Do not wait for a regression or invent numerical budgets.
4. Discuss request-path correlation and diagnosing failures/degradation with
   appropriate logs/spans/metrics, privacy and overhead, without mandating services.
5. Discuss contract/failure verification and significant claim evidence; evaluate
   selected libraries and meaningful replaceable boundaries, including their costs.
6. Check actual discussion coverage before calling design settled. Record concerns,
   decisions, open questions or reasoned contextual deferrals, not checklist-only
   completion. Keep related future features parked and implementation choices flexible.
7. Conclude with bounded follow-up proposals tied to the origin. Create/update
   issues only when requested; unresolved consequential choices need further
   planning, not silent resolution or an automatically executed implementation.

Wait for the human after each meaningful decision question. Reflect their answer
and trade-off before proceeding; keep recommendations distinct from approvals.

## Output and optional resources

Keep planned outcomes, discussion links and acceptance criteria in the host tracker
(Linear in Unframed), referring to existing architecture/ADRs without copying
per-issue specifications into Git. The record names the origin/prior discussion,
intent/invariants/scope, alternatives, topic coverage, decisions and unresolved
questions, evidence needed and traced follow-ups with explicit kickoff status.
Discussion coverage is not implementation or verification completion.

Load [issue formats](references/issue-formats.md) when drafting requested issues
or formalizing the discussion; load [worked examples](references/examples.md)
only for a concrete record or planning exercise. Do not load examples or
token-performance reports routinely. Never fabricate links, approval or results.

## Unframed adapter

Use root `AGENTS.md`, `docs/workflow.md` and the linked Engineering resources for
host context, issue/PR conventions and authority. Respect approved library/API
preferences, including Go JSON v2; do not copy the Go implementation skill's
shared principles into this method or treat planning as executing that skill.
