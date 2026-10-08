#!/usr/bin/env bash
# Tests for gauntlet/receipt.sh against a real herdr server. The seat is
# synthetic: a throwaway tab whose shell shows a token on the alternate screen,
# as an agent CLI does, and which herdr's own report-agent marks as a working
# claude seat. herdr refuses a read of more lines from such a seat, which is what
# the receipt check must cope with. Without a reachable herdr server the test
# skips.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"
receipt="$PWD/../../gauntlet/receipt.sh"

if [[ -z ${HERDR_WORKSPACE_ID-} ]] || ! herdr agent list >/dev/null 2>&1; then
  echo 'skip: no reachable herdr server'
  exit 0
fi

tmp=$(mktemp -d)
created=$(herdr tab create --workspace "$HERDR_WORKSPACE_ID" --cwd "$tmp" --label receipt-test --no-focus)
tab=$(jq -r '.result.tab.tab_id' <<<"$created")
pane=$(jq -r '.result.root_pane.pane_id' <<<"$created")
trap 'herdr tab close "$tab" >/dev/null 2>&1 || true; rm -rf "$tmp"' EXIT

failed=0
fail() {
  printf 'FAIL: %s: %s\n' "$1" "$2"
  failed=1
}

herdr pane report-agent --source receipt-test --agent claude --state working "$pane" >/dev/null
herdr pane run "$pane" 'tput smcup; echo RECEIVED shown-token; sleep 300' >/dev/null
for _ in $(seq 1 20); do
  herdr agent read "$pane" --source visible | grep -Eq '^RECEIVED shown-token *$' && break
  sleep 0.5
done

# The premise: herdr refuses a long read of a working seat.
name='herdr refuses a long read of a working seat'
status=0
out=$(herdr agent read "$pane" --lines 400 2>&1) || status=$?
if [[ "$status" != 0 && "$out" == *agent_not_idle* ]]; then
  printf 'ok: %s\n' "$name"
else
  fail "$name" "exit $status; output: $out"
fi

# name | agent | token | polls | expected exit | error line
cases=(
  "the token on the screen of a working seat is the receipt|$pane|shown-token|4|0|"
  "a token that is not on the screen gets no receipt|$pane|absent-token|2|1|SEND BLOCKED $pane no receipt for absent-token"
  "an agent herdr does not know is reported|gauntlet-test-no-such-agent|shown-token|2|1|SEND BLOCKED gauntlet-test-no-such-agent receipt read failed"
)
for row in "${cases[@]}"; do
  IFS='|' read -r name agent token polls want_status want_line <<<"$row"
  status=0
  "$receipt" "$agent" "$token" "$polls" >/dev/null 2>"$tmp/err" || status=$?
  err=$(grep -v '^{' "$tmp/err" || true)
  if [[ "$status" != "$want_status" ]]; then
    fail "$name" "exit $status, want $want_status; stderr: $err"
  elif [[ "$err" != "$want_line" ]]; then
    fail "$name" "stderr '$err', want '$want_line'"
  else
    printf 'ok: %s\n' "$name"
  fi
done

(( failed )) && exit 1
exit 0
