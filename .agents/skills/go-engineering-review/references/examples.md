# Worked review cases

These synthetic cases calibrate findings against the
[shared principles](../../go-engineering/references/principles.md). They reuse the
key-lookup contract from the
[implementation exercise](../../go-engineering/references/examples.md): `ParseKey`
accepts lowercase ASCII letters, digits and hyphens of 1 to 64 bytes, and
`Lookup.Get` returns the matching record or `ErrNotFound`, honoring cancellation.
They are not Unframed requirements or a complete review.

## Invariant violation presented as a trade-off

The diff removes the 64-byte limit from `ParseKey`, deletes the over-limit
rejection test, and explains: "Performance trade-off: the length check is redundant."

> **Defect, blocking.** `parse.go:31` no longer enforces the approved 64-byte
> maximum, and `parse_test.go` drops the only test protecting it. Keys of any
> length now escape as valid state. No benchmark shows the check is costly, and a
> claimed cost could not override the approved grammar anyway. Restore the check
> and test, or reopen the contract on the issue.

The word "trade-off" does not change the class. A violated requirement stays a defect.

## Brittle implementation-specific tests

A new indexed store's tests assert `len(store.index) == 3`, that a fake source was
called exactly once, and an exact log message. No test covers `ErrNotFound`,
invalid keys or cancellation.

> **Defect, blocking.** `store_test.go` does not check the `Lookup` contract:
> missing keys, invalid input and cancellation are untested. The existing
> assertions would also fail if the linear store were swapped in. Run the shared
> contract tests against both stores and drop the layout and call-count checks.
> Assert the log message only if it is a contract.

If contract tests already existed and only the extra assertions were brittle, this
would be a non-blocking preference, not a defect.

## Unnecessary abstraction

The diff adds `type KeyParser interface { Parse(string) (Key, error) }` with one
implementation, so tests can inject a fake parser returning arbitrary `Key` values.
It also wraps a loop counter as `type Attempt int`.

> **Defect, blocking.** The fake parser in `handler_test.go` returns keys that
> never went through `ParseKey`, so handler tests no longer exercise the grammar
> boundary. Test through the real parser instead.
>
> **Preference, non-blocking.** `KeyParser` has no credible alternative
> implementation, and `Attempt` guards against no confusion between concepts.
> Both add indirection without a guarantee; inline them unless a planned use exists.

Removing the interface is a preference. Bypassing the invariant it enabled is the defect.

## Justified trade-off

The diff replaces the indexed map with a sorted slice and binary search. The PR
cites the issue's memory constraint and includes benchmarks covering construction
and lookup for 10,000 records, with toolchain, CPU, five repetitions and raw output.
Lookups are about 30% slower; resident memory is about 60% lower.

> **Trade-off, non-blocking.** `index.go` trades lookup latency for memory, matching
> the issue's stated constraint, with comparable measured evidence. Contract tests
> still pass for both stores.
>
> **Uncertainty, non-blocking.** The benchmark uses fixture data, not a production
> key distribution. Confirm the trade-off once a representative workload exists.

A reviewer who prefers the map should not reject this change. The evidence
supports the choice, and the preference is not backed by a violated requirement.
