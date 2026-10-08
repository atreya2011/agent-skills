#!/usr/bin/env bash
# Tests for gauntlet/close-decision.sh. Each case feeds one cycle's ledger of
# admitted items to the script and asserts its exit status and its output, which
# is the whole decision: a fix brief's lists, or the closing review notes.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"
script="$PWD/../../gauntlet/close-decision.sh"
locked_head=abc1234

failed=0
fail() {
  printf 'FAIL: %s: %s\n' "$1" "$2"
  failed=1
}

tab=$'\t'
# Joins its arguments into lines.
lines() { printf '%s\n' "$@"; }

# Each case has a ledger (what the orchestrator admitted) and the output the
# decision must print.
declare -A ledger want

ledger[fix]=$(lines \
  "major${tab}check-a${tab}clause a" \
  "minor${tab}check-b${tab}clause b" \
  "point${tab}rename x to y because z" \
  "point${tab}drop the unused flag")
want[fix]=$(lines \
  'FIX' \
  'Accepted red checks:' \
  '- Check: check-a — Clause: clause a' \
  '- Check: check-b — Clause: clause b' \
  'Admitted points:' \
  '- Point: rename x to y because z' \
  '- Point: drop the unused flag')

ledger[fix-major-last]=$(lines \
  "point${tab}drop the unused flag" \
  "minor${tab}check-b${tab}clause b" \
  "major${tab}check-a${tab}clause a")
want[fix-major-last]=$(lines \
  'FIX' \
  'Accepted red checks:' \
  '- Check: check-b — Clause: clause b' \
  '- Check: check-a — Clause: clause a' \
  'Admitted points:' \
  '- Point: drop the unused flag')

ledger[fix-no-points]="major${tab}check-a${tab}clause a"
want[fix-no-points]=$(lines \
  'FIX' \
  'Accepted red checks:' \
  '- Check: check-a — Clause: clause a' \
  'Admitted points: none')

ledger[close-minors-and-points]=$(lines \
  "minor${tab}check-b${tab}clause b" \
  "minor${tab}check-c${tab}clause c" \
  "point${tab}rename x to y because z" \
  "point${tab}drop the unused flag")
want[close-minors-and-points]=$(lines \
  'DIMINISHING RETURNS' \
  '## Review notes' \
  "- Check: check-b — Clause: clause b — HEAD: $locked_head" \
  "- Check: check-c — Clause: clause c — HEAD: $locked_head" \
  "- Point: rename x to y because z — HEAD: $locked_head" \
  "- Point: drop the unused flag — HEAD: $locked_head")

ledger[close-points-only]="point${tab}rename x to y because z"
want[close-points-only]=$(lines \
  'DIMINISHING RETURNS' \
  '## Review notes' \
  "- Point: rename x to y because z — HEAD: $locked_head")

ledger[close-empty]=''
want[close-empty]='DIMINISHING RETURNS'

ledger[refuse-unknown-class]="mjor${tab}check-a${tab}clause a"
want[refuse-unknown-class]=''

# name | case | expected exit
cases=(
  'one major bug yields a fix brief listing every admitted item|fix|0'
  'a major bug listed last still yields a fix brief|fix-major-last|0'
  'a major bug alone yields a fix brief with no points|fix-no-points|0'
  'minors and points only close with exactly the admitted items as review notes|close-minors-and-points|0'
  'points only close with exactly the admitted point as a review note|close-points-only|0'
  'an empty ledger closes with no review notes|close-empty|0'
  'an unknown ledger class is refused with nothing printed|refuse-unknown-class|1'
)
for row in "${cases[@]}"; do
  IFS='|' read -r name key want_status <<<"$row"
  status=0
  out=$(printf '%s' "${ledger[$key]}" | "$script" "$locked_head" 2>/dev/null) || status=$?
  if [[ "$status" != "$want_status" ]]; then
    fail "$name" "exit $status, want $want_status; output: $out"
    continue
  fi
  [[ "$out" == "${want[$key]}" ]] || { fail "$name" "output: $out"; continue; }
  printf 'ok: %s\n' "$name"
done

(( failed )) && exit 1
exit 0
