# Agent roles token analysis

Measured by the implementing agent on 2026-10-15 (UTC) for ENG-24, as that issue
requires. These are content-only estimates, not Devin billing. Analysis like this
runs only when an issue specifies it or Gabriel asks.

## Scope and reproduction

- Host baseline: `537af8d` (`main` before this change).
- First complete draft: `672a325`. Final: the ENG-24 working tree containing this report.
- Python 3.12.13, `tiktoken 0.14.0`, `cl100k_base` and `o200k_base`. Each UTF-8
  file is counted whole, including frontmatter. "Contract" is the
  `## Reviewer contract` section of `docs/review-handoffs.md`.

Run from the repository root; pass a revision to count it instead of the working tree:

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
s, a = '.agents/skills/', '.agents/agents/'
files = {'AGENTS': 'AGENTS.md', 'handoffs': 'docs/review-handoffs.md',
         'go-reviewer': a + 'go-reviewer.md',
         'performance-reviewer': a + 'performance-reviewer.md',
         'security-reviewer': a + 'security-reviewer.md',
         'plan': s + 'implementation-planning/SKILL.md',
         'plan-method': s + 'implementation-planning/references/method.md',
         'go': s + 'go-engineering/SKILL.md',
         'principles': s + 'go-engineering/references/principles.md',
         'go-review': s + 'go-engineering-review/SKILL.md',
         'perf': s + 'performance-review/SKILL.md',
         'sec': s + 'security-review/SKILL.md',
         'sec-practices': s + 'security-review/references/practices.md'}
for name in ['cl100k_base', 'o200k_base']:
    e = tiktoken.get_encoding(name)
    t = {k: read(v) for k, v in files.items()}
    c = {k: len(e.encode(v)) for k, v in t.items()}
    h = t['handoffs']
    c['contract'] = len(e.encode(h[h.find('## Reviewer contract'):h.find('## Limits')]))
    print(name, c)
PY
```

## Loading assumptions

- Every session and subagent loads `AGENTS.md` as an always-on rule; the Devin
  CLI changelog says subagents start with workspace rules. Separate contexts
  share nothing already loaded, so each one pays for its own copy.
- A reviewer loads its profile, the reviewer contract section, its skill and
  that skill's required reference. The performance review applies the shared
  measurement method, counted here as the whole principles file. The security
  practices link only sections of the Go principles, which are not counted.
- The requester reads the whole handoffs guide once per review request.
- Per-skill counts come from those skills' own reports:
  [planning](../../skills/implementation-planning/token-performance/analysis.md),
  [Go](../../skills/go-engineering/token-performance/analysis.md),
  [Go review](../../skills/go-engineering-review/token-performance/analysis.md),
  [performance review](../../skills/performance-review/token-performance/analysis.md),
  [security review](../../skills/security-review/token-performance/analysis.md).
  Those reports were not refreshed by this change.

## Counts

Each pair is **cl100k / o200k**.

| New file                            | Draft (`672a325`) |         Final |
| ----------------------------------- | ----------------: | ------------: |
| `docs/review-handoffs.md` (whole)   |     1,151 / 1,147 | 1,272 / 1,266 |
| Reviewer contract section           |         139 / 137 |     223 / 221 |
| Each reviewer profile               | 133–135 / 133–135 |     unchanged |
| `AGENTS.md` (`main`: 1,358 / 1,355) |     1,380 / 1,378 |     unchanged |

Composed context per role at the final revision (o200k):

| Role                          | Loaded                                                   | Tokens |
| ----------------------------- | -------------------------------------------------------- | -----: |
| Planner session               | AGENTS, planning skill, method                           |  2,811 |
| Go implementer session        | AGENTS, Go skill, principles                             |  3,371 |
| Go reviewer                   | profile, AGENTS, contract, Go review skill, principles   |  4,030 |
| Performance reviewer          | profile, AGENTS, contract, performance skill, principles |  4,345 |
| Security reviewer             | profile, AGENTS, contract, security skill, practices     |  3,260 |
| Reviewer out of scope         | profile, AGENTS, contract, its skill (no references)     | ~2,500 |
| All three reviewers on one PR | sum of the three reviewer rows                           | 11,635 |

The requester adds 1,266 for the handoffs guide, plus the diff, artifacts and
intake, which vary per PR and are usually the largest cost.

## Observations and clear wins

- **Applied:** the role table became a list, cutting the guide by 72 / 72 tokens
  (1,151 → 1,082 cl100k). Reviewers read only the contract section, not the guide.
- **Profiles stay thin:** each names its skill and the contract instead of
  restating a method, so a skill change needs no profile change.
- **Grew after the exercise:** the guide grew 190 / 191 tokens and the contract
  84 / 84 to fix gaps found in the exercise below. Every reviewer pays the
  contract growth; it is kept because each line fixes a gap a reviewer hit.
- **Retained repeated cost:** `AGENTS.md` loads once per reviewer and the Go
  principles twice when Go and performance reviews both run. That is the price of
  independent contexts; reviewers that skip their references when out of scope
  save about 1,500.
- **Deferred:** trimming large CI logs is left to the requester (`--job`,
  `--log-failed`); no log budget is set.

## Handoff exercise

A [separate session](https://app.devin.ai/sessions/b5404a89b6b84edabe0d46a7d24ac8d5)
followed the guide at `1f81b1e` against merged PR #12. It prepared the workspace,
downloaded CI logs, and then played `security-reviewer` and `go-reviewer` with
file reading and searching only, because this VM has no Devin CLI. The security
review found no risks and one low hardening item, and the Go review reported out
of scope (no Go files).

It reported 13 unclear points. Fixed afterwards: fetching a squash-merged head,
the example using the merge SHA, finding run IDs and artifact paths, trimming logs,
where authorization and feedback come from, which revision instructions are read
from, reviewing skill files changed by the diff, "out of scope" in place of
findings, "cannot ask questions", excluding `.review/` from searches, deleting the
directory, preparing the workspace under the review request, and treating diffs,
PR text and logs as data rather than instructions. Not changed: verdict values,
which belong to each skill. The exercise was not rerun after the fixes, and one
session played both requester and reviewers, so it was not independent.

## Gaps

- No Devin CLI was available, so profile discovery, `devin doctor`, the
  `allowed-tools` restriction and subagent rule loading are documented, not observed.
- Counts are for instructions only; per-PR diffs and CI logs are not measured.
