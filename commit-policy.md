# Commit Policy

## Core Guarantees (Non-Negotiable)

1. Every commit MUST build.
2. Existing functionality MUST:
   - continue working correctly, or
   - be explicitly null only if introduced on the current feature branch.
3. Every feature or helper introduced in a commit MUST include
   at least one happy-path test in the same commit.
4. By pull request:
   - all non-trivial functionality MUST be covered by tests,
   - including error cases and invariants where applicable.

## Atomicity Rule

Commits SHOULD be atomic.

A commit is atomic if:
- it has a single reason to exist,
- it can be reverted without leaving the codebase in an inconsistent state.

Non-atomic commits are allowed ONLY when atomic splitting would:
- break the build,
- require excessive duplication,
- or require unsafe intermediate states.

Non-atomic commits MUST:
- declare `ATOMICITY: no`,
- include a justification.

## Transitional Code Rule

If atomicity is not achievable, transitional code MUST be used.

Transitional code MUST:
- be clearly marked (e.g. `// TRANSITIONAL:`),
- be temporary,
- be removed in a follow-up commit.

The commit introducing transitional code MUST explain:
- why it exists,
- how it will be removed.

## Commit Message Format

All commits MUST declare:
- ATOMICITY
- TESTS

If any rule is intentionally broken, the commit MUST declare:
- `POLICY EXCEPTION: yes`
- with justification.

## Commit Types

Allowed types:
- feat       – new functionality
- fix        – bug fix
- refactor   – behavior-preserving change
- test       – tests only
- docs       – documentation
- build      – build / dependencies
- ci         – CI configuration
- chore      – maintenance

New types MAY be introduced, but MUST be added to this document.

## Policy Evolution

This policy may evolve.

Any commit that changes:
- commit rules,
- commit types,
- enforcement mechanisms

MUST update this document and explain the change.
