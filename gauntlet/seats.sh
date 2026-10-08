#!/usr/bin/env bash
# Reads the run's seat table, as the Local file section of SKILL.md defines it,
# and prints one line per chosen seat: the seat, its kind, its args and its
# clear command, separated by tabs. The lines come in launch order: the
# orchestrator, the implementer, then the chosen reviewers.
#
#   seats.sh [--table <name>] <reviewers>
#
# Without --table the table is gauntlet.md; with it, gauntlet-<name>.md. The
# directory is ~/.agents/local, or GAUNTLET_LOCAL_DIR when set. <reviewers> is a
# count, which takes the first rows of the reviewer pool in table order, or
# reviewer names separated by commas, which take the rows reviewer-<name> in
# that order.
#
# A missing table file, or a seat the table lacks, prints SEAT MISSING <seat>
# and exits 1 before any seat line is printed. A missing file lacks the
# orchestrator, so it names that seat. A count above the pool size prints SEAT
# MISSING reviewer. A count below one exits 2.
set -euo pipefail

usage='usage: seats.sh [--table <name>] <reviewers: a count, or names separated by commas>'
table=gauntlet
if [[ ${1-} == --table ]]; then
  table=gauntlet-${2:?$usage}
  shift 2
fi
choice=${1:?$usage}
(( $# == 1 )) || { echo "$usage" >&2; exit 2; }

file=${GAUNTLET_LOCAL_DIR:-$HOME/.agents/local}/$table.md

# Drops the code backticks around a cell and the blanks on both sides of it.
trim() {
  local cell=${1//\`/}
  cell=${cell#"${cell%%[![:space:]]*}"}
  printf '%s' "${cell%"${cell##*[![:space:]]}"}"
}

declare -A row
pool=()
if [[ -f $file ]]; then
  while IFS='|' read -r _ seat kind args clear _; do
    seat=$(trim "$seat")
    row[$seat]=$(trim "$kind")$'\t'$(trim "$args")$'\t'$(trim "$clear")
    [[ $seat == reviewer-* ]] && pool+=("$seat")
  done < <(grep '^|' "$file")
fi

# Stops the run before any seat starts.
missing() {
  echo "SEAT MISSING $1" >&2
  exit 1
}

chosen=(orchestrator implementer)
for seat in "${chosen[@]}"; do
  [[ -v row[$seat] ]] || missing "$seat"
done
if [[ $choice =~ ^[0-9]+$ ]]; then
  count=$((10#$choice))
  (( count >= 1 )) || { echo "$usage" >&2; exit 2; }
  (( count <= ${#pool[@]} )) || missing reviewer
  chosen+=("${pool[@]:0:count}")
else
  IFS=, read -ra names <<<"$choice"
  for name in "${names[@]}"; do
    [[ -v row[reviewer-$name] ]] || missing "reviewer-$name"
    chosen+=("reviewer-$name")
  done
fi

for seat in "${chosen[@]}"; do
  printf '%s\t%s\n' "$seat" "${row[$seat]}"
done
