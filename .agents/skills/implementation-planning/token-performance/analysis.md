# Implementation-planning token analysis

## Current ENG-27 remeasurement

ENG-27 replaced repeated host policy in `SKILL.md` with links. The method, issue
formats and worked examples were not changed. It was measured on 2026-10-15 (UTC)
with Python 3.12.13, `tiktoken 0.14.0`, `cl100k_base` and `o200k_base`, using the
command below with `draft` set to `27fcfc8be77e5e0fa84faba3f649ab30525ffcdd`, the
main revision containing the delivered skill. The final corpus is the ENG-27
working tree. Each pair is **cl100k / o200k**.

| Loading layer / file                       |      Baseline |         Final |       Delta |
| ------------------------------------------ | ------------: | ------------: | ----------: |
| Discovery metadata                         |       63 / 63 |       63 / 63 |       0 / 0 |
| Activated `SKILL.md`                       |     479 / 479 |     375 / 373 | −104 / −106 |
| Required `references/method.md`, unchanged | 1,064 / 1,060 | 1,064 / 1,060 |       0 / 0 |
| Required skill context                     | 1,543 / 1,539 | 1,439 / 1,433 | −104 / −106 |
| Whole runtime bundle (not default loading) | 3,309 / 3,285 | 3,205 / 3,179 | −104 / −106 |
| Host `AGENTS.md` + `docs/workflow.md`      | 2,987 / 2,988 | 2,377 / 2,374 | −610 / −614 |
| Required skill context + those host files  | 4,530 / 4,527 | 3,816 / 3,807 | −714 / −720 |

Required-context savings are 6.74% / 6.89%; the entry point alone is 21.71% /
22.13% smaller. The entry point removed repeated authority, tracker and
issue-creation rules. The unchanged method still requires open unknowns,
issue creation only on request, no invented tracker IDs or results, and separate
planning approval, kickoff, verification and human merge. The entry point keeps
its loading order, discussion pacing and optional-resource conditions.

The scripted exercise below ran against the earlier entry point and was not
repeated for this wording change. ENG-27 verification covers metadata, links,
explicit activation and repository checks only.

The sections after this one are historical ENG-21 records. Their "final" values
and working-tree statements describe that delivery, not the current files.

Measured and inspected by the implementing agent on 2026-10-15 (UTC), after
optimization and the scripted exercise. Content-only estimates, not Devin billing
or proof that token reduction preserves all agents' behavior.

## Scope and reproduction

First complete draft: `3f8ca3b0dfa3151595aff58ec05f5fb8b6ec59f0`.
Delivered runtime: `af6750925f63995b7baf943d2e996e5bf7d93f33`; runtime files remain
identical in the working tree containing this report. No pre-existing planning
skill was replaced. Pre-feature host comparison: `582dbf73f50f4971f2bd98a2690fb5ac9d27684c`.

Python 3.12.13, public `tiktoken` 0.14.0, `cl100k_base` and `o200k_base`; UTF-8 files,
including frontmatter and trailing newline, counted independently then summed.
Discovery counts only text between YAML delimiters (including its newlines),
not a client's wrapper. There are four runtime Markdown files, no scripts/assets,
and one separate verification report. Host inventory is shown separately below.
Hidden instructions, conversation/tool overhead, retrieval duplication, external
Engineering documents, unrelated skills and report context are excluded from
runtime totals. These exclusions are not evidence that clients omit them.

From repository root, with the stated tokenizer version available (no project
dependency or custom audit framework added):

```sh
python - <<'PY'
from pathlib import Path
import subprocess, tiktoken

draft = '3f8ca3b0dfa3151595aff58ec05f5fb8b6ec59f0'
root = Path('.agents/skills/implementation-planning')
host = ['AGENTS.md', 'docs/workflow.md', 'docs/specs/architecture.md',
        'README.md', 'CHANGELOG.md', 'docs/decisions/README.md', 'backend/go.mod']
files = [root/'SKILL.md', *sorted((root/'references').glob('*.md'))]
for name in ['cl100k_base', 'o200k_base']:
    enc = tiktoken.get_encoding(name)
    counts = {}
    for p in files + [Path(p) for p in host]:
        old = subprocess.check_output(['git', 'show', f'{draft}:{p}'], text=True)
        new = p.read_text(encoding='utf-8')
        a, b = len(enc.encode(old)), len(enc.encode(new))
        counts[str(p)] = (a, b)
        print(name, p, a, b, 'delta', b-a, 'change%', round(100*(b-a)/a, 2))
        if p.name == 'SKILL.md':
            print('discovery', *[len(enc.encode(s.split('---', 2)[1]))
                                 for s in [old, new]])
    for label, selected in [('required', [files[0], root/'references/method.md']),
                            ('whole runtime bundle', files)]:
        print(label, *[sum(counts[str(p)][i] for p in selected) for i in (0, 1)])
    report = root/'token-performance/analysis.md'
    print('report only, NOT runtime', len(enc.encode(report.read_text(encoding='utf-8'))))
PY
```

For an exact historical final corpus after later edits, use `git show` at the
delivered runtime revision rather than the current working files. The report is
inventoried/countable separately by this command; final report counts are also
recorded in PR verification, avoiding a self-referential report-size claim.

## Comparable draft → final counts

Each pair below is **cl100k / o200k**. Paths in the first table are relative to
this skill; metadata is a subset, not added again to activated instructions.

