#!/usr/bin/env bash
# Decides how a review cycle ends from the cycle's admission list.
#
#   close-decision.sh <locked-head> < admission-list
#
# The admission list holds one ruled item per line, with fields separated by
# tabs; blank lines are ignored:
#
#   major<TAB><check file><TAB><clause>
#   minor<TAB><check file><TAB><clause>
#   point<TAB><text>
#   skip<TAB><text>
#
# A check file holds the check as the reviewer submitted it and may span many
# lines. The script reads it, so the file's path never reaches the output. A
# skip line is a skipped point and is dropped.
#
# With at least one major line the cycle needs a fix. The script prints FIX, then
# the "- Check:" items, then the "- Point:" items or, with no points, none. Each
# list is ready to paste into the fix brief. Otherwise the run closes. The
# script prints DIMINISHING RETURNS and, when any item was admitted, the
# "## Review notes" list for the PR body: each item with the locked head it
# was raised on.
#
# A line with another first field, or a check file that cannot be read, prints
# CLOSE BLOCKED and exits 1 with nothing on standard output.
set -euo pipefail

usage='usage: close-decision.sh LOCKED_HEAD < admission-list'
(( $# == 1 )) || { echo "$usage" >&2; exit 2; }
locked_head=$1

# Stops before anything is printed.
block() {
  echo "CLOSE BLOCKED $1" >&2
  exit 1
}

# Prints a check as a list item: the file's content in a code block, then the
# clause.
check_item() {
  local content
  content=$(<"$1")
  printf -- '- Check:\n  ```\n'
  printf '%s\n' "$content" | sed 's/^./  &/'
  printf -- '  ```\n  Clause: %s\n' "$2"
}

checks=()
points=()
major=0
while IFS=$'\t' read -r class first second || [[ -n $class ]]; do
  [[ $class =~ ^[[:space:]]*$ ]] && continue
  case $class in
    major | minor)
      [[ -r $first ]] || block "unreadable check file $first"
      [[ $class == major ]] && major=1
      checks+=("$(check_item "$first" "$second")")
      ;;
    point) points+=("- Point: $first") ;;
    skip) ;;
    *) block "unknown class $class" ;;
  esac
done

if (( major )); then
  echo FIX
  printf '%s\n' "${checks[@]}"
  if (( ${#points[@]} )); then
    printf '%s\n' "${points[@]}"
  else
    echo none
  fi
  exit 0
fi

echo 'DIMINISHING RETURNS'
items=("${checks[@]}" "${points[@]}")
if (( ${#items[@]} )); then
  echo '## Review notes'
  for item in "${items[@]}"; do
    printf '%s\n  HEAD: %s\n' "$item" "$locked_head"
  done
fi
