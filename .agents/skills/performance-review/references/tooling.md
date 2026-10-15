# Go benchmark comparison tooling

[`benchcompare`](../scripts/benchcompare.go) wraps native `go test -bench` and
[benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat), pinned to a
reviewed `golang.org/x/perf` commit and run through `go run`. It needs no module
dependency or global install.

## Prerequisites

Git and Go. The first comparison downloads benchstat into the module cache; offline,
set `BENCHSTAT` to a benchstat binary. Set any environment the benchmarks need,
such as `GOEXPERIMENT`. Close heavy background work before measuring.

## Commands

Run from the repository root. `run` measures a revision in a temporary Git
worktree, or the working tree when `-rev` is omitted. Package or file arguments
are relative to `-dir`.

```sh
S=.agents/skills/performance-review/scripts/benchcompare.go
go run $S run -rev main -dir backend -bench . -count 10 -o .perf/NAME/base.txt ./...
go run $S run -dir backend -bench . -count 10 -o .perf/NAME/current.txt ./...
go run $S compare -o .perf/NAME .perf/NAME/base.txt .perf/NAME/current.txt
```

Each run file is benchstat-readable `go test` output preceded by `revision`,
`revision-ref`, `command`, `go-version`, `goexperiment`, `date` and `exit-code`
lines. A working-tree revision ends in `-dirty` when there are uncommitted changes.
`compare` writes `benchstat.txt` and `summary.txt` and prints both.

## Behavior

- `run` exits 1 when `go test` fails or no benchmark matches, keeping the output.
- `compare` exits 1 without a comparison when either file is missing, records a
  failure or nonzero exit, has no results, differs in `goos`, `goarch`, `cpu` or
  `pkg`, or has fewer than `-min-count` samples (default 6) for any benchmark.
- Otherwise it exits 0. This means a comparison exists, not that the cost is
  acceptable. The summary lists significant changes, non-significant results,
  new workloads with no baseline, and removed workloads. A toolchain difference
  is reported as part of the change.
- Each revision runs its own benchmarks. A benchmark whose code changed measures
  a different workload; check the diff before comparing it.

Verify the tool with
`GOEXPERIMENT=jsonv2 go test -race .agents/skills/performance-review/scripts/*.go`
or `make test-skills`.
