# Worked choices and executable exercise

These are synthetic engineering examples, not Unframed content requirements or
production implementations. Run native commands from the repository root with
its pinned Go toolchain. The two Go files in `assets/` form a standalone example
package compiled by file list, not a new application module or dependency.

## Types: protect a useful invariant, not every scalar

A defined type separates concepts:

```go
type UserID string
type RecordID string
```

It prevents accidentally passing one to a function requiring the other, but
`UserID("")` remains possible. If a record key must follow a grammar, the bundled
`Key` uses an unexported field and `ParseKey`: one to 64 ASCII bytes, starting with
a lowercase letter, followed by lowercase letters, digits or hyphens. Parsing
preserves the canonical input; it neither trims nor lowercases it. Exported
`Read` rejects the invalid zero key, then internal lookup receives a typed key
without reparsing raw input. Declaring-package code can still violate that
representation; encapsulation is not a proof against all misuse.

The grammar is a demonstration choice, not a universal slug/identifier policy.
Authorization and whether a record exists remain separate checks. By contrast,
wrapping a loop index in a private struct with a constructor would add friction
without protecting a useful domain invariant.

## Nullable wire input: resolve absence before construction

`ParseKeyInput` is a JSON-boundary adapter for a required key. Its temporary wire
representation uses `*string`: an absent field and explicit JSON `null` both
decode to `nil`. The adapter rejects either before dereferencing or calling
`ParseKey`. A present empty string is not null, but still violates the key grammar
and is rejected by `ParseKey`. Only successful parsing returns an established
domain key; failed parsing returns the invalid Go zero value **with an error**.
Callers must handle that error rather than consume the returned key anyway.

```go
var input struct {
    Key *string `json:"key"`
}
if err := json.Unmarshal(raw, &input); err != nil {
    return Key{}, ErrInvalidKey
}
if input.Key == nil {
    return Key{}, ErrInvalidKey
}
return ParseKey(*input.Key)
```

`TestParseKeyInput` covers valid input, a null document, missing/null/empty keys,
wrong JSON types, invalid grammar and malformed JSON. Nullable values remain at
the wire boundary; downstream lookup still consumes `Key`, not `*string`.

This example deliberately treats missing and null as equivalent errors for a
required field. If an optional update distinguishes "not supplied" from "clear
this value", use presence-aware parsing instead; do not collapse those states or
invent a default domain key. It uses the standard decoder's field matching and
unknown/duplicate-field behavior, not a universal JSON API policy. Never decode
directly into a domain representation in a way that bypasses its constructor:
JSON `null` can leave non-pointer Go values unchanged, including invalid zero
values. The existing zero-key guard at the public read boundary remains useful
for callers that bypass this adapter; it does not replace parsing wire input.

## Interfaces: replace an implementation, preserve its promises

The consumer needs only this contract:

```go
type Lookup interface {
    Get(context.Context, Key) (Record, error)
}
```

For valid keys it returns the matching record or `ErrNotFound`, honoring context
cancellation. A file-backed store or remote provider would need an adapter to
establish the same guarantees. The exercise's linear and indexed in-memory
implementations demonstrate replacement and cost without requiring infrastructure.
Records in the fixture have unique keys; duplicate-key resolution, mutation,
consistency and provider protocols are not modeled here.
The consumer expects a non-nil context, source and logger; handling configuration
mistakes in those dependencies is outside this bounded example.

The interface belongs to the consumer's boundary, not to every parsing/helper
function. An interface around `ParseKey` merely to mock its string checks would
add abstraction without credible alternatives. In a real issue, assess whether
the provider choice can actually vary before introducing this boundary.

## Three verification outcomes

- **Substitution:** `TestReadContract` runs the same behavior tests with the
  linear and indexed stores. It checks results/errors/zero input/cancellation,
  not search counts, map layout, or helper calls.
- **Approved design needs reconsideration:** allowing uppercase keys would
  deliberately change the parser's protected grammar. Existing rejection/property
  tests should fail. Discuss canonicalization, ambiguity and compatibility and
  obtain approval before changing the contract and tests together.
