#!/usr/bin/env bash
# Tests for gauntlet/seats.sh. Each row runs the script with HOME set to a
# temporary home directory whose .agents/local holds the tables under
# seat-tables/, then asserts the exit status, the seats printed and the refusal
# line. The tables are synthetic; a variant that lacks a seat is made from the
# default table in a temporary home directory.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"
seats="$PWD/../../gauntlet/seats.sh"
fixtures="$PWD/seat-tables"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# Makes the home directory $tmp/<name> with every fixture table in
# .agents/local, and drops the row of the given seat from the default table.
home() {
  mkdir -p "$tmp/$1/.agents/local"
  cp "$fixtures"/*.md "$tmp/$1/.agents/local/"
  [[ -z ${2-} ]] || grep -v "^| $2 " "$fixtures/gauntlet.md" >"$tmp/$1/.agents/local/gauntlet.md"
}
home default
home no-orchestrator orchestrator
home no-implementer implementer
home no-beta reviewer-beta
mkdir "$tmp/no-tables"

failed=0
fail() {
  printf 'FAIL: %s: %s\n' "$1" "$2"
  failed=1
}

# Runs the script in the given home directory with the given arguments, leaving
# its exit status, standard output and standard error in status, out and err.
run() {
  local argv
  read -ra argv <<<"$2"
  status=0
  out=$(HOME="$tmp/$1" "$seats" ${argv[@]+"${argv[@]}"} 2>"$tmp/err") || status=$?
  err=$(<"$tmp/err")
}

# name | home | arguments | seats printed, in order
resolve_cases=(
  "every seat resolves for the whole pool|default|3|implementer,reviewer-alpha,reviewer-beta,reviewer-gamma"
  "a count takes the first pool rows|default|2|implementer,reviewer-alpha,reviewer-beta"
  "a count of one launches one reviewer|default|1|implementer,reviewer-alpha"
  "seat names launch in the order given|default|reviewer-gamma,reviewer-alpha|implementer,reviewer-gamma,reviewer-alpha"
  "a reviewer outside the choice may be absent|no-beta|reviewer-alpha,reviewer-gamma|implementer,reviewer-alpha,reviewer-gamma"
  "an alternate table argument selects the alternate file|default|--table small 1|implementer,reviewer-solo"
  "a bare number is a count even when a seat has a numeric name|default|--table numeric 1|implementer,reviewer-2"
  "a count of two takes both numeric seats in table order|default|--table numeric 2|implementer,reviewer-2,reviewer-1"
  "a numeric seat name is taken in its full form|default|--table numeric reviewer-1|implementer,reviewer-1"
  "full numeric seat names launch in the order given|default|--table numeric reviewer-1,reviewer-2|implementer,reviewer-1,reviewer-2"
)
for row in "${resolve_cases[@]}"; do
  IFS='|' read -r name dir arguments want <<<"$row"
  run "$dir" "$arguments"
  if [[ "$status" != 0 ]]; then
    fail "$name" "exit $status, want 0; stderr: $err"
    continue
  fi
  got=$(cut -f1 <<<"$out" | paste -sd, -)
  [[ "$got" == "$want" ]] || { fail "$name" "seats $got, want $want"; continue; }
  printf 'ok: %s\n' "$name"
done

# A seat line carries the kind, the args and the clear command of its row,
# without the code backticks, and keeps a plain-text cell as it is. The
# orchestrator's row has to exist but is not launched.
name='a seat line holds the kind, the args and the clear command'
run default 3
want=$(printf '%s\t%s\t%s\t%s\n' \
  implementer codex '--yolo -m gpt-5.5 -c model_reasoning_effort=high' /new \
  reviewer-alpha claude '--model opus --effort high --dangerously-skip-permissions' /clear \
  reviewer-beta codex '--yolo -m gpt-5.5' /new \
  reviewer-gamma cursor '--yolo --model cursor-grok-4.5-high' 'confirm live before use')
if [[ "$status" == 0 && "$out" == "$want" ]]; then
  printf 'ok: %s\n' "$name"
else
  fail "$name" "exit $status; got: $out"
fi

# name | home | arguments | exit status | error line (a glob)
refusal_cases=(
  "a missing table file|no-tables|1|1|SEAT MISSING orchestrator (no $tmp/no-tables/.agents/local/gauntlet.md)"
  "an alternate table that does not exist|default|--table absent 1|1|SEAT MISSING orchestrator (no $tmp/default/.agents/local/gauntlet-absent.md)"
  "a removed orchestrator|no-orchestrator|1|1|SEAT MISSING orchestrator"
  "a removed implementer|no-implementer|1|1|SEAT MISSING implementer"
  "a removed chosen reviewer|no-beta|reviewer-alpha,reviewer-beta|1|SEAT MISSING reviewer-beta"
  "a reviewer seat outside the pool|default|reviewer-delta|1|SEAT MISSING reviewer-delta"
  "a numeric seat name that is not in the table|default|--table numeric reviewer-3|1|SEAT MISSING reviewer-3"
  "a count above the pool|default|4|1|SEAT MISSING reviewer"
  "a count above the pool once a row is removed|no-beta|3|1|SEAT MISSING reviewer"
  "a count of zero|default|0|2|usage:*"
  "a reviewer name without the reviewer- prefix|default|alpha|2|usage:*"
  "a seat that is not a reviewer|default|orchestrator|2|usage:*"
  "no reviewers argument|default||2|usage:*"
  "a table option without a name|default|--table|2|usage:*"
  "an extra argument|default|1 2|2|usage:*"
)
for row in "${refusal_cases[@]}"; do
  IFS='|' read -r name dir arguments want_status want_line <<<"$row"
  run "$dir" "$arguments"
  if [[ "$status" != "$want_status" ]]; then
    fail "$name" "exit $status, want $want_status; stderr: $err"
    continue
  fi
  [[ -z "$out" ]] || { fail "$name" "printed seats before refusing: $out"; continue; }
  # shellcheck disable=SC2254 # the error line is a glob on purpose
  case "$err" in
    $want_line) printf 'ok: %s\n' "$name" ;;
    *) fail "$name" "stderr '$err', want '$want_line'" ;;
  esac
done

(( failed )) && exit 1
exit 0
