---
name: performance-review
description: Review the performance impact of backend or frontend changes by analyzing supplied performance artifacts against the changelog, diff and any recorded history, and flagging where new performance tests are needed. Never runs benchmarks or tests. Use when explicitly asked for a performance review, not for implementation or general code review.
---

# Performance review

## Required context and loading

Follow the host's authority and delivery rules; skill activation grants no permission.
A review does not authorize optimizing code, commits, approvals or merges.
Identify the review scope, baseline and current revisions, affected areas, any
agreed budgets, and the owner's applicable review responses in the request, PR
threads or issue comments. Collect the performance artifacts that exist: CI
benchmark and comparison output, load-test results, frontend build or runtime
measurements, profiles and recorded history. Report missing context as a gap
instead of assuming it.

Read the shared
[measurement method](../go-engineering/references/principles.md#performance-conscious-design-and-early-measurements)
to judge the artifacts; it applies to every stack. Load the
[worked review](references/examples.md) only to calibrate findings. Do not
routinely load examples or token reports.

## Procedure

Never run benchmarks, tests, profilers, builds or load. Measurements are produced
by CI or the change's author; if needed evidence is missing, request it.

1. Establish intent. Read the changelog between the baseline and current revisions,
   then the actual diff; the changelog is a summary, not exhaustive evidence. Note
   what each change is expected to cost and which areas it touches.
2. Check each artifact is comparable: same workload, machine, toolchain and
   configuration for baseline and current, with revisions, commands, units,
   repetitions and variance recorded. A failed or missing run gives no comparison.
3. Read the raw output, not only summaries. Report non-significant or reversing
   results as uncertainty, not as a change. Where recorded history exists for an affected area, compare
   the current result with its trend, and check that shifts line up with the
   changes the changelog records. History from another machine or configuration
   is context, not a before/after measurement.
4. Mesh the measurements with the code, reading the changed files. For each
   measured change, find the diff that explains it. For each changed path, look for costs such as extra work per
   item, allocations, I/O or queries in loops, contention, or added frontend
   bundle or image weight, and check whether a measurement covers them. Report
   measured facts separately from attribution hypotheses; a profile artifact can
   confirm attribution, a code reading cannot.
5. Flag missing performance tests: a changed or new path that no workload
   exercises, a benchmark that does not reach the change, or an affected area with
   no measurement. Name the workload and metric that would cover it.
6. Apply the owner's relevant review reasoning, classify each finding, and drop
   speculative or duplicate findings.

## Findings

- **Avoidable overhead:** a measured cost the change's intent does not require,
  such as a slowdown in an unrelated path or a cheaper design meeting the same
  requirements. Blocking when meaningful for the workload.
- **Accepted cost:** a measured cost the intended change explains, proportionate
  to its value and scope. Not blocking; record it. "New feature" does not excuse
  a cost unrelated to that feature, and a feature need not make the system slower.
- **Missing performance test:** a path from step 5. Blocking when the path serves
  requests or page renders, or the change claims a performance effect.
- **Uncertainty:** inconclusive or noisy results, incomparable artifacts, a
  missing baseline or history, or a suspected cost the code suggests but no
  measurement shows. State what would resolve it.
- **Budget violation:** a result outside an agreed budget. Only where one exists;
  never invent thresholds.

For each finding, give the workload and metric, baseline and current values with
variance and repetitions, the artifact and its revisions and environment, the
related diff location or changelog entry, the class and blocking rationale, and
any gap. Report improvements as measured facts. End with the artifacts inspected,
what was not measured, and the tests requested. Fixture or local results do not
establish production performance.

## Feedback refinements

Follow the shared
[review feedback rule](../go-engineering/references/principles.md#trade-offs-and-review-feedback).
Change this skill only after the owner's explicit agreement and a reviewed PR.

## Limits

This skill does not run anything, change application code, add CI gates or
automatic runs, require review on every PR, generate production load, use paid
telemetry or provision cloud resources.

## Unframed adapter

Follow [AGENTS.md](../../../AGENTS.md) for host policy. Committed performance
history lives in [perf/history/](../../../perf/README.md) as compressed records
carrying an environment factor; a request or PR names the records and attaches
`perf compare` output or a generated report as the measured artifact. Raw run
artifacts are local-only and unavailable to reviews. Records from a different
environment are context, not a before/after measurement. There is no CI
performance job and no frontend measurement yet; report a missing frontend
measurement as non-blocking until it exists.
