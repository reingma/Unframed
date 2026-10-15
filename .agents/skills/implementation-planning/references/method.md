# Interactive planning method

## Establish and pace the discussion

Start from the originating issue, verified prior discussion and approved intent.
Separate requirements, observations, assumptions, recommendations and approvals.
State core invariants, scope/exclusions, dependencies and uncertainty. Briefly
frame the next meaningful trade-off, recommend an option with reasons, ask one
decision question and wait. Reflect the answer before the next decision; tightly
coupled choices may be discussed together, not an immediate complete plan.
Revisit earlier choices when new evidence matters. Scale detail to risk/scope and
park unrelated future features. Do not freeze implementation internals unnecessarily.

## Discussion round before design is settled

Touch every topic below in actual discussion while it can change the design.
For each, retain a concern, decision, open question or explicit contextual
deferral with rationale and a revisit condition. A heading or unchecked checklist
alone is not discussion evidence; a deferral is not a successful test.

### Security

Identify input sources and trust boundaries, deployment/exposure assumptions,
sensitive data, likely abuse and mitigations. Distinguish observed facts from
unconfirmed hosting/access controls. Consider invalid inputs, path/identifier
trust, resource exhaustion, access checks and safe diagnostics as relevant.
Explain how mitigations will be verified; never infer safety from "internal" or
"local" alone. Significant unknowns remain open rather than becoming requirements.

### Performance and early evidence

Discuss representative workload size/distribution/concurrency and expensive
choices. Plan an initial baseline with revision, environment, inputs and commands
alongside the first executable workload, not only after a suspected regression.
Choose operation benchmarks or application-path measurements according to the
claim; preserve raw results and compare like with like. A missing workload may
justify a bounded measurement/planning follow-up, not made-up numbers or a new
mandatory benchmark suite. Agree budgets explicitly; fixture speed is not
production evidence. Running experiments requires separate authorization.

### Telemetry

Discuss how to follow a request path and diagnose failure/degradation using
existing correlation/log/span/metric conventions as appropriate. Cover propagation,
completion/failure evidence, sensitive-data exclusion, cardinality, volume and
runtime cost. Distributed spans or a new backend can be explicitly deferred when
the path is process-local; do not introduce infrastructure just to cover a topic.

### Verification

Name observable contracts/invariants and failure paths. Choose unit and integration
checks; use property/fuzz tests for uncertain domain boundaries with independent,
meaningful properties, not every trivial helper. Specify evidence for material
security/performance claims, including what cannot yet be checked. Preserve tests
across implementation substitutions; intentional contract changes require design
reconsideration and approval, not weaker assertions. Planned checks are not passes.

### Libraries and interface boundaries

Analyze libraries involved in the proposed path: host preferences, compatibility,
correctness/security, maturity/maintenance, workload performance and integration
cost. Familiarity or existing use alone does not justify a legacy API when a
better suitable option exists; advertised speed alone is insufficient too.
Consult authoritative docs and applicable evidence before consequential selection.
Record alternatives and reasons, or an evidence-gathering follow-up. In Unframed,
the host-approved Go JSON preference is in `AGENTS.md`, not a universal mandate.

When selecting a concrete provider/library/storage/algorithm, discuss the needed
contract, credible alternatives, caller insulation and abstraction/runtime costs.
Define error, cancellation and ownership semantics where relevant. Prefer a small
consumer-owned interface at a real replaceable boundary; pure helpers with no
meaningful substitutes may stay concrete. Neither interfaces everywhere nor
"no abstraction" everywhere is a rule.

## Coverage record and follow-ups

Before settling design, check that each topic has a discussion reference and
substantive outcome. Reopen missing topics; record unresolved decisions visibly.
The concise living tracker record includes origin/prior discussion links, approved
intent, invariants, scope/dependencies, alternatives/costs, decisions versus open
questions/deferrals, planned evidence and the next bounded work items.
Record whose explicit approval exists and what it covers; otherwise say pending.

End with traced follow-up issue proposals: implementation for settled scope,
detailed planning or experiments for consequential unknowns. Reuse/link existing
issues rather than creating duplicates; explain when no additional issue is useful.
An implementation proposal cannot quietly select an unresolved contract or mark
dependent work ready. Split by bounded outcomes/dependencies, not every discussion
topic. Subsequent planning issues need more precise questions/outputs than the
loose starting issue, and links back through the discussion to the origin.

Create or update issues only on the user's request/authorization. Verify actual
origin, discussion-document/comment URLs, parent/related links and dependencies
using host tooling; preserve existing unrelated content. Never invent tracker IDs
or leave unverified placeholder links in a real issue. If unavailable, give a
clearly labeled proposal and state the linking/access gap. Keep requirements in
Linear for Unframed; Git holds implemented architecture/ADRs, not planning copies.
Always separate planning approval, issue creation, implementation kickoff, completed
verification and human merge. No automatic issue-execution pipeline is implied.
