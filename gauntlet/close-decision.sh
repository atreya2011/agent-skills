#!/usr/bin/env bash
#
#
#
#
#
#
set -euo pipefail

usage='usage: close-decision.sh LOCKED_HEAD < admission-list'
(( $# == 1 )) || { echo "$usage" >&2; exit 2; }
locked_head=$1

block() {
  echo "CLOSE BLOCKED $1" >&2
  exit 1
}

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
