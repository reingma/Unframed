# Performance review token analysis

Measured by the implementing agent on 2026-10-15 (UTC) for ENG-22. These are
content-only estimates, not Devin billing, and token counts alone do not prove
review quality.

## Scope and reproduction

- Superseded version: `e7a44b3`, which ran benchmarks through a bundled script
  and tooling guide. Gabriel's PR review made the skill analysis-only, so the
  script and guide were removed.
- Host baseline: `15d32d4`, the `main` revision before this skill.
- Final: the ENG-22 working tree containing this report.
- Python 3.12.13, `tiktoken 0.14.0`, `cl100k_base` and `o200k_base`. UTF-8 files
  are counted independently, including frontmatter, then summed. Discovery counts
  only the text between the YAML delimiters.

Run from the repository root; pass a revision to count that revision instead of
the working tree:

```sh
python - <<'PY'
import subprocess, sys, tiktoken
from pathlib import Path
rev = sys.argv[1] if len(sys.argv) > 1 else None
def read(p):
    if rev is None:
        return Path(p).read_text() if Path(p).exists() else ''
    r = subprocess.run(['git', 'show', f'{rev}:{p}'], capture_output=True, text=True)
    return r.stdout if r.returncode == 0 else ''
r = '.agents/skills/performance-review/'
files = {'SKILL': r + 'SKILL.md', 'examples': r + 'references/examples.md',
         'tooling': r + 'references/tooling.md',
         'principles': '.agents/skills/go-engineering/references/principles.md',
         'AGENTS': 'AGENTS.md', 'workflow': 'docs/workflow.md'}
for name in ['cl100k_base', 'o200k_base']:
    e = tiktoken.get_encoding(name)
    c = {k: len(e.encode(read(v))) for k, v in files.items()}
    c['discovery'] = len(e.encode(read(files['SKILL']).split('---', 2)[1]))
    print(name, c)
PY
```

## Loading assumptions

- Every review loads the entry point and the shared performance section of
  `principles.md`. Agents usually read the whole file, so the table counts all of
  it; the performance section alone is 403 / 401 tokens.
- The worked example is for calibration only. This report is off every loading path.

## Counts

Each pair is **cl100k / o200k**.

| Loading layer / file                       | Superseded (`e7a44b3`) |         Final |
| ------------------------------------------ | ---------------------: | ------------: |
| Discovery metadata                         |                61 / 60 |       66 / 66 |
| Activated `SKILL.md`                       |              972 / 960 | 1,081 / 1,076 |
| Required shared `principles.md`            |          1,538 / 1,536 | 1,538 / 1,536 |
| Required skill context                     |          2,510 / 2,496 | 2,619 / 2,612 |
| Plus `references/tooling.md` (removed)     |          3,164 / 3,155 |             — |
| Optional `references/examples.md`          |              654 / 651 |     770 / 768 |
| Script and tests (removed)                 |          5,569 / 5,611 |             — |
| Whole runtime bundle (not default loading) |          7,849 / 7,881 | 1,851 / 1,844 |
| Host `AGENTS.md` + `docs/workflow.md`      |          2,579 / 2,575 | 2,564 / 2,562 |
| Required skill context + those host files  |          5,089 / 5,071 | 5,183 / 5,174 |

Compared with `main` before this skill, `AGENTS.md` grew by 29 / 29 tokens for
the routing line and `docs/workflow.md` by 49 / 49 for the skill entry and slash
command. The shared principles are unchanged.

## Observations and clear wins

- **Main win, by design:** the skill links to the existing measurement method
  and review feedback rule instead of restating them.
- **Superseded → final:** the entry point grew 109 / 116 tokens for the
  code-meshing step, the missing-performance-test step and class, the never-run
  rule and the exercise clarifications below. Reviews no longer load the 565-token
  tooling guide, so a review that previously needed it is 545 / 543 tokens cheaper. The example grew
  116 / 117 for a missing-performance-test finding.
- **No further trims:** the remaining entry point is the procedure, artifact
  checks, history and frontend handling, five finding classes, the finding format
  and the limits, all required by ENG-22 or Gabriel's review.
- The earlier draft (`b940359`) linked the shared feedback rule instead of
  restating it, saving 18 / 17 tokens; that change is kept.

## Review exercises

Both exercises ran in separate sessions without reading the worked example or
this report, on three synthetic changes to `BenchmarkLookup`: an atomic counter
an approved requirement asks for, a `fmt.Sprintf` key comparison labelled
"tidy", and a frontend page with 20 cover images and no measurements.

The [first exercise](https://app.devin.ai/sessions/e6d1c73234604b08897732420d832565)
tested the superseded version, which ran benchmarks itself. Its results led to
repeat-comparison guidance that the analysis-only version keeps as "reversing
results are uncertainty".

The [analysis-only exercise](https://app.devin.ai/sessions/d1a7ee03bb2b4c86a1f32706cd12e2e3)
used `24951f7` and was given the artifacts CI would attach, including two
contradicting comparisons for the counter.

| Case                     | Expected                         | Observed                                                                                                                        |
| ------------------------ | -------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Atomic counter           | Uncertainty; contention untested | Uncertainty for the reversing results; missing `b.RunParallel` contention benchmark, judged blocking.                           |
| `fmt.Sprintf` comparison | Avoidable overhead, blocking     | Avoidable overhead, blocking; tied 4 allocations per scanned record to the two `Sprintf` calls without a profile.               |
| Frontend page, no data   | Missing performance test         | Missing performance test, judged blocking; requested route sizes, image bytes and lazy loading, and LCP/CLS on a fixed fixture. |

It ran no benchmarks, tests, builds or profilers; its commands only located the
checkout, fetched the branch and read files. It reported four unclear points,
all fixed after the run: "consequential" was undefined (now: serves requests or
page renders, or claims a performance effect), noise was both dropped and
classed (now: report as uncertainty), the files under review were not in the
loading rules (now: read the changed files), and blocking on a missing frontend
measurement asked the author to build baseline tooling (now: non-blocking until
frontend measurement exists).

## Gaps

- **Discovery:** neither exercise session found the skill automatically; it
  exists only on this branch, so they read it from the fetched branch.
  Slash-command discovery in Devin CLI was not tested.
- The analysis-only exercise was not rerun after the four clarifications.
- Synthetic artifacts only. No real CI performance artifact, recorded history,
  frontend measurement or application-path workload exists yet to review.
