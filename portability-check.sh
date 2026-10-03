#!/usr/bin/env bash
# Portability check: fails when a machine value (CONTEXT.md: a string that ties
# text to a person, an organization or a machine) sits in file content, in a
# path name, in a symlink target or in a commit message.
#
#   portability-check.sh [path ...]      scans the worktree's tracked files and
#                                        their names, the targets of the tracked
#                                        symlinks, and any paths given as
#                                        arguments
#   portability-check.sh --commit <sha>  scans that commit's tree, the names and
#                                        symlink targets in it, and its message;
#                                        the pre-push hook runs this for every
#                                        commit it pushes
#
# Built-in patterns: absolute home-directory paths on Unix and on Windows, email
# addresses and IPv4 addresses. Extra patterns are read from
# PORTABILITY_PATTERNS_FILE, default ~/.agents/local/portability-patterns.txt,
# when that file exists; README's "Portability check" section gives their
# format. Extra patterns are never printed; only the matched text is.
#
# A word pattern — the built-in IPv4 pattern and every "w:" pattern — matches
# only where the character on each side is neither a letter nor a digit, so it
# still hits a token that "_" or "-" joins to another word, and never hits
# inside a longer word. Those boundary characters are part of the reported
# match. An example here would be a machine value itself, so the fixtures under
# tests/fixtures/ carry the cases instead.
#
# The fixtures hold machine values on purpose, so a scan skips every
# tests/fixtures/ path that portability-check.test.sh names and scans any other
# file there like the rest of the tree; pass a fixture path as an argument to
# scan it.
#
# Exit 1 with one <path>:<line>:<match> per content or symlink-target hit, one
# <path>:name:<match> per path-name hit and one <sha>:message:<match> per commit
# message hit; --commit puts the commit in front of every line. Exit 0 when
# clean.
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

# Matched anywhere in a line.
free_patterns=(
  '/home/[A-Za-z0-9._-]+'
  '/Users/[A-Za-z0-9._-]+'
  '[A-Za-z]:\\Users\\[A-Za-z0-9._-]+'
  '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}'
)
# Matched only between characters that are neither letters nor digits.
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

# The test names every fixture it uses, and those paths are the only ones a scan
# skips. A commit is read against its own test file.
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

# Reports whether the test names the given path as a fixture.
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

# Queues a string that is not file content, with the prefix its hits are
# reported under. Each line of the string becomes its own entry, so that the
# grep line number of a hit names the line's own label.
add_text() {
  local label=$1 line
  while IFS= read -r line; do
    labels+=("$label")
    texts+=("$line")
  done <<<"$2"
}

# Records a grep status: 0 is a hit, 1 a clean scan, anything else ends the
# check with grep's status.
record() {
  case "$1" in
    0) clean=0 ;;
    1) ;;
    *) exit "$1" ;;
  esac
}

# Scans the contents of the given files.
scan_files() {
  (( $# )) || return 0
  local status=0
  grep -a -n -o -i -E -H "${grep_args[@]}" -- "$@" || status=$?
  record "$status"
}

# Scans every queued string and prints "<label><match>" for each hit.
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
      # git grep skips a symlink's blob, so read the target out of the tree.
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
      # grep reads a symlink's target file, never the target string; scan that.
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
