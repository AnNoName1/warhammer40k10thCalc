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
VALIDATOR="$SCRIPT_DIR/validate-commit-msg.sh"
TMP_MSG="$(mktemp)"
trap 'rm -f "$TMP_MSG"' EXIT

cat > "$TMP_MSG" <<'EOF'
ci(commit-policy): sample commit for validator happy-path test

ATOMICITY: yes
TESTS: yes
DORMANT FEATURE: yes
Guards a feature flag; regression test added.
COMMENTS: no
POLICY EXCEPTION: no
EOF

if ! bash "$VALIDATOR" "$TMP_MSG"; then
  echo "FAIL: validator rejected a well-formed commit message"
  exit 1
fi

echo "PASS: validator accepts a well-formed commit message"
