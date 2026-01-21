#!/usr/bin/env bash
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