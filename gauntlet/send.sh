#!/usr/bin/env bash
set -euo pipefail
target=${1:?usage: send.sh AGENT}
message=$(mktemp)
trap 'rm -f "$message"' EXIT
blocked() { echo "SEND BLOCKED $target $1" >&2; exit 1; }
cat >"$message"
test -s "$message" || blocked "empty message"
# The token stops at the first character outside [[:alnum:]_-], so the period in
# "Reply with RECEIVED abc." is not part of it.
token=$(sed -n 's/.*RECEIVED \([[:alnum:]_-]\{1,\}\).*/\1/p' "$message" | head -1)
test -n "$token" || blocked "no RECEIVED token"
head -n 1 "$message" | grep -q '^AGENT MESSAGE' || blocked "no agent marker"
info=$(herdr agent get "$target") || blocked "agent get failed"
kind=$(jq -r '.result.agent.agent // empty' <<<"$info") || blocked "agent get unparsable"
status=$(jq -r '.result.agent.agent_status // empty' <<<"$info") || blocked "agent get unparsable"
# Escape clears a staged composer before the paste, so it is sent only to a seat
# that is ready for input (herdr reports idle or done). On a working Claude Code
# seat Escape interrupts the running turn, and on a blocked seat it dismisses the
# approval dialog. Codex exits when it receives Escape on an empty composer, so
# the keypress is skipped for a codex seat.
if [[ "$kind" != codex && ( "$status" == idle || "$status" == "done" ) ]]; then
  herdr agent send-keys "$target" esc || blocked "esc failed"
fi
clear_state=$(herdr agent read "$target" --source visible) || blocked "screen read failed"
clear_state=$(sed '/^[[:space:]]*›/d' <<<"$clear_state")
test -n "$clear_state" || blocked "blank screen"
prompt_result=$(herdr agent prompt "$target" "$(<"$message")" --wait --until working --timeout 15000) || prompt_result=agent_prompt_timeout
composer=$(herdr agent read "$target" --source visible) || blocked "composer read failed"
if [[ "$prompt_result" == *agent_prompt_stalled* || "$composer" == *composer* ]]; then
  herdr agent send-keys "$target" Enter || blocked "enter failed"
fi
case "$(herdr agent get "$target")" in *working*|*running*) ;; *) blocked "not working after submit" ;; esac
# Delivery proof: the seat is working and the tab shows this send's unique
# token. The submitted prompt's own echo carries it; a reply is not required.
# Cursor collapses the paste to "[Pasted text #N +M lines]" in the composer, so the
# token appears in the transcript only once the agent echoes it. A seat that was
# working holds the paste as a queued message until its current tool call ends,
# so the echo can come much later. Both poll about 60 s, which stays inside the
# caller's default 2-minute Bash timeout.
if [[ "$kind" == cursor || "$status" == working ]]; then polls=120; else polls=10; fi
for _ in $(seq 1 "$polls"); do
  receipt=$(herdr agent read "$target" --lines 400) || blocked "receipt read failed"
  grep -Fq "RECEIVED $token" <<<"$receipt" && exit 0
  sleep 0.5
done
blocked "no receipt for $token"
