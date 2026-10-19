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

# Surfaces break-glass declarations in commit messages for PR review.
#
# Usage:
#   scan-policy-flags.sh <base>..<head>   prints "<flag>\t<sha>" per hit
#   scan-policy-flags.sh --message <file> prints "<flag>[\t<detail>]" for one
#                                         message
#
# Visibility only: always exits 0 so a flagged, malformed or unreadable
# range never blocks a merge. It does not judge message correctness;
# enforcement stays in validate-commit-msg.sh.

set -uo pipefail

# Field values come in two shapes found in history: inline
# ("ATOMICITY: no") and template blocks ("ATOMICITY:" + "- Atomic: no").
# A present field that does not resolve to exactly one yes/no (untouched
# "yes | no", free text, empty block, conflicting repeats) is never
# guessed: it raises ambiguous-template instead of its own flag. Missing
# fields are left to validate-commit-msg.sh, which already rejects them.
# Checkboxes are counted only inside the TESTS section, because the
# DORMANT FEATURE and COMMENTS sections carry their own checkboxes.
# shellcheck disable=SC2016
SCAN_AWK='
function trim(s) { sub(/^[ \t]+/, "", s); sub(/[ \t\r]+$/, "", s); return s }
function block_value(s) {
  if (match(s, /:[^:]*$/)) s = substr(s, RSTART + 1)
  else sub(/^[ \t]*-[ \t]*/, "", s)
  return toupper(trim(s))
}
function resolve(v) {
  if (v ~ /^(YES|NO)[ \t]*\|/) return ""
  if (v ~ /^YES([^A-Z0-9_]|$)/) return "yes"
  if (v ~ /^NO([^A-Z0-9_]|$)/) return "no"
  return ""
}
function record(field, v,   r) {
  r = resolve(v)
  if (r == "") { unclear[field] = "unclear value"; return }
  if ((field in value) && value[field] != r) unclear[field] = "conflicting values"
  value[field] = r
}
BEGIN {
  nfields = split("ATOMICITY,DORMANT FEATURE,COMMENTS,POLICY EXCEPTION", fields, ",")
  flag_for["ATOMICITY"] = "atomicity-no";            trigger["ATOMICITY"] = "no"
  flag_for["DORMANT FEATURE"] = "dormant-feature";   trigger["DORMANT FEATURE"] = "yes"
  flag_for["COMMENTS"] = "comments";                 trigger["COMMENTS"] = "yes"
  flag_for["POLICY EXCEPTION"] = "policy-exception"; trigger["POLICY EXCEPTION"] = "yes"
}
{
  line = $0; sub(/\r$/, "", line); u = toupper(line)
  field = ""
  if (u ~ /^ATOMICITY:/) field = "ATOMICITY"
  else if (u ~ /^DORMANT FEATURE:/) field = "DORMANT FEATURE"
  else if (u ~ /^COMMENTS:/) field = "COMMENTS"
  else if (u ~ /^POLICY EXCEPTION:/) field = "POLICY EXCEPTION"
  is_tests = (u ~ /^TESTS:/)

  if (field != "" || is_tests || line ~ /^[A-Z][A-Z ]*:/) {
    if (pending != "") unclear[pending] = "empty block"
    in_tests = is_tests; pending = ""
    if (is_tests) seen_tests = 1
    if (field != "") {
      rest = trim(substr(line, index(line, ":") + 1))
      if (rest != "") record(field, toupper(rest)); else pending = field
    }
  } else if (pending != "" && trim(line) != "") {
    record(pending, block_value(line)); pending = ""
  }

  if (in_tests) {
    b = line; boxes += gsub(/\[[ xX]\]/, "", b)
    c = line; checked += gsub(/\[[xX]\]/, "", c)
  }
}
END {
  if (pending != "") unclear[pending] = "empty block"
  ambiguous = ""
  for (i = 1; i <= nfields; i++) {
    f = fields[i]
    if (f in unclear) ambiguous = ambiguous (ambiguous == "" ? "" : "; ") f ": " unclear[f]
    else if (value[f] == trigger[f]) print flag_for[f]
  }
  if (!seen_tests || (boxes > 0 && checked == 0)) print "tests-unconfirmed"
  if (ambiguous != "") print "ambiguous-template\t" ambiguous
}
'

scan_message() {
  awk "$SCAN_AWK"
}

write_summary() {
  local rows="$1"
  [[ -n "${GITHUB_STEP_SUMMARY:-}" ]] || return 0
  {
    echo "## Policy flags"
    echo
    if [[ -z "$rows" ]]; then
      echo "No break-glass declarations in this range."
    else
      echo "| Flag | Commit | Subject | Detail |"
      echo "| ---- | ------ | ------- | ------ |"
      printf '%s' "$rows"
    fi
  } >> "$GITHUB_STEP_SUMMARY"
}

if [[ "${1:-}" == "--message" ]]; then
  scan_message < "${2:?--message requires a file}"
  exit 0
fi

RANGE="${1:-}"
if [[ -z "$RANGE" ]]; then
  echo "WARN: no commit range given; nothing scanned." >&2
  exit 0
fi

if ! SHAS=$(git rev-list --no-merges --reverse "$RANGE" 2>/dev/null); then
  echo "WARN: cannot list commits in $RANGE; nothing scanned." >&2
  exit 0
fi

ROWS=""
for sha in $SHAS; do
  subject=$(git show -s --format=%s "$sha" | sed 's/|/\\|/g')
  while IFS=$'\t' read -r flag detail; do
    printf '%s\t%s\n' "$flag" "$sha"
    ROWS+="| \`$flag\` | \`${sha:0:7}\` | $subject | ${detail//|/\\|} |"$'\n'
  done < <(git show -s --format=%B "$sha" | scan_message)
done

write_summary "$ROWS"
exit 0
