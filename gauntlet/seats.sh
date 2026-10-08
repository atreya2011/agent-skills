#!/usr/bin/env bash
# Reads the run's seat table, as the Local file section of SKILL.md defines it,
# and prints one line per seat to launch: the seat, its kind, its args and its
# clear command, separated by tabs. The lines come in launch order: the
# implementer, then the chosen reviewers. The orchestrator is the session that
# runs this script, so its row has to exist but is not printed.
#
#   seats.sh [--table <name>] <reviewers>
#
# Without --table the table is ~/.agents/local/gauntlet.md; with it,
# ~/.agents/local/gauntlet-<name>.md. <reviewers> is a count, which takes the
# first rows of the reviewer pool in table order, or full seat names of the form
# reviewer-<name> separated by commas, which take those rows in that order. A bare
# number is always a count, even when a seat is named reviewer-<number>.
#
# A missing table file prints SEAT MISSING orchestrator and the file's path. A
# seat the table lacks prints SEAT MISSING <seat>, and a count above the pool
# size prints SEAT MISSING reviewer. Each exits 1 before any line is printed. A
# missing or malformed argument prints the usage line and exits 2.
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

# Removes every backtick from a cell, then the blanks on both sides of it.
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

# Stops the run before any seat starts.
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
