# Roles and review handoffs

Follow [AGENTS.md](../AGENTS.md) for authority, scope and merge policy. This guide
owns how the five roles are started and how reviews are handed off. Each role's
method lives in its skill; this guide does not restate it.

## Roles

| Role                 | How it runs                                                                                                              | Method                                                                                           |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ |
| Planner              | Its own interactive terminal session: run `devin` in the checkout, then `/implementation-planning ENG-<n>`               | [implementation-planning](../.agents/skills/implementation-planning/SKILL.md)                    |
| Implementer          | Its own interactive terminal session, only after Gabriel's explicit authorization: `/go-engineering ENG-<n>` for Go work | [go-engineering](../.agents/skills/go-engineering/SKILL.md), otherwise [AGENTS.md](../AGENTS.md) |
| Go reviewer          | `go-reviewer` subagent profile                                                                                           | [go-engineering-review](../.agents/skills/go-engineering-review/SKILL.md)                        |
| Performance reviewer | `performance-reviewer` subagent profile                                                                                  | [performance-review](../.agents/skills/performance-review/SKILL.md)                              |
| Security reviewer    | `security-reviewer` subagent profile                                                                                     | [security-review](../.agents/skills/security-review/SKILL.md)                                    |

Planner and implementer are separate primary sessions so Gabriel converses with
each directly and neither sees the other's conversation. They are not subagents:
the Devin CLI cannot start a primary session as a custom profile, and a subagent
cannot ask the user questions; its parent only summarizes its result. A finished
plan never starts implementation.

The reviewer profiles live in [.agents/agents/](../.agents/agents/), the
project location for [Devin CLI custom subagents](https://docs.devin.ai/cli/subagents#custom-subagents),
which are experimental. Each allows only the `read`, `grep` and `glob` tools; on
a subagent profile that list is a restriction, not an auto-approval. Reviewers
run no code, tests, benchmarks or scanners and cannot edit, use Git, the web or
MCP tools. Clients that do not load these profiles, including Devin cloud
sessions, can follow the review skill in a separate session instead, but that
read-only limit is then only instructed, not enforced.

## Requesting a review

Reviews run only when Gabriel asks for them, one reviewer or a named set, not on
every PR. The requesting session (Gabriel's or the implementer's) prepares
everything, because reviewers can only read files:

1. Create `.review/<pr>-<short-sha>/` in the checkout; `.review/` is gitignored.
   Check out the PR head there with `git worktree add .review/<pr>-<short-sha>/tree <head-sha>`
   and save `git diff <base-sha>...<head-sha>` as `diff.patch`.
2. Download the CI/CD artifacts the review needs into `artifacts/`, for example
   with `gh run download <run-id>` or `gh run view <run-id> --log`. Record any
   artifact that does not exist rather than producing it locally.
3. Write `intake.md`: repository and PR URLs, the issue, its approved scope and
   the authorization link, base and head SHAs, the files above, relevant
   architecture, ADR and deployment facts, and Gabriel's applicable feedback.
   Add the baseline, workload and environment for performance, and exposure,
   trust boundaries and triggers for security. Write missing facts as gaps.
4. Ask for each reviewer by name, for example "run the security-reviewer
   subagent on `.review/12-537af8d/intake.md`". Reviewers may run in parallel.
5. Relay each report unchanged. The implementer fixes actionable findings within
   scope and escalates design, security-boundary or requirement changes.
6. After new commits, earlier reports are stale for the claims the commits touch.
   Prepare a new directory at the new head and rerun the affected reviewers.
   Remove finished worktrees with `git worktree remove`.

Gabriel reviews and decides whether to merge.

## Reviewer contract

- Review only the pinned tree, diff and artifacts in the intake. Do not assume
  the author's conversation, uncommitted files or other revisions.
- If the change is outside your role, report "out of scope" rather than a pass.
  If an input is missing, report it as not assessed or ask the requester for it.
- Apply Gabriel's feedback to the current review only. Durable changes to a
  profile or skill need his explicit agreement and a reviewed PR.
- Return the reviewed head SHA, the skill followed, findings in that skill's
  format, the inputs inspected, and what was not assessed. Never fix code.

## Limits

Each reviewer is its own agent with its own context window and model calls, so
cost adds per reviewer. Profiles pin no model: reviewers use the default subagent
model, which `agent.subagent_model` in the user config can override. There are no
triggers, schedules, automatic or repeated reviews, or nested subagents; ENG-25
covers future triggers. Built-in Devin Review is a separate integration that does
not use these profiles.