- **Bug:** removing the length limit or allowing traversal syntax violates the
  current contract. Restore the implementation; do not weaken rejection tests.

`FuzzParseKey` has an independent regex grammar oracle, canonical preservation
and round-trip properties, and invalid-state rejection. Interesting seeds cover
empty input, Unicode, NUL, path syntax, and length boundaries. Arbitrary inputs
also exercise crash freedom, but no-panic alone would not establish correctness.
The regex expresses the contract rather than copying the parser's byte loop.

```sh
gofmt -l .agents/skills/go-engineering/assets/*.go
go vet .agents/skills/go-engineering/assets/*.go
go test -race .agents/skills/go-engineering/assets/*.go
go test -run '^$' -fuzz FuzzParseKey -fuzztime 10s -parallel 2 .agents/skills/go-engineering/assets/*.go
```

Named-file commands work outside the application module. `make check-skills` and
`make test-skills` run static/race checks, including fuzz seeds, not a fuzz campaign
or a performance gate. Save discovered fuzz failures appropriately; do not dismiss
them merely because the seed suite passes.

## Early, scoped performance evidence

`BenchmarkLookup` compares an already-built linear store and indexed store for
a successful lookup at the end of datasets of eight and 1,024 unique records.
Each run checks the result and reports time and allocations. This deliberately
tests a particular search cost; it excludes index construction, retained memory,
logging, remote I/O and end-to-end request time. Compare misses, access
distributions, construction and memory separately if they matter to an actual
decision. Both alternatives use the standard library and introduce no dependency.

Before changing a meaningful workload, preserve raw results and identifying
metadata outside source control unless the repository specifies otherwise:

```sh
git rev-parse HEAD
git status --short
go version
go env GOOS GOARCH GOEXPERIMENT
# Also record CPU, load, workload, configuration and whether source is modified.
go test -run '^$' -bench BenchmarkLookup -benchmem -count 5 .agents/skills/go-engineering/assets/*.go
```

Run comparable base/current revisions in equivalent environments without
competing heavy work. Use the changelog plus diff to understand intended costs.
Inspect distributions/noise and use existing comparison/profiling tools where
needed; no threshold or performance CI gate is supplied here. Comparing these
two alternatives is not a historical regression measurement, and absence of a
baseline prevents a before/after claim. A local fixture result does not establish
Unframed performance or justify a library replacement.

## Request-path evidence without a tracing service

`Read` passes the original context to `Lookup.Get` and adds the supplied request
identifier to its structured logger explicitly. `TestRequestCorrelationAndFailure`
observes correlation and a deadline at that boundary and reads actual JSON log
output for start/failure events without freezing message text or timestamps. It
also checks a provider's sensitive error contents do not leak to logs.
The success-path check observes completion and excludes record payloads from logs.

The identifier is a safe locally supplied fixture value. Real HTTP ingress must
establish its trust/length policy rather than blindly accepting a header. This is
process-local propagation, not distributed tracing; a multi-process path needs
the host project's agreed protocol. Missing identifiers, sampling,
redaction policy and observability cost deserve checks when they are
part of the actual issue. Do not introduce a collector, log every helper, or log
raw payloads merely to demonstrate traceability.

## Representative bounded implementation exercise

Invoke the skill, read the shared principles, then inspect/run the example:

1. State the grammar, zero-key, missing-record and context contracts above.
2. Explain the opaque key versus a defined string and the replaceable lookup
   boundary versus a parser interface. Note the declaring-package/zero loopholes.
3. Run the same contract suite against both implementations, fuzz the parser,
   and inspect the correlation/failure log checks.
4. Run repeated operation benchmarks. Record workload/environment/raw outputs and
   explain excluded costs, uncertainty and why no production/baseline claim follows.
5. Classify substitution, intentional grammar change, and invalid-key acceptance
   separately. If experimenting with a mutation, use a disposable copy, not the
   application's code or weaker tests.
6. Report commands, actual outcomes, trade-offs, gaps, and the independent review
   still needed. This exercise demonstrates the method, not agent isolation,
   permission enforcement, cloud activation, or application integration.
