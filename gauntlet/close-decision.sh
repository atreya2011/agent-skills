#!/usr/bin/env bash
# Decides how a review cycle ends from the items the orchestrator admitted.
#
#   close-decision.sh <locked-head> < ledger
#
# The ledger holds one admitted item per line, with fields separated by tabs:
#
#   major<TAB><check><TAB><clause>
#   minor<TAB><check><TAB><clause>
#   point<TAB><text>
#
# A point the orchestrator skipped is dropped, so it never enters the ledger.
#
# With at least one major line the cycle needs a fix. The script prints FIX and
# then the two lists the fix brief takes, "Accepted red checks:" and "Admitted
# points:", with every admitted item in ledger order. Otherwise the run closes.
# The script prints DIMINISHING RETURNS and, when the ledger is not empty, the
# "## Review notes" list for the PR body: each admitted item with the locked
# head it was raised on.
#
# A line whose first field is not major, minor or point prints CLOSE BLOCKED and
# exits 1 with nothing on standard output.
set -euo pipefail

locked_head=${1:?usage: close-decision.sh LOCKED_HEAD < ledger}
checks=()
points=()
major=0
while IFS=$'\t' read -r class first second || [[ -n $class ]]; do
  case $class in
    major) major=1; checks+=("Check: $first — Clause: $second") ;;
    minor) checks+=("Check: $first — Clause: $second") ;;
    point) points+=("Point: $first") ;;
    *)
      echo "CLOSE BLOCKED unknown ledger class: $class" >&2
      exit 1
      ;;
  esac
done

if (( major )); then
  echo FIX
  echo 'Accepted red checks:'
  printf -- '- %s\n' "${checks[@]}"
  if (( ${#points[@]} )); then
    echo 'Admitted points:'
    printf -- '- %s\n' "${points[@]}"
  else
    echo 'Admitted points: none'
  fi
  exit 0
fi

echo 'DIMINISHING RETURNS'
items=("${checks[@]}" "${points[@]}")
if (( ${#items[@]} )); then
  echo '## Review notes'
  for item in "${items[@]}"; do
    printf -- '- %s — HEAD: %s\n' "$item" "$locked_head"
  done
fi
