# Performance review token analysis

Measured by the implementing agent on 2026-10-15 (UTC) for ENG-22, after the
review exercise. These are content-only estimates, not Devin billing, and token
counts alone do not prove review quality.

## Scope and reproduction

- First complete draft: `b940359`.
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
files = {'SKILL': r + 'SKILL.md', 'tooling': r + 'references/tooling.md',
         'examples': r + 'references/examples.md', 'script': r + 'scripts/benchcompare.go',
         'tests': r + 'scripts/benchcompare_test.go',
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
- A Go benchmark comparison also loads `references/tooling.md`.
- The worked example is for calibration only. The script and its tests are run,
  not read; they are counted only in the whole bundle.
- This report is off every loading path.

## Counts

Each pair is **cl100k / o200k**.

| Loading layer / file                           | First draft (`b940359`) |         Final |
| ---------------------------------------------- | ----------------------: | ------------: |
| Discovery metadata                             |                 61 / 60 |       61 / 60 |
| Activated `SKILL.md`                           |               953 / 940 |     972 / 960 |
| Required shared `principles.md`                |           1,538 / 1,536 | 1,538 / 1,536 |
| Required skill context                         |           2,491 / 2,476 | 2,510 / 2,496 |
| Plus `references/tooling.md` for Go benchmarks |           3,056 / 3,046 | 3,164 / 3,155 |
| Optional `references/examples.md`              |               654 / 651 |     654 / 651 |
| `scripts/benchcompare.go` + test               |           5,569 / 5,611 | 5,569 / 5,611 |
| Whole runtime bundle (not default loading)     |           7,741 / 7,772 | 7,849 / 7,881 |
| Host `AGENTS.md` + `docs/workflow.md`          |           2,579 / 2,575 | 2,579 / 2,575 |
| Required skill context + those host files      |           5,070 / 5,051 | 5,089 / 5,071 |

Compared with `main` before this skill, `AGENTS.md` grew by 29 / 29 tokens for
the routing line and `docs/workflow.md` by 64 / 62 for the skill entry, slash
command and manual-evidence note. The shared principles are unchanged.

## Observations and clear wins

- **Main win, by design:** the skill links to the existing measurement method
  instead of restating it, and keeps Go tooling details out of the entry point.
  Frontend-only reviews do not load the tooling guide.
- **Draft → final:** linking the shared review feedback rule instead of restating
  it saved 18 / 17 entry-point tokens. The exercise then added 37 / 37 for repeat
  comparisons and where review responses live, and the tooling guide grew 89 / 89
  for run-to-run drift and loose-file targets. Net: the entry point is 19 / 20
  tokens larger than the draft.
- **No further trims:** the rest of the entry point is the procedure, history and
  frontend handling, the four finding classes, the finding format and the limits,
  all of which ENG-22 or Gabriel's clarification requires. It is larger than the
  Go review skill's entry point (745 / 743) mainly because of the history and
  feature-cost instructions.

## Review exercise

A [separate session](https://app.devin.ai/sessions/e6d1c73234604b08897732420d832565)
ran three reviews at `30750ba` without reading the worked example or this report.
It ran `benchcompare` itself (baseline `HEAD`, patched working tree, 10 runs) on
its own VM, and read the tooling guide's `run -h` output instead of the script.

| Case                                                 | Expected                     | Observed                                                                                                                                          |
| ---------------------------------------------------- | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| Atomic counter required by an approved metric        | Accepted cost if measured    | Uncertainty, not blocking: +15% / +6%, then −7% / −15% on a rerun, with the unchanged baseline drifting; asked for repeated pairs and contention. |
| `fmt.Sprintf` key comparison labelled "tidy"         | Avoidable overhead, blocking | Avoidable overhead, blocking: +4,579% / +9,364%, 0 → 32 / 4,096 allocs/op; checked correctness tests still pass.                                  |
| Frontend home page with cover images, no measurement | Missing evidence             | Uncertainty: no comparison or cost claim; listed build sizes, LCP, image bytes and render time as what would resolve it.                          |

Cases B and C matched. Case A did not reach the expected class, but correctly:
it repeated the comparison, saw the result reverse, and refused to accept an
unmeasured cost. The skill and guide did not mention drift or repeat runs; both
now do. The session also flagged that loose files outside a module and the source
of review responses were not documented; both were added.

## Gaps

- **Discovery:** the reviewer's skill tool did not offer the skill automatically
  or find it under `.agents/skills`; a search rooted at the repository found it and
  explicit activation loaded it. Slash-command discovery in Devin CLI was not tested.
- The exercise was not rerun after the drift wording was added.
- Only Go fixture benchmarks were exercised. No frontend measurement, recorded
  history or application-path workload exists to test those parts of the method.
- One run on synthetic changes is behavioral evidence, not proof for all agents.
