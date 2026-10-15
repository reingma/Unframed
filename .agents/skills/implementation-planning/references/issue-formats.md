# Discussion and follow-up formats

Adapt these headings to scope; they are not permission or mandatory boilerplate.
Use verified tracker URLs, not these placeholders, in real records. Preserve
unrelated content when updating an existing issue. Create issues only when asked.

## Discussion record

```text
Origin: [origin issue](<verified issue URL>)
Prior discussion: [document/thread](<verified source URL>)
Intent, invariants, scope/exclusions and dependencies:
Facts / assumptions / unknowns:
Alternatives, recommendation and costs:
Decisions: who agreed to what; do not imply implementation kickoff
Coverage: topic → discussion link/round → concern/outcome/open question
  Security; performance/early evidence; telemetry; verification; libraries/interfaces
Deferrals: rationale, revisit condition and responsible follow-up
Evidence: planned commands/workloads/checks, not reported passes
Next work: bounded issue proposals + origin/parent/dependency links
Approval and implementation kickoff: actual status, otherwise pending
```

## Implementation issue requested after discussion

```text
Title: Implement <bounded outcome>
Origin / planning source: [origin](<verified issue URL>)
Approved discussion: [planning record](<verified document/comment URL>)
Outcome and approved scope / exclusions:
Contracts / invariants / failure semantics:
Decisions and rationale: selected option, alternatives, costs, contextual deferrals
Dependencies / blocking decisions: link existing issues; do not hide open choices
Verification / acceptance: observable behavior and evidence; agreed budgets only
Delivery: host documentation/changelog/review requirements
Kickoff: separate explicit implementation authorization required
```

## Detailed follow-up planning issue

```text
Title: Resolve <specific consequential uncertainty>
Origin: [origin issue](<verified issue URL>)
Previous discussion: [record](<verified document/comment URL>)
Known facts / provisional decisions / assumptions still unverified:
Decision to resolve and why it blocks <linked bounded work>:
Credible options, constraints and trade-offs:
Evidence needed: proposed experiment/workload/check, prerequisites, approval limits
Expected output: decision rationale, coverage record, updated bounded proposals
Acceptance: discussion/evidence addressed and remaining gaps visible, not code complete
Relationships: verified parent/related/blocking issues
Authority: planning only; experiment execution and implementation need explicit kickoff
```

Do not turn unknowns into requirements to make an issue look ready. A planning
issue may conclude with another precise planning proposal; it should not carry
unrelated future features forward or fabricate a deployment/acceptance threshold.
