#!/usr/bin/env bash
# Tests for portability-check.sh. Each row runs the check on the repo plus one
# fixture (or the repo alone) with the extra-patterns file present or absent,
# then asserts the exit status and that every reported hit sits in the fixture.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"
check=./portability-check.sh
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# "absent" points at a file that does not exist; "extra" at the synthetic
# patterns fixture, so no case depends on ~/.agents/local.
patterns_absent="$tmp/absent.txt"
patterns_extra="$PWD/tests/fixtures/extra-patterns.txt"

# name | patterns file | fixture (empty = repo only) | expected exit
cases=(
  'built-in home path|absent|tests/fixtures/home-path.txt|1'
  'built-in Users path|absent|tests/fixtures/users-path.txt|1'
  'built-in email|absent|tests/fixtures/email.txt|1'
  'built-in IPv4|absent|tests/fixtures/ipv4.txt|1'
  'clean repo|absent||0'
  'extra pattern without the extra file|absent|tests/fixtures/extra-pattern.txt|0'
  'extra pattern with the extra file|extra|tests/fixtures/extra-pattern.txt|1'
  'whole-word extra pattern on a whole word|extra|tests/fixtures/extra-word.txt|1'
  'whole-word extra pattern inside a word|extra|tests/fixtures/extra-word-inside.txt|0'
)

failed=0
fail() {
  printf 'FAIL: %s: %s\n' "$1" "$2"
  failed=1
}

for row in "${cases[@]}"; do
  IFS='|' read -r name mode fixture want_status <<<"$row"
  case "$mode" in
    absent) patterns_file=$patterns_absent ;;
    extra) patterns_file=$patterns_extra ;;
  esac
  args=()
  [[ -n "$fixture" ]] && args+=("$fixture")

  status=0
  output=$(PORTABILITY_PATTERNS_FILE="$patterns_file" "$check" ${args[@]+"${args[@]}"}) || status=$?
  if [[ "$status" != "$want_status" ]]; then
    fail "$name" "exit $status, want $want_status; output: $output"
    continue
  fi

  if [[ "$want_status" == 0 ]]; then
    [[ -z "$output" ]] || fail "$name" "expected no output, got: $output"
    printf 'ok: %s\n' "$name"
    continue
  fi

  # Every hit names the fixture, and the reported text comes from the fixture.
  prefix="$PWD/$fixture:1:"
  reported=0
  while IFS= read -r line; do
    reported=1
    [[ "$line" == "$prefix"* ]] || { fail "$name" "hit outside the fixture: $line"; continue; }
    match=${line#"$prefix"}
    [[ -n "$match" ]] || fail "$name" "empty match in: $line"
    grep -qF -- "$match" "$fixture" || fail "$name" "reported text not in fixture: $match"
  done <<<"$output"
  (( reported )) || fail "$name" "no hit reported"
  printf 'ok: %s\n' "$name"
done

(( failed )) && exit 1
exit 0
