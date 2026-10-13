# Infrastructure boundary

This directory reserves space for future Terraform infrastructure. ENG-14 does not
define Terraform resources, select AWS services, configure providers, provision
infrastructure, or require an AWS account. No Terraform command is needed to run
or verify the skeleton.

Before implementation, a separately approved infrastructure specification should
address hosting options, costs, regions, state storage, and access boundaries.
AWS is an intended hosting option, not a requirement for a self-hosted installation.

Initial guardrails for that future work:

- Start with one deployment root; introduce modules only for demonstrated reuse.
- Isolate state per environment. Use remote state and locking once deploying.
- Never commit state, plans, credentials, or real variable values. State can contain
  sensitive data. Commit provider lockfiles when providers are introduced.
- Use least-privilege roles. Prefer short-lived CI credentials, such as OIDC,
  rather than stored AWS access keys.
- Keep deployment separate from application verification. No deployment jobs or
  cloud credentials belong in this skeleton's CI.
