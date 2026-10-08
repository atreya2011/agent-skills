#!/usr/bin/env bash
# Tests for gauntlet/close-decision.sh. Each case feeds one cycle's admission
# list to the script and asserts its exit status, its output, which is the whole
# decision (a fix brief's lists, or the closing review notes), and its error
# line. The check files are synthetic and live in a temporary directory, so the
# output must never contain their path.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"
script="$PWD/../../gauntlet/close-decision.sh"
locked_head=abc1234
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

failed=0
fail() {
  printf 'FAIL: %s: %s\n' "$1" "$2"
  failed=1
}

tab=$'\t'
# Joins its arguments into lines.
lines() { printf '%s\n' "$@"; }

printf 'run-thing --flag\n' >"$tmp/check-a"
printf 'cd service\nrun-thing --flag\n\nread-thing | grep -q ok\n' >"$tmp/check-b"
printf 'run-other-thing\n' >"$tmp/check-c"

# The rendered form of each check, as the output must show it.
item_a=$(lines '- Check:' '  ```' '  run-thing --flag' '  ```' '  Clause: clause a')
item_b=$(lines '- Check:' '  ```' '  cd service' '  run-thing --flag' '' '  read-thing | grep -q ok' '  ```' '  Clause: clause b')
item_c=$(lines '- Check:' '  ```' '  run-other-thing' '  ```' '  Clause: clause c')

# Each case has an admission list (what the orchestrator ruled), the output the
# decision must print and the error line it must print, empty for none.
declare -A list want err

list[fix]=$(lines \
  "major${tab}$tmp/check-a${tab}clause a" \
  "minor${tab}$tmp/check-b${tab}clause b" \
  "point${tab}rename x to y because z" \
  "point${tab}drop the unused flag")
want[fix]=$(lines "FIX" "$item_a" "$item_b" \
  '- Point: rename x to y because z' \
  '- Point: drop the unused flag')

list[fix-major-last]=$(lines \
  "point${tab}drop the unused flag" \
  "minor${tab}$tmp/check-b${tab}clause b" \
  "major${tab}$tmp/check-a${tab}clause a")
want[fix-major-last]=$(lines "FIX" "$item_b" "$item_a" '- Point: drop the unused flag')

list[fix-no-points]="major${tab}$tmp/check-a${tab}clause a"
want[fix-no-points]=$(lines "FIX" "$item_a" 'none')

list[fix-skipped-point]=$(lines \
  "major${tab}$tmp/check-a${tab}clause a" \
  "skip${tab}rewrite everything")
want[fix-skipped-point]=$(lines "FIX" "$item_a" 'none')

list[close-minors-and-points]=$(lines \
  "minor${tab}$tmp/check-b${tab}clause b" \
  "minor${tab}$tmp/check-c${tab}clause c" \
  "point${tab}rename x to y because z" \
  "point${tab}drop the unused flag")
want[close-minors-and-points]=$(lines \
  'DIMINISHING RETURNS' \
  '## Review notes' \
  "$item_b" "  HEAD: $locked_head" \
  "$item_c" "  HEAD: $locked_head" \
  '- Point: rename x to y because z' "  HEAD: $locked_head" \
  '- Point: drop the unused flag' "  HEAD: $locked_head")

list[close-skipped-point]=$(lines \
  "point${tab}rename x to y because z" \
  "skip${tab}rewrite everything")
want[close-skipped-point]=$(lines \
  'DIMINISHING RETURNS' \
  '## Review notes' \
  '- Point: rename x to y because z' "  HEAD: $locked_head")

list[close-blank-lines]=$(lines \
  '' \
  "minor${tab}$tmp/check-a${tab}clause a" \
  '' \
  '   ' \
  "point${tab}rename x to y because z" \
  '')
want[close-blank-lines]=$(lines \
  'DIMINISHING RETURNS' \
  '## Review notes' \
  "$item_a" "  HEAD: $locked_head" \
  '- Point: rename x to y because z' "  HEAD: $locked_head")

list[close-empty]=''
want[close-empty]='DIMINISHING RETURNS'

list[refuse-unknown-class]="mjor${tab}$tmp/check-a${tab}clause a"
want[refuse-unknown-class]=''
err[refuse-unknown-class]='CLOSE BLOCKED unknown class mjor'

list[refuse-unreadable-check]="major${tab}$tmp/no-such-check${tab}clause a"
want[refuse-unreadable-check]=''
err[refuse-unreadable-check]="CLOSE BLOCKED unreadable check file $tmp/no-such-check"

# name | case | expected exit
cases=(
  'one major bug yields a fix brief listing every admitted item|fix|0'
  'a major bug listed last still yields a fix brief|fix-major-last|0'
  'a major bug alone yields a fix brief with no points|fix-no-points|0'
  'a skipped point is dropped from a fix brief|fix-skipped-point|0'
  'minors and points only close with exactly the admitted items as review notes|close-minors-and-points|0'
  'a skipped point is absent from the review notes|close-skipped-point|0'
  'blank lines in the admission list are ignored|close-blank-lines|0'
  'an empty admission list closes with no review notes|close-empty|0'
  'an unknown class is refused with nothing printed|refuse-unknown-class|1'
  'an unreadable check file is refused with nothing printed|refuse-unreadable-check|1'
)
for row in "${cases[@]}"; do
  IFS='|' read -r name key want_status <<<"$row"
  status=0
  out=$(printf '%s' "${list[$key]}" | "$script" "$locked_head" 2>"$tmp/err") || status=$?
  error=$(<"$tmp/err")
  if [[ "$status" != "$want_status" ]]; then
    fail "$name" "exit $status, want $want_status; output: $out"
    continue
  fi
  [[ "$out" == "${want[$key]}" ]] || { fail "$name" "output: $out"; continue; }
  [[ "$error" == "${err[$key]-}" ]] || { fail "$name" "error line: $error"; continue; }
  [[ "$out" != *"$tmp"* ]] || { fail "$name" "output holds a check file path"; continue; }
  printf 'ok: %s\n' "$name"
done

(( failed )) && exit 1
exit 0
