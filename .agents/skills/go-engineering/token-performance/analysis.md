# Go engineering token-performance analysis

## Scope and method

Measured on 2026-10-15 (UTC), after optimization. Baseline is independently reread
from `9d5b5b5cc90e1ab26481e5a67bf50d4b0ced50e1`; final is this PR's working tree.
The runtime corpus is `SKILL.md`, both `references/*.md` and both `assets/*.go`,
relative to this skill directory. Selected host context is root `AGENTS.md` and
`docs/workflow.md`. Architecture/changelog changes and this new report are
inventoried separately below, not silently folded into required skill guidance.

Python 3.12.13, `tiktoken 0.14.0`, `cl100k_base` and `o200k_base`; read UTF-8 text
and count `len(encoding.encode(text))`, summing individual file counts for groups.
Discovery counts use the raw text between the first two `---` delimiters,
including adjacent newlines. Full entry-point counts already include metadata;
do **not** add discovery again to activated totals. The unchanged starting corpus
reproduces the original review counts with this representation.

Content-only estimates: neither tokenizer is claimed to be Devin's tokenizer or
billing metric. Tool envelopes, history, repeated reads, platform overhead, remote
issues/Engineering resources and other task-specific repository intake are excluded.
Selected host totals are not a complete invocation cost. Token reduction alone
does not demonstrate equal behavioral quality or cheaper wall-clock execution.

Reproduce from repository root using a one-off external installation (not a
repository dependency or CI tool). This session used the same pinned package in
`$HOME/.devin-files/token-audit-libs` instead of the example target below:

```sh
python -m pip install --target /tmp/unframed-token-analysis tiktoken==0.14.0
PYTHONPATH=/tmp/unframed-token-analysis python - <<'PY'
from pathlib import Path
import hashlib, subprocess, tiktoken

base = '9d5b5b5cc90e1ab26481e5a67bf50d4b0ced50e1'
skill = Path('.agents/skills/go-engineering')
runtime = sorted(p for p in skill.rglob('*') if p.is_file()
                 and 'token-performance' not in p.parts)
host = [Path('AGENTS.md'), Path('docs/workflow.md')]
extra = [Path('docs/specs/architecture.md'), Path('CHANGELOG.md')]
report = skill / 'token-performance/analysis.md'
texts = {}
for p in runtime + host + extra + [report]:
    old = subprocess.run(['git', 'show', f'{base}:{p}'],
                         capture_output=True, text=True)
    texts[p] = (old.stdout if old.returncode == 0 else '', p.read_text())
    print('sha256', p, hashlib.sha256(p.read_bytes()).hexdigest())
print('tiktoken', tiktoken.__version__, 'baseline', base)
for name in ['cl100k_base', 'o200k_base']:
    enc = tiktoken.get_encoding(name)
    counts = {p: [len(enc.encode(s)) for s in pair]
              for p, pair in texts.items()}
    print(name)
    for p, (a, b) in counts.items():
        print(p, a, b, 'delta', b-a,
              'change%', round(100*(b-a)/a, 2) if a else 'new')
    print('discovery', [len(enc.encode(s.split('---', 2)[1]))
                        for s in texts[skill / 'SKILL.md']])
    core = [skill / 'SKILL.md', skill / 'references/principles.md']
    groups = {'core': core, 'optional': [p for p in runtime if p not in core],
              'runtime bundle': runtime, 'selected host': host,
              'core + selected host': core + host}
    for label, files in groups.items():
        a, b = [sum(counts[p][i] for p in files) for i in range(2)]
        print(label, a, b, 'delta', b-a, 'change%', round(100*(b-a)/a, 2))
PY
```

The script inventories the report itself without requiring it at runtime.
Missing baseline report text counts as zero because the report is new. Full Git
history containing the baseline is required; a shallow clone may need that commit.

## Comparable measurements

Paths below are skill-relative except host documents. Arrows mean baseline → final.

| Loading layer / file                  |   cl100k_base |    o200k_base |
| ------------------------------------- | ------------: | ------------: |
| Discovery: name/description payload   |       64 → 50 |       64 → 50 |
| Activated: `SKILL.md` (83 → 53 lines) |     931 → 538 |     928 → 533 |
| Required: `references/principles.md`  | 1,494 → 1,494 | 1,492 → 1,492 |
| Optional: `references/examples.md`    | 2,063 → 2,063 | 2,062 → 2,062 |
| Optional: `assets/lookup.go`          |     499 → 499 |     499 → 499 |
| Optional: `assets/lookup_test.go`     | 2,341 → 2,341 | 2,342 → 2,342 |
| Host: `AGENTS.md`                     | 1,336 → 1,336 | 1,334 → 1,334 |
| Host: `docs/workflow.md`              | 1,307 → 1,469 | 1,311 → 1,474 |

