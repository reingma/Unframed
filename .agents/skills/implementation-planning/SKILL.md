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

## Discussion and handoff

Follow the method's discussion round, one meaningful decision question at a time;
wait for the human and reflect the answer/trade-off before proceeding. Check
actual topic coverage before settling design; consequential unknowns remain open.
Keep recommendations, planning approval and completed implementation distinct.

## Output and optional resources

Keep the discussion record and bounded follow-up proposals in the host tracker
(Linear in Unframed), linked to the origin/prior discussion and existing
architecture/ADRs, not copied into Git. Create/update issues only when requested;
unresolved choices need focused planning, not automatically executed implementation.

Load [issue formats](references/issue-formats.md) when drafting requested issues
or formalizing the discussion; load [worked examples](references/examples.md)
only for a concrete record or planning exercise. Do not load examples or
token-performance reports routinely. Never fabricate links, approval or results.

## Unframed adapter

Use root `AGENTS.md`, `docs/workflow.md` and the linked Engineering resources for
host context, issue/PR conventions and authority. Respect approved library/API
preferences, including Go JSON v2; do not copy the Go implementation skill's
shared principles into this method or treat planning as executing that skill.
