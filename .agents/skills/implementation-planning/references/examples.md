# Worked planning examples

These are illustrative, scripted fixtures, not real product specifications,
human approvals or executed measurements. Tracker references below are labels;
real records must use verified origin/discussion links as in the issue formats.

## Representative discussion: local catalog preview

Starting intent: plan a Go component that reads a user-selected local catalog
directory, parses JSON entries and previews them. No network/cloud capability is
approved. Workload and symlink policy are initially unknown. The facilitator
asks one decision question per turn and waits; answers below are scripted inputs.

1. **Scope/security.** Facilitator: "Inputs can escape the selected root through
   traversal or symlinks. Shall we reject symlinks in this first slice?" Answer:
   "Reject them; no network. I have not decided the file-size limit." Outcome:
   reject symlinks, validate root/path containment, do not log payloads; adversarial
   path and oversized-input behavior need verification. Size policy stays open.
2. **Performance.** "A repeated scan could dominate preview time. Should the
   first runnable slice measure representative scans before choosing a cache?"
   Answer: "Yes, compare small and large catalogs; no latency budget yet."
   Outcome: initial baseline at first executable scan, record revision/environment,
   catalog size/distribution, repeated scan timing and allocations; no fabricated
   timings, cache design or pass threshold. Clarify representative sizes first.
3. **Telemetry.** "One local process needs correlation, but not distributed
   tracing. Use a safe operation ID with completion/failure events initially?"
   Answer: "Yes; omit paths and contents." Outcome: propagate context/deadlines,
   safe correlation, verify completion/failure and redaction, assess volume/cost.
   **Justified deferral:** distributed spans/backend deferred because there is no
   cross-process path; revisit if an approved remote provider creates one.
4. **Libraries/interfaces.** "Follow the host-approved JSON API, and insulate
   preview callers from catalog storage without wrapping the JSON decoder itself?"
   Answer: "Explain alternatives before recommending a boundary." Outcome: compare
   API compatibility, null/presence semantics, security/maintenance and workload
   costs. Unframed prefers `encoding/json/v2` with `GOEXPERIMENT=jsonv2`; evidence
   must justify a departure rather than choosing legacy JSON from habit.
   A consumer-owned catalog reader is credible for directory and in-memory fixture
   implementations, with cancellation/error/ownership contracts. A pure entry-key
   parser with one stable implementation needs no interface just for mocking:
   API surface/indirection adds cost without useful replaceability. A future remote
   provider is not approved merely because the boundary can accommodate one.
5. **Verification.** "Keep reader contract tests across implementations and fuzz
   the uncertain entry-key boundary with an independent grammar oracle?" Answer:
   "Yes, plus invalid/null JSON, traversal, cancellation and redaction tests."
   Outcome: unit parser/failure checks, integration filesystem interactions,
   contract substitutions, meaningful fuzzing and early measurements. Intentional
   contract changes reopen planning; tests must not be weakened to stay green.
6. **Follow-up.** "File-size policy still blocks a ready implementation issue.
   Shall we draft a focused planning follow-up and a dependent implementation
   proposal, with neither authorized to run?" Answer: "Draft them; do not create
   issues or implement." Outcome: proposals below, explicitly not kickoff.

### Example meeting output

- Origin: Catalog-preview planning A; prior discussion: scripted rounds 1–6.
- Intent/invariants: local-only preview, root containment, no symlinks or sensitive
  diagnostics. No cache/remote storage requirements introduced.
- Coverage: security → round 1, performance → 2, telemetry → 3,
  libraries/interfaces → 4, verification → 5; deferrals and costs recorded above.
- Decisions versus unknowns: scripted fixture choices only; real human approval
  pending. File-size policy and representative workload sizes remain unresolved.
- Evidence: planned boundary/integration/contract/fuzz and baseline checks; none
  executed by this discussion. Topic coverage does not make implementation complete.
- Follow-ups: planning B resolves limits/workload; implementation C depends on B
  and a settled contract. Both trace to A and its discussion. Creation/kickoff pending.

### Example bounded issue proposals

**Planning B — Resolve catalog input limits and baseline workload.** Source: A,
rounds 1–2 and 6. Establish maximum file/aggregate handling semantics and realistic
catalog distributions; compare bounded-rejection options and compatibility costs.
Output: explicit limit decision or remaining gap, measurement fixture proposal,
updated acceptance for C. No arbitrary limit, performance target, experiment
execution or code change is approved. This is more precise than the loose origin.

**Implementation C — Implement local catalog preview boundary.** Source: A's
record; blocked by B. Outcome: approved root/symlink/input behavior with parse-once
representations and a small replaceable catalog-reader contract. Acceptance:
observable invalid-input, cancellation, redaction and substitution behavior;
first executable workload captures a reproducible baseline. Excludes remote
providers, cache policy, tracing services and production load. Limits/workload
are pending, not silently chosen. Separate human implementation kickoff required.

## Reusable exercise checks

Give a planner the starting intent above without the scripted answers. Observe
that it asks one meaningful question and waits rather than producing a settled
design. Supply answers turn by turn; request a compact record/proposals at the
end without authorizing issue creation or execution. Verify the record cites actual
discussion for every topic, retains consequential unknowns and the deferral,
compares libraries, distinguishes a replaceable reader from an unnecessary helper
interface, and does not claim tests/benchmarks ran. A fresh client's result is
evidence for that run, not proof all agents obey the guidance.
