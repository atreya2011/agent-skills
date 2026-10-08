#!/usr/bin/env bash
# Tests for gauntlet/seats.sh. Each row runs the script against a fixture
# directory that stands in for ~/.agents/local, then asserts the exit status,
# the seats printed and the refusal line. The tables under seat-tables/ are
# synthetic; a variant that lacks a seat is made from the default table in a
# temporary directory.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"
seats="$PWD/../../gauntlet/seats.sh"
fixtures="$PWD/seat-tables"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# Copies the default table into $tmp/<name>/ without the row of the given seat.
variant() {
  mkdir -p "$tmp/$1"
  grep -v "^| $2 " "$fixtures/gauntlet.md" >"$tmp/$1/gauntlet.md"
}
variant no-orchestrator orchestrator
variant no-implementer implementer
variant no-beta reviewer-beta
mkdir "$tmp/empty"

failed=0
fail() {
  printf 'FAIL: %s: %s\n' "$1" "$2"
  failed=1
}

# Runs the script in the given directory with the given arguments, leaving its
# exit status, standard output and standard error in status, out and err.
run() {
  local dir=$1 argv
  read -ra argv <<<"$2"
  status=0
  out=$(GAUNTLET_LOCAL_DIR="$dir" "$seats" "${argv[@]}" 2>"$tmp/err") || status=$?
  err=$(<"$tmp/err")
}

# name | directory | arguments | seats printed, in order
resolve_cases=(
  "every seat resolves for the whole pool|$fixtures|3|orchestrator,implementer,reviewer-alpha,reviewer-beta,reviewer-gamma"
  "a count takes the first pool rows|$fixtures|2|orchestrator,implementer,reviewer-alpha,reviewer-beta"
  "a count of one launches one reviewer|$fixtures|1|orchestrator,implementer,reviewer-alpha"
  "names launch in the order given|$fixtures|gamma,alpha|orchestrator,implementer,reviewer-gamma,reviewer-alpha"
  "a reviewer outside the choice may be absent|$tmp/no-beta|alpha,gamma|orchestrator,implementer,reviewer-alpha,reviewer-gamma"
  "an alternate table argument selects the alternate file|$fixtures|--table small 1|orchestrator,implementer,reviewer-solo"
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
# without the code backticks, and keeps a plain-text cell as it is.
name='a seat line holds the kind, the args and the clear command'
run "$fixtures" 3
want=$(printf '%s\t%s\t%s\t%s\n' \
  orchestrator claude '--model opus --effort high' /clear \
  implementer codex '--yolo -m gpt-5.5 -c model_reasoning_effort=high' /new \
  reviewer-alpha claude '--model opus --effort high --dangerously-skip-permissions' /clear \
  reviewer-beta codex '--yolo -m gpt-5.5' /new \
  reviewer-gamma cursor '--yolo --model cursor-grok-4.5-high' 'confirm live before use')
if [[ "$status" == 0 && "$out" == "$want" ]]; then
  printf 'ok: %s\n' "$name"
else
  fail "$name" "exit $status; got: $out"
fi

# name | directory | arguments | exit status | refusal line (a glob)
refusal_cases=(
  "a missing table file|$tmp/empty|1|1|SEAT MISSING orchestrator"
  "an alternate table that does not exist|$fixtures|--table absent 1|1|SEAT MISSING orchestrator"
  "a removed orchestrator|$tmp/no-orchestrator|1|1|SEAT MISSING orchestrator"
  "a removed implementer|$tmp/no-implementer|1|1|SEAT MISSING implementer"
  "a removed chosen reviewer|$tmp/no-beta|alpha,beta|1|SEAT MISSING reviewer-beta"
  "a reviewer name outside the pool|$fixtures|delta|1|SEAT MISSING reviewer-delta"
  "a count above the pool|$fixtures|4|1|SEAT MISSING reviewer"
  "a count above the pool once a row is removed|$tmp/no-beta|3|1|SEAT MISSING reviewer"
  "a count of zero|$fixtures|0|2|usage:*"
)
for row in "${refusal_cases[@]}"; do
  IFS='|' read -r name dir arguments want_status want_line <<<"$row"
  run "$dir" "$arguments"
  if [[ "$status" != "$want_status" ]]; then
    fail "$name" "exit $status, want $want_status; stderr: $err"
    continue
  fi
  [[ -z "$out" ]] || { fail "$name" "printed seats before refusing: $out"; continue; }
  # shellcheck disable=SC2254 # the refusal line is a glob on purpose
  case "$err" in
    $want_line) printf 'ok: %s\n' "$name" ;;
    *) fail "$name" "stderr '$err', want '$want_line'" ;;
  esac
done

(( failed )) && exit 1
exit 0
