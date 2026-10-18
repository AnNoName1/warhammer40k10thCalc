# Commit Policy

POLICY_VERSION: 1

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

## Dormant Feature Rule

Dormant code is code that implements a real, specified feature or rule but
is intentionally inactive — behind a flag, an unreachable branch, or a
default-off condition.

Dormant code MUST be covered by one of:
- a full test suite verifying the dormant behavior itself is correct, or
- a regression guard test that fails if the feature is activated without
  that verification work having been done.

The commit that introduces or modifies dormant code MUST declare
`DORMANT FEATURE: yes` and explain:
- what feature or rule the code implements,
- which of the two coverage options above was used.

## Comment Rule

All comments MUST be written in English.

Comments that explain *what* code does ("what-comments") are prohibited by
default. A what-comment MAY be kept only if:
- it explains complex math that the code's structure cannot convey on its
  own, or
- it explains an external rule or domain constraint (e.g. wargame rules,
  physics, regulation) that the code's structure cannot convey on its own.

A what-comment that exists only because the surrounding code is unclear
(poor naming, unstructured logic) is not a valid reason to keep it. The
code MUST instead be reworked until the comment is unnecessary — renaming,
extracting a function, naming a constant, etc. — and the comment removed.

Comments that explain *why* code exists or behaves a certain way
("why-comments": rationale, invariants, warnings, workarounds) are not
restricted by this rule.

The commit MUST declare `COMMENTS: yes` if it adds, retains, or eliminates
any what-comment, and explain:
- which of the two valid reasons above applies, if one was kept, or
- if the comment was compensating for unclear code, what method was used
  to make it redundant, confirming it was removed.

## Commit Message Format

All commits MUST declare:
- ATOMICITY
- TESTS
- DORMANT FEATURE
- COMMENTS

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

### Policy Version Sync

The commit format is defined in three places that MUST stay in sync:
- `commit-policy.md` (this document),
- `.gitmessage` (commit template),
- `scripts/validate-commit-msg.sh` (commit-msg validator).

Each file carries a `POLICY_VERSION: <N>` marker. Any change to the commit
format or its fields (adding, removing, renaming a field, changing allowed
values or types, changing validation rules) MUST increment the version in
all three files in the same commit.

`scripts/check-policy-version.sh` enforces that all three markers hold the
same number. The check is stateless: it compares the current contents of
the three files only and does not inspect history or diffs, so it cannot
tell whether a given change required a bump — that remains the author's
responsibility. It runs as a blocking step in the `pre-commit` hook and in
CI (`Remote-Push Guard` workflow).
