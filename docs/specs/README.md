# Living architecture specification

The [current architecture](architecture.md) is the single running specification of
the implemented system: structure, responsibilities, contracts, tooling, and known
boundaries. Update it as development changes those facts. Do not create
issue-prefixed specifications or copy planning documents from Linear into Git.

Linear owns planned requirements, approval, and acceptance criteria. GitHub owns
source, the living architecture, decision rationale, and verification history.

1. Discuss requirements and challenge assumptions in the planning issue.
2. Agree on requirements, boundaries, and acceptance criteria in Linear.
3. Obtain Gabriel's explicit implementation authorization before changing code,
   creating a branch, committing, or opening a PR.
4. Implement on the documented feature branch, update the living architecture,
   verify the criteria, and open a linked PR for human review.
5. Gabriel decides whether to merge. Agents never merge.

Material changes to architecture, security boundaries, infrastructure, or approved
scope return to discussion and human approval. Agent instructions and environment
configuration are separate workflow work.

## Current system

- [Current architecture](architecture.md)
