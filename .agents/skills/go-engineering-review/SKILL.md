---
name: go-engineering-review
description: Review Go changes against the shared Go engineering principles, reporting evidence-backed findings that separate confirmed defects, uncertainties, justified trade-offs and non-blocking preferences. Use when explicitly asked to review Go changes, not for implementation or planning.
---

# Go engineering review

## Required context and loading

Follow the host's authority and delivery rules; skill activation grants no permission.
A review does not authorize fixes, commits, approvals or merges.
Read the issue's approved scope and acceptance criteria, the actual diff at the
reviewed revision, relevant tests and evidence, host instructions, architecture,
decisions, and the human owner's applicable review responses. Report missing
context as a gap instead of assuming it.

Read the [shared principles](../go-engineering/references/principles.md) before
judging. They are the single implementation/review method; do not keep a review copy.
Load [review examples](references/examples.md) only to calibrate findings or run
a review exercise. Do not routinely load examples or token reports.

## Procedure

1. Establish what the change promises: approved scope, invariants, failure and
   zero-value behavior, and any intentional contract change.
2. Check for missing contracts, such as unparsed input, bypassable invariants,
   ignored errors, lost context, unsafe logging or unmeasured costly choices.
3. Check for unnecessary restriction or abstraction. A named type or interface is
   not by itself evidence of good design; ask what guarantee or alternative it serves.
4. Check that tests protect invariants and failure paths and would survive an
   implementation substitution. An intentional contract change goes back to the
   issue or design; never recommend weakening assertions to obtain a pass.
5. Check the performance, security or maintenance evidence behind each trade-off.
   Calling something a trade-off does not exempt it from approved requirements.
   Claim runtime behavior only from evidence you inspected, such as CI results or
   attached output; do not run checks yourself, and request missing evidence.
6. Apply the owner's relevant review reasoning, classify each finding, and drop
   speculative or duplicate findings.

## Findings

- **Defect:** a confirmed requirement, contract or invariant violation, or a
  security risk. Blocking.
- **Uncertainty:** missing evidence or context that could hide a defect. State
  what would resolve it. Blocking when the risk is consequential, including an
  approved criterion with no test or other evidence.
- **Trade-off:** a justified choice whose cost and evidence are recorded. Not blocking.
- **Preference:** a non-blocking suggestion. Missing a preferred pattern alone is
  not grounds to reject a change.

For each finding, give the location at the reviewed revision, the evidence, the
affected invariant or risk, the class and blocking rationale, and any untested
gap. End with a verdict, the evidence inspected, and what was not reviewed or
lacks evidence.

## Feedback refinements

Use the owner's reasoning in the current review. Propose enduring refinements
separately, naming the source and the general rule. Change the shared principles
or this skill only after the owner's explicit agreement and a reviewed PR. Do not
silently generalize a situational comment or rewrite guidance automatically.

## Limits

This skill does not wire automatic reviews, require review on every PR, apply
fixes, or replace a security audit or performance gate.

## Unframed adapter

Follow [AGENTS.md](../../../AGENTS.md) for host policy, including the JSON API
preference, and [workflow guidance](../../../docs/workflow.md) for discovery and
loading. The review examples are synthetic, not application requirements.
