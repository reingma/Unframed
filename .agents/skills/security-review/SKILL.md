---
name: security-review
description: Review backend, frontend, dependency or CI changes for security risk by reading the code against specific security practices in its actual exposure and execution context, using only artifacts CI/CD produced. Never runs code, tests or scanners, and is not a comprehensive vulnerability test. Use when explicitly asked for a security review, not for implementation or general code review.
---

# Security review

## Required context and loading

Follow the host's authority and delivery rules; skill activation grants no permission.
A review does not authorize fixes, commits, approvals, merges, scans or exploit attempts.
Identify the review scope, baseline and current revisions, approved requirements
and the owner's applicable review responses in the request, PR threads or issue
comments. Establish the context that decides risk: architecture and deployment
topology, public or internal exposure, authentication and authorization, trust
boundaries, data sensitivity, input sources, and what triggers the code (a
request, a page render, an operator, a build, a CI event). Report missing context
as a gap instead of assuming it; an unknown deployment fact is not a protection.

Read the [security practices](references/practices.md) the code is reviewed
against. Load the [worked reviews](references/examples.md) only to calibrate
findings. Do not routinely load examples or token reports.

## Procedure

Never run code, tests, scanners, builds or exploit attempts. Use only artifacts
CI/CD produced, such as check results, dependency audit or scan output and build
logs. If a needed artifact is missing, report what it would have shown as not assessed.

1. Establish intent. Read the changelog between the revisions, then the actual
   diff, including dependency manifests, lockfiles, CI workflows and configuration.
2. Place each changed path: deployed runtime code, a privileged or admin path,
   development or build tooling, or CI and supply chain. Local-only tooling is
   not risk-free by default.
3. Trace each changed path from its inputs to what it reaches, reading the
   boundaries it touches, and check it against the relevant practices.
4. Read any CI/CD security artifacts in that context. A scanner match alone does
   not prove exploitability; a clean, failed or missing scan does not prove absence.
5. Where a specific, reasoned suspicion remains that code is exploitable, propose
   a targeted test for the author or CI to add: the input, the path it takes and
   the expected safe outcome. Do not propose tests without such a suspicion.
6. Apply the owner's relevant review reasoning, classify each finding, and drop
   speculative or duplicate findings.

## Findings

- **Confirmed risk:** code, configuration or artifact evidence shows an
  exploitable weakness with a reachable trigger. Blocking unless severity is low
  and the owner accepts it.
- **Suspected risk:** a specific weakness the code suggests but the evidence
  cannot confirm. Give the proposed test. Blocking when it would be high severity
  if confirmed.
- **Hardening:** a cheaper or safer pattern with no identified exploit path.
  Not blocking.
- **Not assessed:** an area or artifact outside what was reviewed, or context
  that decides risk and is missing.

Rate severity from exposure, authentication needed, trigger, data sensitivity and
impact, with the rationale. When an unknown fact changes the rating, give the
rating conditionally, for example "high if the backend is publicly reachable";
never downgrade on a hypothetical restriction.

For each finding, give the location at the reviewed revision, the affected
behavior, reachability and triggering prerequisites, existing controls, impact,
evidence and uncertainty, class and severity rationale, and mitigation. End with
a verdict, the code and artifacts inspected, and what was not assessed. A review
does not establish that the change is secure.

## Feedback refinements

Follow the shared
[review feedback rule](../go-engineering/references/principles.md#trade-offs-and-review-feedback).
Change this skill only after the owner's explicit agreement and a reviewed PR.

## Limits

This skill does not run anything, apply fixes or dependency upgrades, change
deployment or configuration, scan or attack production, expose secrets, add CI
gates or automatic reviews, require review on every PR, or bypass a security gate.

## Unframed adapter

Follow [AGENTS.md](../../../AGENTS.md) for host policy and the
[architecture contracts](../../../docs/specs/architecture.md#http-and-rendering-contract)
for the request path and server-only configuration. Nothing is deployed and there
is no authentication yet, so exposure-dependent findings are conditional. CI
produces no dependency audit or secret-scan output yet; report those as not assessed.
