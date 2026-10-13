# Implementation specifications

Specifications record approved scope, contracts, and behavioral acceptance
criteria. Use an issue-prefixed filename and link the specification from Linear.
Keep one authoritative requirements document; issue descriptions should link it
rather than maintain a competing copy. PRs provide implementation and verification
evidence, not a second specification.

1. Discuss requirements and challenge assumptions in the planning issue.
2. Agree on a specification, including boundaries and acceptance criteria.
3. Obtain Gabriel's explicit implementation authorization before changing code,
   creating a branch, committing, or opening a PR.
4. Implement on the documented feature branch, verify the criteria, and open a
   linked PR for human review.
5. Gabriel decides whether to merge. Agents never merge.

Material changes to architecture, security boundaries, infrastructure, or approved
scope return to discussion and human approval. Agent instructions and a Devin
environment blueprint remain separate follow-up work after commands are verified.

## Specifications

- [ENG-14: initial monorepo skeleton](eng-14-monorepo-skeleton.md)
