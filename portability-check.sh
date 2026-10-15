#!/usr/bin/env bash
#
#
#
#
#
set -euo pipefail

commit=""
if [[ "${1-}" == --commit ]]; then
  commit="${2-}"
  if [[ -z "$commit" || $# -gt 2 ]]; then
    printf 'usage: %s --commit <sha>\n' "${0##*/}" >&2
    exit 2
  fi
  shift 2
fi

patterns_file="${PORTABILITY_PATTERNS_FILE:-$HOME/.agents/local/portability-patterns.txt}"

free_patterns=(
  '/home/[A-Za-z0-9._-]+'
  '/Users/[A-Za-z0-9._-]+'
  '[A-Za-z]:\\Users\\[A-Za-z0-9._-]+'
  '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}'
)
word_patterns=(
  '([0-9]{1,3}\.){3}[0-9]{1,3}'
)

if [[ -f "$patterns_file" ]]; then
  while IFS= read -r line || [[ -n "$line" ]]; do
    line=${line%$'\r'}
    [[ -z "$line" || "$line" == \#* ]] && continue
    if [[ "$line" == w:* ]]; then
      word_patterns+=("${line#w:}")
    else
      free_patterns+=("$line")
    fi
  done <"$patterns_file"
fi

grep_args=()
for pattern in "${free_patterns[@]}"; do grep_args+=(-e "$pattern"); done
for pattern in "${word_patterns[@]}"; do
  grep_args+=(-e "(^|[^0-9A-Za-z])($pattern)($|[^0-9A-Za-z])")
done

files=()
for path in "$@"; do
  [[ "$path" == /* ]] || path="$PWD/$path"
  files+=("$path")
done
cd "$(git rev-parse --show-toplevel)"

test_source=""
if [[ -n "$commit" ]]; then
  test_source=$(git show "$commit:portability-check.test.sh" 2>/dev/null || true)
elif [[ -f portability-check.test.sh ]]; then
  test_source=$(<portability-check.test.sh)
fi
exempt=()
while IFS= read -r path; do
  exempt+=("$path")
done < <(printf '%s\n' "$test_source" | grep -o -E 'tests/fixtures/[A-Za-z0-9._-]+' | sort -u)

is_exempt() {
  local exempt_path
  for exempt_path in ${exempt[@]+"${exempt[@]}"}; do
    [[ "$1" == "$exempt_path" ]] && return 0
  done
  return 1
}

clean=1
labels=()
texts=()

add_text() {
  local label=$1 line
  while IFS= read -r line; do
    labels+=("$label")
    texts+=("$line")
  done <<<"$2"
}

record() {
  case "$1" in
    0) clean=0 ;;
    1) ;;
    *) exit "$1" ;;
  esac
}

scan_files() {
  (( $# )) || return 0
  local status=0
  grep -a -n -o -i -E -H "${grep_args[@]}" -- "$@" || status=$?
  record "$status"
}

scan_texts() {
  (( ${#texts[@]} )) || return 0
  local out status=0 line number
  out=$(printf '%s\n' "${texts[@]}" | grep -a -n -o -i -E "${grep_args[@]}") || status=$?
  record "$status"
  [[ "$status" == 0 ]] || return 0
  while IFS= read -r line; do
    number=${line%%:*}
    printf '%s%s\n' "${labels[number-1]}" "${line#*:}"
  done <<<"$out"
}

if [[ -n "$commit" ]]; then
  pathspec=(.)
  for path in ${exempt[@]+"${exempt[@]}"}; do pathspec+=(":!$path"); done
  status=0
  git grep -a -n -o -i -E "${grep_args[@]}" "$commit" -- "${pathspec[@]}" || status=$?
  record "$status"

  while IFS= read -r -d '' entry; do
    meta=${entry%%$'\t'*}
    file=${entry#*$'\t'}
    is_exempt "$file" && continue
    add_text "$commit:$file:name:" "$file"
    if [[ "${meta%% *}" == 120000 ]]; then
      add_text "$commit:$file:1:" "$(git cat-file -p "${meta##* }")"
    fi
  done < <(git ls-tree -r -z "$commit")

  add_text "$commit:message:" "$(git log -1 --format=%B "$commit")"
else
  while IFS= read -r -d '' entry; do
    mode=${entry%% *}
    file=${entry#*$'\t'}
    is_exempt "$file" && continue
    add_text "$file:name:" "$file"
    if [[ "$mode" == 120000 ]]; then
      add_text "$file:1:" "$(readlink "$file" 2>/dev/null || true)"
    elif [[ -f "$file" ]]; then
      files+=("$file")
    fi
  done < <(git ls-files -s -z)
fi

scan_files ${files[@]+"${files[@]}"}
scan_texts
(( clean )) && exit 0
exit 1
