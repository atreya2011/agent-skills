#!/usr/bin/env bash
# Portability check: fails when a tracked file, or a path given as an argument,
# contains a machine value (CONTEXT.md: a string that ties text to a person, an
# organization or a machine).
#
# Built-in patterns: absolute home-directory paths, email addresses and IPv4
# addresses. Extra patterns are read from PORTABILITY_PATTERNS_FILE, default
# ~/.agents/local/portability-patterns.txt, when that file exists: one extended
# regular expression per line, matched case-insensitively; a "w:" prefix matches
# the pattern as a whole word only; "#" lines and blank lines are ignored. Extra
# patterns are never printed; only the matched text is.
#
# tests/fixtures/ holds this check's own fixtures and is skipped in the tracked
# scan; pass a fixture path as an argument to scan it.
#
# Exit 1 with one file:line:match per hit; exit 0 when clean.
set -euo pipefail

patterns_file="${PORTABILITY_PATTERNS_FILE:-$HOME/.agents/local/portability-patterns.txt}"

# Matched anywhere in a line.
free_patterns=(
  '/home/[A-Za-z0-9._-]+'
  '/Users/[A-Za-z0-9._-]+'
  '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}'
)
# Matched as whole words only.
word_patterns=(
  '([0-9]{1,3}\.){3}[0-9]{1,3}'
)

if [[ -f "$patterns_file" ]]; then
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ -z "$line" || "$line" == \#* ]] && continue
    if [[ "$line" == w:* ]]; then
      word_patterns+=("${line#w:}")
    else
      free_patterns+=("$line")
    fi
  done <"$patterns_file"
fi

files=()
for path in "$@"; do
  [[ "$path" == /* ]] || path="$PWD/$path"
  files+=("$path")
done
cd "$(git rev-parse --show-toplevel)"
while IFS= read -r -d '' file; do
  [[ "$file" == tests/fixtures/* ]] && continue
  [[ -f "$file" ]] || continue
  files+=("$file")
done < <(git ls-files -z)
(( ${#files[@]} )) || exit 0

# Runs grep with the given options and -e patterns over every file. Returns 1
# on a hit, 0 on none; a grep failure ends the check with grep's status.
scan() {
  local status=0
  grep -a -n -o -i -E -H "$@" -- "${files[@]}" || status=$?
  case "$status" in
    0) return 1 ;;
    1) return 0 ;;
    *) exit "$status" ;;
  esac
}

free_args=()
for pattern in "${free_patterns[@]}"; do free_args+=(-e "$pattern"); done
word_args=()
for pattern in "${word_patterns[@]}"; do word_args+=(-e "$pattern"); done

clean=1
scan "${free_args[@]}" || clean=0
scan -w "${word_args[@]}" || clean=0
(( clean )) && exit 0
exit 1