| Cumulative scenario / change     |                   cl100k_base |                    o200k_base |
| -------------------------------- | ----------------------------: | ----------------------------: |
| Required core                    | 2,425 → 2,032 (−393; −16.21%) | 2,420 → 2,025 (−395; −16.32%) |
| Optional documentation + assets  |         4,903 → 4,903 (0; 0%) |         4,903 → 4,903 (0; 0%) |
| All runtime Markdown             |  4,488 → 4,095 (−393; −8.76%) |  4,482 → 4,087 (−395; −8.81%) |
| Runtime bundle, excluding report |  7,328 → 6,935 (−393; −5.36%) |  7,323 → 6,928 (−395; −5.39%) |
| Selected host context            |  2,643 → 2,805 (+162; +6.13%) |  2,645 → 2,808 (+163; +6.16%) |
| Core + selected host context     |  5,068 → 4,837 (−231; −4.56%) |  5,065 → 4,833 (−232; −4.58%) |

Entry point alone: −393/−395 tokens (−42.21%/−42.56%). Discovery: −14 tokens
(−21.88%) in both. The required core is approximately 2.03k, slightly above the
proposed 1.8–2.0k range; that was a hypothesis, not a gate. Retained portable
authority, loading instructions and explicit JSON v2 guidance justify the cost.

Supplementary inventory: `docs/specs/architecture.md` is 2,149 → 2,189 / 2,160 →
2,200 tokens; `CHANGELOG.md` is 178 → 209 in both. Those additions cost another
40/40 and 31/31 tokens when read. The new `token-performance/analysis.md` is
2,676/2,678 tokens (baseline absent), inventoried separately by the reproduction
command; its cost is not counted as zero merely because it is optional. Reading
every stored artifact now adds this entire report to 6,935/6,928 tokens and exceeds
the old bundle. It must
remain off routine loading paths; no report link or mandatory read was added to
the skill or shared principles.

## Applied wins and final loading assessment

- Replaced repeated principle explanations with a six-step execution/navigation
  procedure and compact evidence handoff. The shared method remains authoritative.
- Shortened discovery description while retaining Go implementation intent and
  the planning/review exclusions; the skill name is unchanged.
- Explicitly made relevant example sections and exercise source on-demand.
  Running native checks does not require reading source into agent context.
  Avoiding a full optional read keeps 4,903 tokens out of that scenario; this is
  conditional avoidance, **not** additional measured compression or billing savings.
- Documented one maintained, off-path report per skill in existing workflow
  guidance. Its host-context increase is included rather than hidden.

The [Agent Skills specification](https://agentskills.io/specification) intends
metadata at discovery, entry-point instructions at activation and resources as
needed. Observed here: `skill_search` for `go-engineering` against this checkout
returned no results, while `search_skills_and_knowledge` found its updated
metadata and explicit `skill_invoke` returned the new body. Shared principles were
then read explicitly; invocation output did not automatically include references,
assets or this report. No claim is made about unseen client reads, cached context,
automatic CLI discovery or exact runtime overhead. The search limitation already
existed at baseline; explicit activation remains usable. Other clients need their
own discovery/loading verification.

## Preservation and verification

Manual baseline/final diff review retained explicit human authorization, approved
scope, consequential-gap escalation, independent review, truthful evidence, and
no provisioning/tracing service, automatic pipeline, global/profile configuration,
security bypass, production load or merge authorization. Host rules are unchanged.
JSON v2 and `GOEXPERIMENT=jsonv2` remain explicit for application/tests/examples.

`git diff 9d5b5b5cc90e1ab26481e5a67bf50d4b0ced50e1 --
.agents/skills/go-engineering/references .agents/skills/go-engineering/assets`
is empty. Those unchanged files preserve null/zero/presence parsing, credible
interfaces/substitution contracts, independent fuzz properties, measurements,
cancellation/request correlation, data exclusion and human-feedback safeguards.
This is author self-analysis, not an independent behavioral review.

Actual checks on Linux/amd64, Go 1.26.9, Node 24.21.0, npm 11.19.0:

- Frontmatter parsed with Prettier's YAML parser; name/description constraints,
  Markdown local-file links and whitespace checked; passed. Report links are not
  runtime dependencies. Explicit activation passed as described above.
- `make check test build`: passed Go formatting/vet/race tests (including skill
  checks), frontend formatting/ESLint/TypeScript, 15 frontend unit tests and both
  independent production builds.
- `make test-e2e`: passed all three existing production Playwright tests. No new
  application flow or UI-driven testing session was needed.
- `GOEXPERIMENT=jsonv2 go test -run '^$' -fuzz FuzzParseKey -fuzztime 5s -parallel 2
.agents/skills/go-engineering/assets/*.go`: passed, 352,965 executions.
- `GOEXPERIMENT=jsonv2 go test -run '^$' -bench BenchmarkLookup -benchmem
-benchtime 100ms -count 3 .agents/skills/go-engineering/assets/*.go`: passed;
  all samples reported 0 B/op and 0 allocs/op. These exercise-only measurements,
  taken alongside repository checks, establish neither production performance nor
  a historical speed regression/improvement.

## Verdict and deferred reductions

Clear-win compression and loading improvements are implemented; final remeasurement
found no further unambiguous reduction worth applying. Remaining entry-point
overlap is routing and evidence prompts, not a second method explanation. Removing
the authority/limits or JSON reminder would reduce portability/safety. Splitting
required principles risks missed safeguards for a small cost; removing meaningful
examples sacrifices optional verification without improving required loading.
No custom audit framework, new repository dependency, CI budget, platform configuration or
application behavior change is introduced. Human review/merge remains required.
