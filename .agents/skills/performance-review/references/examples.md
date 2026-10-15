# Worked performance review

This calibration case uses artifacts from a controlled run of the Go skill's
[`BenchmarkLookup`](../../go-engineering/assets/lookup_test.go), 10 repetitions
per revision on one Linux machine. The change was temporary; the artifacts
calibrate the review, not application performance.

## Inputs

- **Changelog:** "Indexed lookup normalizes keys before reading the map. Adds
  `indexedLookup.Keys` for admin listing."
- **Diff:** `indexedLookup.Get` now looks up
  `Key{value: strings.Clone(key.String())}` instead of `key`. A new `Keys` method
  sorts and returns every key. No benchmark was added.
- **Artifact:** the benchstat comparison attached to the PR, baseline `main` and
  current at the PR head, same machine and toolchain.

```text
                              │       baseline      │                current                │
                              │       sec/op        │    sec/op     vs base                 │
Lookup/linear/records=8-8              31.57n ±  6%    31.31n ± 7%         ~ (p=0.541 n=10)
Lookup/indexed/records=8-8             14.90n ±  7%    18.68n ± 3%   +25.40% (p=0.000 n=10)
Lookup/linear/records=1024-8           1.542µ ±  6%    1.509µ ± 5%         ~ (p=0.393 n=10)
Lookup/indexed/records=1024-8          9.850n ± 10%   24.915n ± 4%  +152.96% (p=0.000 n=10)
```

Allocations were unchanged at 0 B/op and 0 allocs/op.

## Findings

> **Avoidable overhead, blocking.** Indexed lookups are 25% slower at 8 records
> and 153% slower at 1,024 (p=0.000, n=10, same machine and toolchain). The diff
> explains it: every `Get` now copies the key string before hashing. Parsed keys
> are already canonical because `ParseKey` builds them, so the copy buys no
> guarantee. Use `key` directly.
>
> **Missing performance test, not blocking.** `Keys` sorts every key on each
> call, O(n log n), and no benchmark exercises it. Add a `Keys` benchmark at the
> listing sizes admin pages expect. Not blocking because it serves only admin
> listing; blocking if it reaches a request path.
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
- If a rerun artifact reversed the indexed result, or the baseline came from a
  different machine, the finding would be an **uncertainty**, not a regression.
- If the artifact recorded a failed run, there is no comparison; request a
  passing run instead of producing one.
