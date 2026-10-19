#!/usr/bin/env bash
# Copyright (c) 2026 Olbutov Aleksandr
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

# Correctness tests for scan-policy-flags.sh. Run blocking in remote-guard,
# unlike the scanner itself, which must never fail the policy-flags check.
# Usage: bash scripts/scan-policy-flags_test.sh

set -uo pipefail

SCANNER="$(cd "$(dirname "$0")" && pwd)/scan-policy-flags.sh"
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
FAILURES=0

report() {
  local name="$1" want="$2" got="$3"
  if [[ "$got" == "$want" ]]; then
    echo "ok   $name"
  else
    echo "FAIL $name: want [$want], got [$got]"
    FAILURES=$((FAILURES + 1))
  fi
}

expect_flags() {
  cat > "$WORK/msg"
  report "$1" "$2" "$(bash "$SCANNER" --message "$WORK/msg" | cut -f1 | paste -sd' ' -)"
}

expect_detail() {
  cat > "$WORK/msg"
  report "$1" "$2" "$(bash "$SCANNER" --message "$WORK/msg" | grep '^ambiguous-template' | cut -f2)"
}

expect_flags "clean inline message" "" <<'EOF'
feat(x): add thing

ATOMICITY: yes
TESTS: added TestThing.
DORMANT FEATURE: no
COMMENTS: no
POLICY EXCEPTION: no
EOF

expect_flags "inline atomicity no, any case and spacing" "atomicity-no" <<'EOF'
feat(x): add thing

atomicity:no
TESTS: added TestThing.
Dormant Feature: NO
COMMENTS: no
POLICY EXCEPTION: no
EOF

expect_flags "template block form" "atomicity-no" <<'EOF'
feat(x): add thing

ATOMICITY:
- [cite_start]Atomic: no
- Splitting would break the build.

TESTS:
- Tests added in this commit:
  - [x] happy-path

DORMANT FEATURE:
- Dormant code introduced or modified: no

COMMENTS:
- What-comments added, retained, or eliminated in this commit: no

POLICY EXCEPTION:
- Required: no
EOF

expect_flags "prose mention is not a declaration" "" <<'EOF'
docs(x): explain atomicity

WHY:
- Explains what ATOMICITY: no means.
ATOMICITY: yes
TESTS: docs only.
DORMANT FEATURE: no
COMMENTS: no
POLICY EXCEPTION: no
EOF

expect_flags "inline and block yes declarations" "dormant-feature comments policy-exception" <<'EOF'
feat(x): add thing

ATOMICITY: yes
TESTS: added TestThing.
DORMANT FEATURE: Yes
COMMENTS:
- What-comments added, retained, or eliminated in this commit: yes
POLICY EXCEPTION: yes
- Rule 3 broken because the helper is generated.
EOF

expect_flags "all TESTS checkboxes unchecked" "tests-unconfirmed" <<'EOF'
ci(x): fix step

ATOMICITY: yes
TESTS:
- Tests added in this commit:
  - [ ] happy-path
  - [ ] error cases
DORMANT FEATURE: no
COMMENTS: no
POLICY EXCEPTION: no
EOF

expect_flags "checked box outside TESTS does not confirm tests" "dormant-feature tests-unconfirmed" <<'EOF'
feat(x): add thing

ATOMICITY: yes
TESTS:
  - [ ] happy-path
DORMANT FEATURE: yes
  - [X] regression guard test
COMMENTS: no
POLICY EXCEPTION: no
EOF

expect_flags "one checked TESTS box confirms tests" "" <<'EOF'
feat(x): add thing

ATOMICITY: yes
TESTS:
  - [ ] happy-path
  - [X] error cases
DORMANT FEATURE: no
COMMENTS: no
POLICY EXCEPTION: no
EOF

expect_flags "missing TESTS section" "tests-unconfirmed" <<'EOF'
feat(x): add thing

ATOMICITY: yes
DORMANT FEATURE: no
COMMENTS: no
POLICY EXCEPTION: no
EOF

expect_flags "untouched template value is ambiguous, not a declaration" "ambiguous-template" <<'EOF'
feat(x): add thing

ATOMICITY: yes
TESTS: added TestThing.
DORMANT FEATURE: yes | no
COMMENTS: no
POLICY EXCEPTION: no
EOF

expect_detail "ambiguity detail names each field and reason" \
  "ATOMICITY: unclear value; DORMANT FEATURE: empty block; COMMENTS: conflicting values" <<'EOF'
feat(x): add thing

ATOMICITY: mostly
TESTS: added TestThing.
DORMANT FEATURE:

COMMENTS: no
COMMENTS: yes
EOF

expect_flags "unclear field suppresses its own flag only" "policy-exception ambiguous-template" <<'EOF'
feat(x): add thing

ATOMICITY:
- If no, explain why an atomic split is not feasible.
TESTS: added TestThing.
DORMANT FEATURE: no
COMMENTS: no
POLICY EXCEPTION: yes
- Generated code.
EOF

expect_flags "missing fields are left to the validator" "tests-unconfirmed" <<'EOF'
Added hit prop calc, test
EOF

FIELDS=$'ATOMICITY: yes\nTESTS: none.\nDORMANT FEATURE: no\nCOMMENTS: no\nPOLICY EXCEPTION: no'
git init -q "$WORK/repo"
(
  cd "$WORK/repo"
  git config user.email t@example.com && git config user.name t
  git commit -q --allow-empty -m "chore: base" -m "$FIELDS"
  git checkout -q -b side && git commit -q --allow-empty -m "chore: side" -m "$FIELDS"
  git checkout -q - && git checkout -q -b pr
  git commit -q --allow-empty -m "feat: a" -m "${FIELDS/ATOMICITY: yes/ATOMICITY: no}"
  git merge -q --no-ff --no-edit side
  git commit -q --allow-empty -m "feat: b | c" -m "${FIELDS/COMMENTS: no/COMMENTS: yes | no}"
)
BASE=$(git -C "$WORK/repo" rev-parse side~1)

report "range scan skips merges, keeps commit order" \
  "atomicity-no ambiguous-template" \
  "$(cd "$WORK/repo" && bash "$SCANNER" "$BASE..pr" | cut -f1 | paste -sd' ' -)"

report "range scan prints full sha per flag" "40" \
  "$(cd "$WORK/repo" && bash "$SCANNER" "$BASE..pr" | head -1 | cut -f2 | tr -d '\n' | wc -c | tr -d ' ')"

export GITHUB_STEP_SUMMARY="$WORK/summary.md"
(cd "$WORK/repo" && bash "$SCANNER" "$BASE..pr" > /dev/null)
report "step summary row escapes pipes in subject and detail" "1" \
  "$(grep -c '| `ambiguous-template` | `[0-9a-f]\{7\}` | feat: b \\| c | COMMENTS: unclear value |' "$GITHUB_STEP_SUMMARY")"
unset GITHUB_STEP_SUMMARY

(cd "$WORK/repo" && bash "$SCANNER" "no-such-ref..pr" > /dev/null 2>&1)
report "unreadable range still exits 0" "0" "$?"

[[ "$FAILURES" -eq 0 ]] || { echo "$FAILURES test(s) failed"; exit 1; }
echo "all tests passed"
