# 0001 — Monorepo and rendering boundary

Status: accepted in ENG-11/ENG-13 planning and explicitly authorized for ENG-14.

## Context

Unframed is a personal engineering blog and an experiment in specification-driven,
AI-assisted development. It should eventually be reusable and self-hosted, with
external content and presentation configuration. The initial slice only verifies
two runnable applications and their connection.

## Decision

- Keep one shallow monorepo with independently buildable Go and Next.js applications.
- Go owns content/configuration loading and validation, Markdown processing, and
  preparation of article data when those features are approved.
- Next.js App Router, React, and strict TypeScript own complete reader-facing HTML,
  presentation, routing, navigation, and metadata. Next.js was chosen over Vite
  for integrated rendering and configuration-aware initial HTML, not JSON support.
- Use JSON-over-HTTP between applications. Next.js fetches Go server-side; the
  browser does not call Go directly in the skeleton.
- Use Go's standard `net/http` server with httprouter for explicit routing. Its
  literal/parameter route restrictions are an intentional guardrail for clean
  endpoints. Use standard handlers where practical, not a custom routing layer.
- Validate external TypeScript response data with Zod and infer response types
  from schemas. Go retains responsibility for backend validation.

## Consequences

Runtime rendering requires a Node-compatible frontend process alongside Go;
self-hosting is supported and Vercel is not required. Routing restrictions reduce
flexibility, and changes to content/configuration freshness will need an explicit
caching decision. The article representation, Markdown libraries, configuration
schema, publishing rules, and deployment remain undecided. Future binary/download
responses need not be JSON.

Terraform has a reserved area, but no infrastructure is provisioned. The living
architecture and decisions stay alongside code; planned requirements remain in
Linear rather than being copied into issue-specific repository documents.

## References

- [ENG-11](https://linear.app/unframed-blog/issue/ENG-11/define-architecture-and-workflow)
- [ENG-13](https://linear.app/unframed-blog/issue/ENG-13/plan-for-monorepo-implementation)
- [ENG-14](https://linear.app/unframed-blog/issue/ENG-14/implement-initial-go-and-nextjs-monorepo-skeleton)
- [Current architecture](../specs/architecture.md)
