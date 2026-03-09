# CI Feature Placement & Job Topology

## CI Levels

```
pre-commit → pre-push → commit → remote-push → MR → nightly(not implemented yet)
```

**Definitions**

* **Δ (delta)**: affected files / packages relative to a known base commit
* **Σ (total)**: whole repository
* **Remote-push**: server-side validation on every push (do not trust local hooks)
* **MR**: merge-readiness + policy enforcement
* **Nightly**: non-blocking, non-deterministic, capacity/system checks

---

## Feature Placement Table (Authoritative)

| Feature                            | Δ / Σ | Level       | Notes / Invariant Enforced                |
| ---------------------------------- | ----- | ----------- | ----------------------------------------- |
| gofmt                              | Δ     | pre-commit  | Syntactic invariant, zero false positives |
| go vet                             | Δ     | pre-commit  | Cheap semantic validation                 |
| go test -short                     | Δ     | pre-commit  | Fast logic guard                          |
| add-license (check)                | Δ     | pre-commit  | Mechanical invariant                      |
| commit message policy              | Σ     | pre-commit      | Metadata-only rule                    |
| add-license (enforce)              | Σ     | pre-push    | Repo-wide consistency                     |
| swag-fmt                           | Δ     | pre-push    | Generated artifact hygiene                |
| swag-init                          | Σ     | pre-push    | Spec drift detection                      |
| **commit message policy (server)** | Σ     | remote-push | Trust reset                               |
| **gofmt (server)**                 | Δ     | remote-push | Duplicate critical invariant              |
| **go vet (server)**                | Δ     | remote-push | Duplicate critical invariant              |
| **build**                          | Σ     | remote-push | Graph completeness                        |
| **unit tests (full)**              | Σ     | remote-push | Hard correctness gate                     |
| dependency check                   | Σ     | MR          | Module graph risk                         |
| dependency license compliance      | Σ     | MR          | Legal / supply-chain risk                 |
| swagger generation                 | Σ     | MR          | SSOT enforcement                          |
| swagger lint                       | Σ     | MR          | API semantic validity                     |
| **oasdiff (breaking check)**       | Σ     | MR          | API compatibility *policy*                |
| golangci-lint                      | Δ     | MR          | Expensive static analysis                 |
| integration tests                  | Σ     | MR          | Endpoint correctness                      |
| code coverage threshold            | Σ     | MR          | Structural quality gate                   |
| e2e tests                          | Σ     | nightly     | Flaky / infra-heavy                       |
| load / stress tests                | Σ     | nightly     | Capacity validation                       |

---

## Job Grouping

### Remote-push CI (fast, blocking)

#### Job: `remote-guard`

**Target:** ≤ 60s
**Purpose:** fail before build/test

Steps (strict order):

1. checkout (`fetch-depth: 0`)
2. commit message check
3. gofmt (Δ)
4. go vet (Δ)
5. add-license (check)

---

#### Job: `remote-build-test`

**Needs:** `remote-guard`
**Target:** ≤ 60s

Steps:

1. setup Go + cache
2. go build ./...
3. go test ./... (unit tests)

---

## Merge Request CI (amortized, correctness + policy)

#### Job: `analysis`

**Target:** ~60s

Steps:

1. dependency vulnerability check
2. dependency license scan

---

#### Job: `linting-and-spec`

**Target:** ~60–75s

Steps:

1. swagger generation
2. swagger lint
3. golangci-lint (Δ)

---

#### Job: `api-diff`

**Needs:** swagger generation
**Target:** ~10–20s

Steps:

1. generate OpenAPI (base branch)
2. generate OpenAPI (PR branch)
3. run `oasdiff`
4. label PR / fail if unapproved

---

#### Job: `build-and-test`

**Target:** ~60–90s

Steps:

1. go build
2. go test ./... (unit + integration)
3. coverage threshold check

---

## Nightly CI (non-blocking) - not implemented yet

#### Job: `system-tests`

Steps:

* e2e tests
* load / stress tests

Rules:

* retries allowed
* no merge gating
* trend analysis only

---

## Dependency Graph (Simplified)

```
remote-guard
   ↓
remote-build-test
   ↓
MR:
   ├─ analysis
   ├─ linting-and-spec ─┐
   ├─ api-diff          ├─ merge gate
   └─ build-and-test   ─┘

nightly (detached)
```

---

## Global Invariants Enforced

* No malformed commit reaches shared history
* No build- or unit-test–breaking commit reaches main
* No silent API breaking change merges without explicit approval
* Expensive checks are amortized at MR or nightly level
* GitHub Actions billing minimized via job coalescing
