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

# Quote variable to handle Windows paths with spaces
COMMIT_MSG_FILE="$1"
COMMIT_MSG=$(cat "$COMMIT_MSG_FILE")

# 1. Header Validation: <type>(<scope>): <summary>
TYPE_REGEX="^(feat|fix|refactor|test|docs|build|ci|chore)"
if ! echo "$COMMIT_MSG" | grep -qE "$TYPE_REGEX(\(.*\))?: .*"; then
  echo "ERROR: Invalid commit type format."
  exit 1
fi

# 2. Mandatory Fields (Literal match)
for field in "ATOMICITY:" "TESTS:"; do
  if ! echo "$COMMIT_MSG" | grep -q "$field"; then
    echo "ERROR: Missing mandatory field: $field"
    exit 1
  fi
done

# 3. Justification Logic (Check for non-whitespace content on next line)
if echo "$COMMIT_MSG" | grep -q "POLICY EXCEPTION: yes"; then
  # Extract line after the match, trim it, check if empty
  JUSTIFICATION=$(grep -A 1 "POLICY EXCEPTION: yes" "$COMMIT_MSG_FILE" | tail -n 1 | xargs)
  if [[ -z "$JUSTIFICATION" ]]; then
    echo "ERROR: POLICY EXCEPTION: yes requires a justification on the following line."
    exit 1
  fi
fi