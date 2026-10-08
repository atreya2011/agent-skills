#!/usr/bin/env bash
# Tests for the agent marker check in gauntlet/send.sh. The script checks the
# message before it touches a seat, so a refused message never reaches herdr. A
# message that passes the marker check goes on to ask herdr about the target;
# the target below names no seat, so herdr, or the shell when herdr is not
# installed, fails that step and the script reports it.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"
script="$PWD/../../gauntlet/send.sh"
target=gauntlet-test-no-such-seat
marker='AGENT MESSAGE (orchestrator, not a human).'

failed=0
fail() {
  printf 'FAIL: %s: %s\n' "$1" "$2"
  failed=1
}

# name | message, with \n for a line break | reason on the SEND BLOCKED line
cases=(
  "a message without the agent marker is refused|Fix this task. Begin your response with exactly: RECEIVED tok1|no agent marker"
  "a message whose first line is not the agent marker is refused|Context first.\n$marker Fix this task. Begin your response with exactly: RECEIVED tok2|no agent marker"
  "a message that opens with the agent marker gets past the marker check|$marker Fix this task. Begin your response with exactly: RECEIVED tok3|agent get failed"
)
for row in "${cases[@]}"; do
  IFS='|' read -r name message reason <<<"$row"
  status=0
  out=$(printf '%b\n' "$message" | "$script" "$target" 2>&1) || status=$?
  if [[ "$status" != 1 ]]; then
    fail "$name" "exit $status, want 1; output: $out"
    continue
  fi
  case "$out" in
    *"SEND BLOCKED $target $reason"*) printf 'ok: %s\n' "$name" ;;
    *) fail "$name" "no 'SEND BLOCKED $target $reason' in: $out" ;;
  esac
done

(( failed )) && exit 1
exit 0
