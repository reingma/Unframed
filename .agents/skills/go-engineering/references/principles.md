# Shared Go engineering principles

This is the single method reference for Go implementation and its review mirror.
The implementation skill owns this material; the review skill references it
rather than maintaining a competing copy. Host-repository rules and approved
requirements govern authority and product contracts. These principles guide
judgment; patterns alone do not establish correctness.

## Boundary parsing and meaningful types

Parse untrusted input once into domain representations encoding established
invariants. Internal functions should consume those representations instead of
rechecking raw strings/numbers at every step. State the grammar, failure behavior,
and whether parsing preserves or normalizes information.

Use Go defined types/newtypes to prevent mixing concepts when that distinction
matters. A `type UserID string` distinguishes it from an article ID, but conversion
and the zero value still allow invalid contents. For stronger invariants, consider
unexported fields and parsing/constructors. Go structs still have a zero value,
and code in the declaring package can bypass construction: explicitly decide
whether zero is useful or must be rejected at an exported entry boundary.
Unmarshal/database paths must also establish the invariant, not bypass it.
Resolve missing, null and zero inputs at that parsing boundary according to the
contract, before constructing required domain values. Use nullable/presence-aware
wire representations when needed; distinguish omission from explicit null if
they have different meanings. Never ignore a parse error and consume its zero
result as valid, or invent a default merely to avoid handling absence.

Do not wrap every scalar or introduce constructor ceremony without a useful
guarantee. Prefer ergonomic representations when all underlying values are valid.
Do not mistake syntactic validity for existence, ownership, or authorization.
Permissions and mutable state need checks at their use/transaction boundary;
record their validity scope and handle cancellation, races, and stale facts.

## Interfaces at credible implementation boundaries

Prefer small consumer-owned interfaces where several credible implementations
could satisfy the needed contract: storage, external providers, or expensive
processing choices. Describe input/output invariants, failure and cancellation
semantics, and the alternatives the interface protects. Keep provider/router
details out of the domain contract where practical.

Concrete examples or a known replaceable choice justify a boundary, not just the
desire to mock every helper. Avoid interfaces for pure arithmetic/parsing helpers
or a single stable component without meaningful alternatives. Consider API
surface, indirection, allocation, testing, and reversal costs. An interface does
not make incompatible implementations interchangeable unless they preserve the
contract (including ordering, errors, consistency and resource ownership).

## Flexible, invariant-focused verification

Tests protect observable promises and failure semantics, not helper call counts,
internal layout, an algorithm, or arbitrary implementation-specific ordering.
Use unit tests for local behavior and integration tests for boundary interactions.
Contract tests should work across credible interchangeable implementations.

Three distinct outcomes matter:

1. **Implementation substitution:** the contract is unchanged; existing behavior
   tests should still pass without being rewritten to describe the new internals.
2. **Intentional contract change:** a protected promise changes; test failures
   prompt reconsideration of the issue/design and approval before tests change.
3. **Invariant violation:** approved behavior is broken; fix the code or escalate
   the unsuitable design, rather than weakening assertions to obtain green tests.

Property tests/fuzzing suit uncertain input boundaries when independent, meaningful
properties exist: accepted values satisfy the domain grammar, canonical values
round-trip, rejected data cannot escape as valid state, and arbitrary input cannot
panic. Seed interesting valid/invalid/boundary cases. Avoid circular oracles that
merely call the implementation twice; crash freedom alone is not correctness.
Test important zero values and external-state failure paths separately.
Not every constant response or trivial helper needs fuzzing or a new framework.

## Performance-conscious design and early measurements

Assess workload, asymptotic cost, allocations, correctness, security, maturity,
maintenance, and integration complexity. Standard-library status is not enough
to justify an expensive choice; advertised benchmark speed is not enough to
justify a dependency. Reuse existing libraries where suitable, avoiding obvious
poor complexity without waiting for a production incident. Compare plausible
alternatives when the decision matters; explain why a comparison is unnecessary
for a low-risk choice instead of manufacturing a benchmark.
Apply the host repository's approved library/API preferences to new code, tests
and examples, not only application handlers. Explain evidence-backed departures
for compatibility, contracts or workload costs; do not silently revert to a
legacy/default API for convenience. Reusable guidance does not impose one
project's toolchain-specific choice on every other project.

Introduce measurements alongside the first meaningful executable workload, before
later decisions hide their cost. Operation benchmarks measure a specific action;
application-path/load tests measure end-to-end behavior. Keep correctness tests
separate. Preserve raw output, base/current revisions, toolchain/OS/CPU, workload
size/distribution, units, concurrency, repetitions, and relevant configuration.
Control background load and compare like with like. Include setup costs when
they matter; excluding construction from a lookup benchmark cannot prove the
whole implementation faster or cheaper.

Inspect latency/throughput and memory/allocation evidence appropriate to the
workload. Repeat noisy measurements and profile when attribution matters. New
features can have justified cost, but "new feature" does not excuse arbitrary
regression; distinguish expected cost, avoidable overhead, noise, and missing
evidence. Budgets and regression gates require agreed expectations, not invented
universal thresholds. A missing baseline means no before/after claim, and a
fixture benchmark is not production evidence. Keep credentials and sensitive
workload data out of artifacts.

## Correlated request-path observability

Propagate `context.Context` across I/O/service boundaries, preserving cancellation
and deadlines instead of replacing it with `context.Background()`. Use existing
structured logger/tracing conventions to correlate events using a safe request or
trace identifier. A logger's `InfoContext` alone does not automatically add an ID
stored in context: derive logger attributes explicitly or use an established
context-aware handler. Across processes, use the agreed propagation protocol;
a process-local context value is not distributed tracing.

Instrument meaningful boundaries, time spent, and failures where they explain a
request path. Spans are useful when justified, but traceability does not mean
logging every function call or payload. Establish identifier trust/size policy at
ingress; never log credentials, sensitive payloads, or raw untrusted data just to
aid debugging. Review redaction, cardinality, volume, sampling, and overhead.
Observe identifiers, completion/failure, and context propagation in tests where
those are contracts; do not freeze incidental message text or timestamp format.

## Trade-offs and review feedback

Record the invariant or risk, alternatives, chosen option, cost, evidence, and
remaining uncertainty for consequential choices. Distinguish a defect/contract
violation, a justified trade-off, and a non-blocking preference. Not using a pattern
is not by itself a defect; calling a violation a trade-off does not excuse it.

Use relevant human review reasoning for the current change. Durable refinements
need explicit agreement and reviewed updates to this shared source. Do not
automatically generalize a situational response or silently rewrite the skill.
