#!/usr/bin/env bash
# Tests for portability-check.sh. Each row runs the check on the repo plus one
# fixture (or the repo alone) with the extra-patterns file present or absent,
# then asserts the exit status and that every reported hit sits in the fixture.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"
check="$PWD/portability-check.sh"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# "absent" points at a file that does not exist; "extra" at the synthetic
# patterns fixture and "crlf" at the same patterns with CRLF line endings, so no
# case depends on ~/.agents/local.
patterns_absent="$tmp/absent.txt"
patterns_extra="$PWD/tests/fixtures/extra-patterns.txt"
patterns_crlf="$PWD/tests/fixtures/extra-patterns-crlf.txt"

# name | patterns file | fixture (empty = repo only) | expected exit
cases=(
  'built-in home path|absent|tests/fixtures/home-path.txt|1'
  'built-in Users path|absent|tests/fixtures/users-path.txt|1'
  'built-in Windows Users path|absent|tests/fixtures/windows-path.txt|1'
  'built-in email|absent|tests/fixtures/email.txt|1'
  'built-in IPv4|absent|tests/fixtures/ipv4.txt|1'
  'built-in IPv4 joined by underscores|absent|tests/fixtures/ipv4-underscore.txt|1'
  # The boundary that keeps a longer word out keeps a letter-joined address out.
  'built-in IPv4 joined to a letter|absent|tests/fixtures/ipv4-letter-adjacent.txt|0'
  'clean repo|absent||0'
  'extra pattern without the extra file|absent|tests/fixtures/extra-pattern.txt|0'
  'extra pattern with the extra file|extra|tests/fixtures/extra-pattern.txt|1'
  'whole-word extra pattern on a whole word|extra|tests/fixtures/extra-word.txt|1'
  'whole-word extra pattern inside a word|extra|tests/fixtures/extra-word-inside.txt|0'
  'whole-word extra pattern joined by an underscore|extra|tests/fixtures/word-underscore.txt|1'
  'extra pattern from a crlf patterns file|crlf|tests/fixtures/extra-pattern.txt|1'
  'whole-word extra pattern from a crlf patterns file|crlf|tests/fixtures/extra-word.txt|1'
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
    crlf) patterns_file=$patterns_crlf ;;
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

# A tracked path name, a tracked symlink and a commit whose content the worktree
# no longer holds can only come from git, so they live in a throwaway repo. It
# gets an empty hooks path so that no global hook runs, and an identity whose
# host has no dot, which keeps this tracked file free of machine values.
repo="$tmp/repo"
mkdir -p "$tmp/nohooks"
git init -q -b main "$repo"
git -C "$repo" config core.hooksPath "$tmp/nohooks"
git -C "$repo" config user.name portability-check-test
git -C "$repo" config user.email test@localhost

# The planted values come from fixtures, so this file holds none of them.
bad_name=$(cat tests/fixtures/bad-name.txt)
bad_target=$(cat tests/fixtures/bad-link-target.txt)

# Commits everything in the throwaway repo and prints the new commit.
commit_repo() {
  git -C "$repo" add -A
  git -C "$repo" commit -q -m "$1"
  git -C "$repo" rev-parse HEAD
}

# Runs the check inside the throwaway repo, leaving its output in run_output and
# its exit status in run_status.
run_in_repo() {
  run_status=0
  run_output=$(cd "$repo" && PORTABILITY_PATTERNS_FILE="$patterns_absent" "$check") || run_status=$?
}

cp tests/fixtures/home-path.txt "$repo/doc.md"
commit_repo 'feat: add a doc' >/dev/null
git -C "$repo" rm -q doc.md
commit_repo 'chore: remove the doc' >/dev/null

name='worktree scan is clean once the machine value is removed'
run_in_repo
if [[ "$run_status" == 0 && -z "$run_output" ]]; then
  printf 'ok: %s\n' "$name"
else
  fail "$name" "exit $run_status, want 0; output: $run_output"
fi

printf 'The note says nothing about any machine.\n' >"$repo/$bad_name"
ln -s "$bad_target" "$repo/link"
commit_repo 'feat: add a note and a link' >/dev/null

name='worktree scan reaches names and symlink targets'
run_in_repo
[[ "$run_status" == 1 ]] || fail "$name" "exit $run_status, want 1; output: $run_output"

# behavior | line the worktree scan must report
worktree_cases=(
  "tracked path name|$bad_name:name:"
  "tracked symlink target|link:1:"
)
for row in "${worktree_cases[@]}"; do
  IFS='|' read -r name marker <<<"$row"
  case "$run_output" in
    *"$marker"*) printf 'ok: %s\n' "$name" ;;
    *) fail "$name" "no line starting $marker in: $run_output" ;;
  esac
done

(( failed )) && exit 1
exit 0