| Loading layer / file                       |         Draft |         Final |       Delta |
| ------------------------------------------ | ------------: | ------------: | ----------: |
| Discovery metadata                         |       63 / 63 |       63 / 63 |       0 / 0 |
| Activated `SKILL.md` (entire file)         |     704 / 705 |     479 / 479 | −225 / −226 |
| Required `references/method.md`            | 1,064 / 1,060 | 1,064 / 1,060 |       0 / 0 |
| Optional `references/issue-formats.md`     |     520 / 512 |     520 / 512 |       0 / 0 |
| Optional `references/examples.md`          | 1,246 / 1,234 | 1,246 / 1,234 |       0 / 0 |
| Required skill context                     | 1,768 / 1,765 | 1,543 / 1,539 | −225 / −226 |
| Whole runtime bundle (not default loading) | 3,534 / 3,511 | 3,309 / 3,285 | −225 / −226 |

Entry point savings: **31.96% / 32.06%**; required-context savings:
**12.73% / 12.80%**; whole-bundle savings: **6.37% / 6.44%**.
Optional resources add 1,766 / 1,746 only when needed. Draft and final have the
same method, examples and issue formats; compression removed repeated workflow
and record summaries, not topic requirements or safeguards.

| Host file (not part of portable skill) | Draft = final, cl100k / o200k |
| -------------------------------------- | ----------------------------: |
| `AGENTS.md`                            |                 1,409 / 1,406 |
| `docs/workflow.md`                     |                 1,578 / 1,582 |
| `docs/specs/architecture.md`           |                 2,256 / 2,267 |
| `README.md`                            |                 1,523 / 1,528 |
| `CHANGELOG.md`                         |                     234 / 234 |
| `docs/decisions/README.md`             |                     322 / 322 |
| `backend/go.mod`                       |                       38 / 38 |

Illustrative required skill + AGENTS/workflow subtotal: **4,755 → 4,530** /
**4,753 → 4,527** (−225/−226, **4.73% / 4.75%**); this is not all host context
or an actual client bill. Pre-feature AGENTS/workflow were 1,336/1,469 cl100k and
1,334/1,474 o200k: this new skill adds **73 + 109 / 72 + 108** host tokens for
the always-on trigger and discovery documentation. Do not confuse that necessary
feature cost with draft-to-final savings. Other host context varies with the task.

## Loading observations and retained costs

Intended disclosure: discovery metadata → full entry point → required method;
formats only when drafting records/issues, examples only for relevant exercises,
report only for analysis/review. The entry point names these conditions and does
not link the report as required context. No assets need blanket loading.

Parent `search_skills_and_knowledge` found the metadata and `skill_invoke`
activated the skill; one `skill_search` at the narrower `.agents/skills` path
returned no match. In the fresh exercise client, repository `skill_search`
discovered it and explicit activation succeeded. This is observed API behavior,
not independent verification of Devin CLI slash-command discovery or other clients.

Exercise events show the method loaded before discussion, formats only for the
requested final record, and no worked-example/report read. Host reads included
AGENTS, workflow (also re-read), architecture, decision index and Go module.
Unique file content for that path is **7,666 / 7,666** tokens (required skill +
formats + those host files); repeated output, tool wrappers and conversation cost
make actual context different. Progressive disclosure saves the 1,246/1,234-token
worked example from that path, not all host or conversation overhead.

The fresh final analysis found no additional unambiguous reduction. Kept the
explicit authority, wait/reflect pacing and optional-loading cues in the entry
point; retained one complete required method and meaningful on-demand formats and
examples. Moving required coverage to optional context or deleting trade-offs,
deferrals or example outcomes would weaken the method. No arbitrary target, new
token CI gate, paid service, platform installation or new dependency was added.

## Verification and observable exercise

[Scripted planning exercise](https://app.devin.ai/sessions/3218edea89a647089a798da67e1db005)
at the delivered runtime revision used separate fixture-user responses, not
Gabriel approvals or the optional model transcript. Observed one question followed
by a wait and answer reflection across seven rounds:

- R1–2: filesystem/symlink trust boundary, safe all-or-nothing failures; input
  limits and race/platform guarantees stayed open.
- R3: representative workload and first-executable application-path baseline;
  no invented numbers, cache/parallelism decision or executed experiment.
- R4: correlation, safe completion/failure diagnostics, privacy/cardinality/cost;
  distributed spans/backend deferred for the process-local path, with revisit trigger.
- R5–6: provisional host JSON v2 versus alternatives/evidence; consumer-owned
  loading boundary with directory/in-memory substitutions and ownership/errors/
  cancellation, while pure helpers remained concrete to avoid needless indirection.
- R7: unit/integration contract and failure evidence; meaningful independent
  properties prerequisite to fuzzing; fixtures do not prove filesystem safety.

The requested final record cited actual rounds and the verified conversation;
follow-up proposals separated workload/input, enforceable safety and schema/
contract planning from dependent, explicitly not-ready implementation. No fake
tracker IDs, issue writes, product kickoff, successful tests or deployment facts.
This is behavior evidence for one scripted run, not universal compliance or a
human-approved product plan. Real tracker creation/deduplication was deliberately
not exercised; external Engineering resources were not fetched by that client.

Parent checks passed: YAML parsing; matching name/directory and bounded description;
no permission/profile metadata; 35 local-file links before adding this report;
Prettier on changed Markdown; `git diff --check`; `make check test build` and
`make test-e2e` (15 frontend unit and 3 integration tests, Go race tests including
unchanged skill exercises). Diff inspection confirmed no Go skill/application or
blueprint changes. No benchmark was needed for this documentation-only runtime.
The final report is checked/countable separately, not mandatory instructions.
