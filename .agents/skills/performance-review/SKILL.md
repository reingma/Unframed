---
name: performance-review
description: Review the performance impact of backend or frontend changes by running comparable baseline and current measurements, inspecting raw output, and judging cost against the changelog, diff and any recorded history. Use when explicitly asked for a performance review, not for implementation or general code review.
---

# Performance review

## Required context and loading

Follow the host's authority and delivery rules; skill activation grants no permission.
A review does not authorize optimizing code, commits, approvals or merges.
Identify the review scope, baseline and current revisions, affected areas, workloads,
existing test commands and results, environment, any agreed budgets, recorded
performance history, and the owner's applicable review responses in the
request, PR threads or issue comments. Report missing
context as a gap instead of assuming it.

Read the shared
[measurement method](../go-engineering/references/principles.md#performance-conscious-design-and-early-measurements)
before measuring; it applies to every stack. Load the
[tooling guide](references/tooling.md) to run Go benchmark comparisons, and the
[worked review](references/examples.md) only to calibrate findings. Do not
routinely load examples, scripts or token reports.

## Procedure

1. Establish intent. Read the changelog between the baseline and current revisions,
   then the actual diff; the changelog is a summary, not exhaustive evidence. Note
   what each change is expected to cost and which areas it touches.
2. Select workloads for those areas: operation benchmarks, application-path or
   load tests, and frontend build or runtime measurements. If nothing measures an
   affected area, report the missing measurement. A static reading or fixture
   benchmark is not evidence of application performance.
3. Run the baseline and current revisions on the same machine, toolchain and
   configuration, with repetitions. Record revisions, commands, workload and
   data, environment, units and variance. Report no comparison when the baseline
   is missing, a run failed, or the configurations differ. Repeat the comparison
   when a change is small or the machine is shared, and attribute only changes
   that persist across repeats.
4. Inspect the raw output and comparisons, not only summaries. Where recorded
   history exists for an affected area, compare the current result with its
   trend too, and check that shifts line up with the changes the changelog
   records. History from another machine or configuration is context, not a
   before/after measurement.
5. Profile or run focused measurements when attribution matters. Report measured
   facts separately from attribution hypotheses.
6. Apply the owner's relevant review reasoning, classify each finding, and drop
   speculative or duplicate findings.

## Findings

- **Avoidable overhead:** a measured cost the change's intent does not require,
  such as a slowdown in an unrelated path or a cheaper design meeting the same
  requirements. Blocking when meaningful for the workload.
- **Accepted cost:** a measured cost the intended change explains, proportionate
  to its value and scope. Not blocking; record it. "New feature" does not excuse
  a cost unrelated to that feature, and a feature need not make the system slower.
- **Uncertainty:** inconclusive or noisy results, a missing baseline, workload or
  history, or an unattributed change. State what would resolve it.
- **Budget violation:** a result outside an agreed budget. Only where one exists;
  never invent thresholds.

For each finding, give the workload and metric, baseline and current values with
variance and repetitions, the revisions and environment, the related changelog or
diff entry, the class and blocking rationale, and any gap. Report improvements as
measured facts. End with the commands run, where the raw output is, and what was
not measured. Local results do not establish production performance.

## Feedback refinements

Follow the shared
[review feedback rule](../go-engineering/references/principles.md#trade-offs-and-review-feedback).
Change this skill only after the owner's explicit agreement and a reviewed PR.

## Limits

This skill does not change application code, add CI gates or automatic runs,
require review on every PR, generate production load, use paid telemetry or
provision cloud resources.

## Unframed adapter

Follow [AGENTS.md](../../../AGENTS.md) for host policy, including
`GOEXPERIMENT=jsonv2` for Go commands. Keep raw results in the gitignored `.perf/`
directory and summarize them with raw excerpts in the PR or review. No performance
history or frontend tooling is defined yet; until they are, rerun the baseline
and report missing frontend measurements.
