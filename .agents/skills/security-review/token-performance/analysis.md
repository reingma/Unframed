# Security review token analysis

Measured by the implementing agent on 2026-10-15 (UTC) for ENG-23. These are
content-only estimates, not Devin billing, and token counts alone do not prove
review quality.

## Scope and reproduction

- Host baseline: `daeac3d`, the `main` revision before this skill.
- First complete draft: `3c12811`.
- Final: the ENG-23 working tree containing this report.
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
r = '.agents/skills/security-review/'
files = {'SKILL': r + 'SKILL.md', 'practices': r + 'references/practices.md',
         'examples': r + 'references/examples.md',
         'AGENTS': 'AGENTS.md', 'workflow': 'docs/workflow.md'}
for name in ['cl100k_base', 'o200k_base']:
    e = tiktoken.get_encoding(name)
    c = {k: len(e.encode(read(v))) for k, v in files.items()}
    c['discovery'] = len(e.encode(read(files['SKILL']).split('---', 2)[1]))
    print(name, c)
PY
```

## Loading assumptions

- Every review loads the entry point and `references/practices.md`. The practices
  link the shared Go boundary-parsing and review-feedback sections; reviewers in
  the exercise read only those sections, so they are not counted here.
- The worked reviews are for calibration only. This report is off every loading path.

## Counts

Each pair is **cl100k / o200k**.

| Loading layer / file                       | Draft (`3c12811`) |         Final |
| ------------------------------------------ | ----------------: | ------------: |
| Discovery metadata                         |           77 / 76 |       73 / 72 |
| Activated `SKILL.md`                       |         984 / 975 | 1,022 / 1,013 |
| Required `references/practices.md`         |         513 / 514 |     513 / 514 |
| Required skill context                     |     1,497 / 1,489 | 1,535 / 1,527 |
| Optional `references/examples.md`          |         794 / 797 |     794 / 797 |
| Whole runtime bundle (not default loading) |     2,291 / 2,286 | 2,329 / 2,324 |
| Host `AGENTS.md` + `docs/workflow.md`      |     2,639 / 2,636 | 2,639 / 2,636 |

Compared with `main`, `AGENTS.md` grew by 28 / 28 tokens for the routing line and
`docs/workflow.md` by 47 / 46 for the skill entry and slash command.

## Observations and clear wins

- **By design:** the skill bundles no tooling, links the shared boundary-parsing
  and feedback rules instead of restating them, and keeps the three worked
  reviews optional.
- **Draft → final:** the discovery description shrank 4 / 4 tokens, which every
  session pays. The entry point grew 38 / 38 tokens for four clarifications from
  the exercise below; they close gaps a reviewer hit, so they are kept.
- **No further trims:** the practices are already one short paragraph per area,
  and the entry point holds the context intake, procedure, four finding classes,
  conditional severity, finding format and limits that ENG-23 requires.

## Review exercise

A [separate session](https://app.devin.ai/sessions/80db484006514762b28c8bcc8ad9cfe1)
reviewed five synthetic changes at `3c12811` without reading the worked reviews.

| Case                                             | Observed                                                                                                    |
| ------------------------------------------------ | ----------------------------------------------------------------------------------------------------------- |
| Unauthenticated `POST /admin/reindex`            | Confirmed risk, high if the listener is reachable; noted `:8080` binds all interfaces. Blocking.            |
| `BACKEND_URL` renamed `NEXT_PUBLIC_`             | Confirmed risk against the server-only contract; low for a plain origin, higher with credentials. Blocking. |
| `curl \| bash` in a read-only `pull_request` job | Confirmed supply-chain risk, medium given the stated limits, higher on other triggers. Blocking.            |
| `fetch(searchParams.url)` preview page           | Confirmed SSRF and unbounded response, high; noted JSX escaping rules out XSS. Blocking.                    |
| CI `npm audit` ReDoS in a lint-only plugin       | Hardening, low: the trigger is the repository's own config. Not blocking.                                   |

It ran no code, tests or scanners; its commands only located the checkout, fetched
the branch and read files. It proposed tests only where a suspicion remained.
Four points it found unclear were fixed afterwards: the host's `npm audit` rule
versus never running scanners (running checks belongs to implementation and CI),
a supplied audit artifact versus the adapter's "not assessed" (use it when
supplied), blocking on a conditional severity (decide from the highest rating),
and regression tests for confirmed risks (allowed in the mitigation). The
exercise was not rerun after these fixes.

## Gaps

- **Discovery:** the exercise session activated the skill only after checking out
  the branch, because the skill is not on `main`. Slash-command discovery in Devin
  CLI was not tested.
- Synthetic changes only. No real CI security artifact exists to review, and
  nothing is deployed, so exposure-dependent ratings stay conditional.
