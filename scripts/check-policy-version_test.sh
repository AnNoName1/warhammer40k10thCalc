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

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/check-policy-version.sh"
FIXTURE_ROOT="$(mktemp -d)"
trap 'rm -rf "$FIXTURE_ROOT"' EXIT

failures=0

write_fixture() {
  local policy_md="$1" gitmessage="$2" validator="$3"
  rm -rf "$FIXTURE_ROOT"/*
  mkdir -p "$FIXTURE_ROOT/scripts"
  printf '# Commit Policy\r\n\r\n%s\r\n' "$policy_md" > "$FIXTURE_ROOT/commit-policy.md"
  printf '%s\r\n<type>(<scope>): <summary>\r\n' "$gitmessage" > "$FIXTURE_ROOT/.gitmessage"
  printf '#!/usr/bin/env bash\n%s\nset -euo pipefail\n' "$validator" > "$FIXTURE_ROOT/scripts/validate-commit-msg.sh"
}

expect_pass() {
  local name="$1" output
  if output=$(bash "$CHECKER" "$FIXTURE_ROOT" 2>&1); then
    echo "PASS: $name"
  else
    echo "FAIL: $name: expected success, got:"
    echo "$output"
    failures=$((failures + 1))
  fi
}

expect_fail_mentioning() {
  local name="$1" expected="$2" output
  if output=$(bash "$CHECKER" "$FIXTURE_ROOT" 2>&1); then
    echo "FAIL: $name: expected failure, checker passed"
    failures=$((failures + 1))
  elif ! grep -qF -- "$expected" <<< "$output"; then
    echo "FAIL: $name: output does not mention '$expected':"
    echo "$output"
    failures=$((failures + 1))
  else
    echo "PASS: $name"
  fi
}

write_fixture "POLICY_VERSION: 3" "# POLICY_VERSION: 3" "# POLICY_VERSION: 3"
expect_pass "all three versions equal"

write_fixture "POLICY_VERSION: 4" "# POLICY_VERSION: 3" "# POLICY_VERSION: 3"
expect_fail_mentioning "commit-policy.md out of sync" "Mismatched file: commit-policy.md (has 4, others have 3)"

write_fixture "POLICY_VERSION: 3" "# POLICY_VERSION: 2" "# POLICY_VERSION: 3"
expect_fail_mentioning ".gitmessage out of sync" "Mismatched file: .gitmessage (has 2, others have 3)"

write_fixture "POLICY_VERSION: 3" "# POLICY_VERSION: 3" "# POLICY_VERSION: 4"
expect_fail_mentioning "validator out of sync" "Mismatched file: scripts/validate-commit-msg.sh (has 4, others have 3)"

write_fixture "POLICY_VERSION: 1" "# POLICY_VERSION: 2" "# POLICY_VERSION: 3"
expect_fail_mentioning "all three differ" "All files disagree"

write_fixture "no marker here" "# POLICY_VERSION: 3" "# POLICY_VERSION: 3"
expect_fail_mentioning "missing marker" "commit-policy.md: no 'POLICY_VERSION: <N>' marker found"

write_fixture "POLICY_VERSION: 3" $'# POLICY_VERSION: 3\r\n# POLICY_VERSION: 4' "# POLICY_VERSION: 3"
expect_fail_mentioning "duplicate marker" ".gitmessage: 2 POLICY_VERSION markers found"

write_fixture "POLICY_VERSION: 3" "# POLICY_VERSION: 3" "# POLICY_VERSION: 3"
rm "$FIXTURE_ROOT/.gitmessage"
expect_fail_mentioning "missing file" ".gitmessage: file not found"

if [[ "$failures" -ne 0 ]]; then
  echo "$failures check-policy-version test(s) failed"
  exit 1
fi
