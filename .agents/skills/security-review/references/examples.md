# Worked security reviews

These synthetic cases calibrate findings against the
[security practices](practices.md). They show how exposure, authentication and
triggers change risk. They are not Unframed requirements or a complete review,
and the reviewer ran nothing; the evidence is the diff and CI output.

## Same weakness, two contexts

A diff adds `GET /articles/{slug}/raw` to the Go router. The handler reads
`os.ReadFile(filepath.Join(contentDir, r.PathValue("slug")))` and returns the
bytes. A Next.js page forwards its `slug` route parameter to that endpoint.
Separately, a new `cmd/seed` tool reads `filepath.Join(contentDir, *name)` from a
command-line flag.

> **Suspected risk, high if the backend is publicly reachable; blocking.**
> `articles.go:24` joins an unparsed slug into a file path. `PathValue` returns
> the unescaped segment, so `..%2F..%2Fgo.mod` survives router path cleaning as
> `../../go.mod` and `filepath.Join` resolves it outside `contentDir`. The Next.js
> page makes this reachable from any browser without authentication. Nothing is
> deployed, so exposure is unknown, not low. Proposed test: a handler test
> requesting `/articles/..%2F..%2Fgo.mod/raw` that expects 400 and no file read.
> Mitigation: parse the slug into a key type that rejects separators and dots.
>
> **Hardening, not blocking.** `cmd/seed/main.go:18` has the same join, but its
> input is a flag typed by the developer running it, who already has file access.
> Reusing the slug parser would keep the two paths consistent.

The pattern is identical; the trigger and who controls the input decide the class.

## CI supply chain

A workflow change adds a `pull_request_target` trigger to label new PRs. The job
checks out `${{ github.event.pull_request.head.sha }}`, runs `npm ci && npm run
label`, uses `actions/labeler@v5` by tag, and sets `permissions: write-all`.

> **Confirmed risk, high; blocking.** `ci.yml:12-30`: `pull_request_target` runs
> with a write token and repository secrets, and the job executes the PR's own
> `package.json` scripts. Anyone who can open a PR can run code with write access.
> The workflow file is the evidence; no test is needed. Mitigation: use
> `pull_request`, or never check out or run head code in this job, and grant only
> `pull-requests: write`.
>
> **Hardening, not blocking.** Pin `actions/labeler` to a commit SHA like the other actions.

## Dependency change with no scan output

`frontend/package.json` adds `marked` for article rendering and the lockfile adds
it with no install script. CI passed; it runs no dependency audit.

> **Not assessed.** Known vulnerabilities in `marked` or its dependencies: CI
> produced no audit output.
>
> **Suspected risk, high once user-controlled Markdown is rendered; blocking.**
> `article.tsx:9` passes `marked.parse(body)` to `dangerouslySetInnerHTML`, and
> `marked` does not sanitize HTML. A body containing `<img src=x onerror=...>`
> would run script in readers' browsers. Proposed test: render that body and
> assert no `onerror` attribute reaches the DOM. Mitigation: sanitize with an
> allow-list.

Without the `dangerouslySetInnerHTML` use there would be no suspicion and no test
to propose; the missing audit stays a gap, not a finding.
