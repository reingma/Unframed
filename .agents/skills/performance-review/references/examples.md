# Worked performance review

This calibration case comes from a controlled run of the Go skill's
[`BenchmarkLookup`](../../go-engineering/assets/lookup_test.go) on one Linux
machine. The change was temporary; the results validate the tooling, not
application performance.

## Inputs

- **Changelog:** "Indexed lookup normalizes keys before reading the map."
- **Diff:** `indexedLookup.Get` now looks up
  `Key{value: strings.Clone(key.String())}` instead of `key`. `Key` values are
  already canonical because `ParseKey` builds them, so the copy adds no guarantee.
- **Workload:** already-built stores of 8 and 1,024 records, one lookup per
  iteration; 10 runs per revision with `benchcompare`.

## Comparison excerpt

```text
                              │ .perf/demo/base.txt │          .perf/demo/slow.txt          │
                              │       sec/op        │    sec/op     vs base                 │
Lookup/linear/records=8-8              31.57n ±  6%    31.31n ± 7%         ~ (p=0.541 n=10)
Lookup/indexed/records=8-8             14.90n ±  7%    18.68n ± 3%   +25.40% (p=0.000 n=10)
Lookup/linear/records=1024-8           1.542µ ±  6%    1.509µ ± 5%         ~ (p=0.393 n=10)
Lookup/indexed/records=1024-8          9.850n ± 10%   24.915n ± 4%  +152.96% (p=0.000 n=10)
```

Allocations were unchanged at 0 B/op and 0 allocs/op.

## Findings

> **Avoidable overhead, blocking.** Indexed lookups are 25% slower at 8 records
> and 153% slower at 1,024 (p=0.000, n=10, same machine and toolchain). The
> changelog explains the copy as normalization, but parsed keys are already
> canonical, so the cost buys no guarantee. Use `key` directly.
>
> **Uncertainty, not blocking.** The linear store shows no significant change.
> That is inconclusive, not proof of equality, and this fixture says nothing
> about request-path latency.

## How the classes would change

- If the changelog had introduced case-insensitive keys that a parsed `Key` does
  not already guarantee, the same measured cost would be an **accepted cost**,
  recorded with its numbers. It would still be checked for a cheaper design.
- With an agreed lookup budget, a result above it would be a **budget
  violation**; without one, do not invent a threshold.
- An unchanged rerun showed every result as `~`. A build failure produced no
  comparison: `benchcompare` exited 1 because the run file recorded the failure.
