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
# Escape empties a staged composer before the paste, so it is sent only to a seat
# that is ready for input (herdr reports idle or done). On a working Claude Code
# seat Escape interrupts the running turn, and on a blocked seat it dismisses the
# approval dialog. Codex exits when it receives Escape on an empty composer, so
# the keypress is skipped for a codex seat.
if [[ "$kind" != codex && ( "$status" == idle || "$status" == "done" ) ]]; then
  herdr agent send-keys "$target" esc || blocked "esc failed"
fi
screen=$(herdr agent read "$target" --source visible) || blocked "screen read failed"
screen=$(sed '/^[[:space:]]*›/d' <<<"$screen")
test -n "$screen" || blocked "blank screen"
prompt_result=$(herdr agent prompt "$target" "$(<"$message")" --wait --until working --timeout 15000) || prompt_result=agent_prompt_timeout
composer=$(herdr agent read "$target" --source visible) || blocked "composer read failed"
if [[ "$prompt_result" == *agent_prompt_stalled* || "$composer" == *composer* ]]; then
  herdr agent send-keys "$target" Enter || blocked "enter failed"
fi
case "$(herdr agent get "$target")" in *working*|*running*) ;; *) blocked "not working after submit" ;; esac
# Delivery proof: the seat is working and its screen shows this send's unique
# token.
"${BASH_SOURCE[0]%/*}/receipt.sh" "$target" "$token"
