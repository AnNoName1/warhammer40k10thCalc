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

# Stateless by design (see commit-policy.md, "Policy Version Sync"): compares
# only the current contents of the policy files, never history or diffs.

set -euo pipefail

REPO_ROOT="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"

POLICY_FILES=(
  "commit-policy.md"
  ".gitmessage"
  "scripts/validate-commit-msg.sh"
)

# Optional leading "#" lets the marker live inside shell and template comments.
MARKER_REGEX='^[#[:space:]]*POLICY_VERSION:[[:space:]]*([0-9]+)[[:space:]]*$'

read_policy_version() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    echo "ERROR: $path: file not found." >&2
    return 1
  fi

  local versions
  versions=$(tr -d '\r' < "$path" | sed -nE "s/$MARKER_REGEX/\1/p")
  local count
  count=$(printf '%s' "$versions" | grep -c . || true)

  if [[ "$count" -eq 0 ]]; then
    echo "ERROR: $path: no 'POLICY_VERSION: <N>' marker found." >&2
    return 1
  fi
  if [[ "$count" -gt 1 ]]; then
    echo "ERROR: $path: $count POLICY_VERSION markers found, expected exactly one." >&2
    return 1
  fi
  echo "$versions"
}

declare -A version_of
failed=0
for file in "${POLICY_FILES[@]}"; do
  if ! version_of[$file]=$(read_policy_version "$REPO_ROOT/$file"); then
    failed=1
  fi
done
if [[ "$failed" -ne 0 ]]; then
  exit 1
fi

majority_version=$(printf '%s\n' "${version_of[@]}" | sort | uniq -c | sort -rn | awk 'NR==1 { print $2 }')
majority_count=$(printf '%s\n' "${version_of[@]}" | grep -cx "$majority_version")

if [[ "$majority_count" -eq "${#POLICY_FILES[@]}" ]]; then
  echo "OK: POLICY_VERSION $majority_version in all policy files."
  exit 0
fi

echo "ERROR: POLICY_VERSION mismatch between commit policy files:"
for file in "${POLICY_FILES[@]}"; do
  echo "  $file: ${version_of[$file]}"
done
# With no strict majority there is no reference version to blame against.
if [[ "$majority_count" -gt 1 ]]; then
  for file in "${POLICY_FILES[@]}"; do
    if [[ "${version_of[$file]}" != "$majority_version" ]]; then
      echo "Mismatched file: $file (has ${version_of[$file]}, others have $majority_version)."
    fi
  done
else
  echo "All files disagree; no reference version."
fi
echo "Any change to the commit format MUST bump POLICY_VERSION in all three files (commit-policy.md, Policy Version Sync)."
exit 1
