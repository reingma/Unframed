# Go engineering review token analysis

Measured by the implementing agent on 2026-10-15 (UTC) for ENG-20, after the
review exercise and its follow-up edit. These are content-only estimates, not
Devin billing, and token counts alone do not prove review quality.

## Scope and reproduction

- First complete draft: `8f734805ebea00b5a4d4bac0c3993dcef3a2e9f9`.
- Host baseline: `5d4e3f9`, the `main` revision before this skill.
- Final: the ENG-20 working tree containing this report.
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
r = '.agents/skills/go-engineering-review/'
files = {'SKILL': r + 'SKILL.md', 'examples': r + 'references/examples.md',
         'principles': '.agents/skills/go-engineering/references/principles.md',
         'AGENTS': 'AGENTS.md', 'workflow': 'docs/workflow.md'}
for name in ['cl100k_base', 'o200k_base']:
    e = tiktoken.get_encoding(name)
    c = {k: len(e.encode(read(v))) for k, v in files.items()}
    c['discovery'] = len(e.encode(read(files['SKILL']).split('---', 2)[1]))
    print(name, c)
PY
```

## Counts

Each pair is **cl100k / o200k**.

| Loading layer / file                       | First draft (`8f73480`) |         Final |
| ------------------------------------------ | ----------------------: | ------------: |
| Discovery metadata                         |                 54 / 55 |       54 / 55 |
| Activated `SKILL.md`                       |               733 / 731 |     745 / 743 |
| Required shared `principles.md`            |           1,538 / 1,536 | 1,538 / 1,536 |
| Required skill context                     |           2,271 / 2,267 | 2,283 / 2,279 |
| Optional `references/examples.md`          |               772 / 777 |     794 / 798 |
| Whole runtime bundle (not default loading) |           3,043 / 3,044 | 3,077 / 3,077 |
| Host `AGENTS.md` + `docs/workflow.md`      |           2,486 / 2,484 | 2,486 / 2,484 |
| Required skill context + those host files  |           4,757 / 4,751 | 4,769 / 4,763 |

Compared with `main` before this skill, the shared principles grew by 14 / 14
tokens for the review-skill link, `AGENTS.md` by 30 / 30 for the routing line, and
`docs/workflow.md` by 51 / 51 for the skill entry and slash command.

## Observations and clear wins

- **Main win, by design:** the review skill links to the implementation skill's
  principles instead of mirroring them. A review copy would add about 1,500
  tokens per activation and a second source to keep in sync.
- **Draft → final:** the entry point grew 12 / 12 tokens. The exercise showed the
  examples called missing test evidence a defect while the class definitions make
  it an uncertainty; the final wording says an approved criterion with no evidence
  is a blocking uncertainty, and the examples now match.
- **No further trims:** the remaining entry-point text is the procedure, the four
  finding classes, the finding format, the refinement rule and the limits, all of
  which ENG-20 requires. Its required context is 2,283 / 2,279 tokens, versus 1,996 /
  1,993 for the Go implementation skill's entry point plus the same principles.
- Examples and this report are off the mandatory loading path. A routine review
  reads only the entry point and principles.

## Review exercise

A [separate session](https://app.devin.ai/sessions/19e60f1fda9b4af89666e837afc1a73e)
reviewed four synthetic fixtures at the first draft, without reading the review
examples or token reports. The fixtures were different from the worked cases.

| Fixture                                             | Expected                                        | Observed                                                                                                                                           |
| --------------------------------------------------- | ----------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Slug grammar check removed as a "perf trade-off"    | Defect, blocking                                | Defect, blocking, for the removed check and deleted tests; said the trade-off label is no exemption; unmeasured perf claim as uncertainty.         |
| Cache test asserting map size, call count, log text | Blocking; layout and log text brittle           | Missing contract tests as blocking uncertainty; layout and exact log text as blocking defect; kept the call count as an observable promise.        |
| Handler with single-use parser interface and fake   | Blocking evidence gap; abstraction a preference | Fake parser and missing 400 test as blocking uncertainty; parser interface and `RetryCount` as preferences; store interface justified by Postgres. |
| Sorted slice for a stated memory limit, benchmarked | Trade-off, not blocking                         | Trade-off, not blocking, without inventing a latency budget; evidence questions as non-blocking uncertainties.                                     |

Every verdict and blocking decision matched the expected outcome. The one
difference was the defect-versus-uncertainty label for missing tests, fixed as
described above. The reviewer stated that nothing was runnable and that every
runtime point came from reading the fixture text.

## Gaps

- **Discovery:** the reviewer's skill tool first found only the skills in its
  pre-cloned `main` checkout. It loaded the review skill after scanning its own
  checkout path, which is the documented fallback. In this session, path search
  and explicit activation both worked. Slash-command discovery in Devin CLI was not tested.
- The reviewer also had personal notes on Gabriel's review preferences, so the
  run is not fully isolated from host context.
- One scripted run on synthetic text is behavioral evidence, not proof for all
  agents. The exercise was not rerun after the wording fix, and no real PR was reviewed.

## ENG-22 update

ENG-22 made reviews inspect-only: the procedure now claims runtime behavior only
from inspected evidence and requests missing evidence instead of running checks,
and the summary lists the evidence inspected. Measured with the same command:
`SKILL.md` went from 745 / 743 to 762 / 760 tokens (+17 / +17), required skill
context from 2,283 / 2,279 to 2,300 / 2,296, and `references/examples.md` from
794 / 798 to 796 / 800. The ENG-20 review exercise was not rerun.
