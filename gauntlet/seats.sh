#!/usr/bin/env bash
#
#
#
set -euo pipefail

usage='usage: seats.sh [--table <name>] <reviewers: a count, or reviewer-<name> seats separated by commas>'
bad_usage() {
  echo "$usage" >&2
  exit 2
}

table=gauntlet
if [[ ${1-} == --table ]]; then
  (( $# >= 2 )) || bad_usage
  table=gauntlet-$2
  shift 2
fi
(( $# == 1 )) || bad_usage
choice=$1

file=$HOME/.agents/local/$table.md
if [[ ! -f $file ]]; then
  echo "SEAT MISSING orchestrator (no $file)" >&2
  exit 1
fi

trim() {
  local cell=${1//\`/}
  cell=${cell#"${cell%%[![:space:]]*}"}
  printf '%s' "${cell%"${cell##*[![:space:]]}"}"
}

declare -A row
pool=()
while IFS='|' read -r _ seat kind args clear _; do
  seat=$(trim "$seat")
  row[$seat]=$(trim "$kind")$'\t'$(trim "$args")$'\t'$(trim "$clear")
  [[ $seat == reviewer-* ]] && pool+=("$seat")
done < <(grep '^|' "$file")

missing() {
  echo "SEAT MISSING $1" >&2
  exit 1
}

for seat in orchestrator implementer; do
  [[ -v row[$seat] ]] || missing "$seat"
done
chosen=(implementer)
if [[ $choice =~ ^[0-9]+$ ]]; then
  count=$((10#$choice))
  (( count >= 1 )) || bad_usage
  (( count <= ${#pool[@]} )) || missing reviewer
  chosen+=("${pool[@]:0:count}")
else
  IFS=, read -ra names <<<"$choice"
  for seat in "${names[@]}"; do
    [[ $seat == reviewer-* ]] || bad_usage
    [[ -v row[$seat] ]] || missing "$seat"
    chosen+=("$seat")
  done
fi

for seat in "${chosen[@]}"; do
  printf '%s\t%s\n' "$seat" "${row[$seat]}"
done
