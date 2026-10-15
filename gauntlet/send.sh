#!/usr/bin/env bash
set -euo pipefail
target=${1:?usage: send.sh AGENT}
message=$(mktemp)
trap 'rm -f "$message"' EXIT
blocked() { echo "SEND BLOCKED $target $1" >&2; exit 1; }
cat >"$message"
test -s "$message" || blocked "empty message"
token=$(sed -n 's/.*RECEIVED \([[:alnum:]_-]\{1,\}\).*/\1/p' "$message" | head -1)
test -n "$token" || blocked "no RECEIVED token"
head -n 1 "$message" | grep -q '^AGENT MESSAGE' || blocked "no agent marker"
info=$(herdr agent get "$target") || blocked "agent get failed"
kind=$(jq -r '.result.agent.agent // empty' <<<"$info") || blocked "agent get unparsable"
status=$(jq -r '.result.agent.agent_status // empty' <<<"$info") || blocked "agent get unparsable"
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
"${BASH_SOURCE[0]%/*}/receipt.sh" "$target" "$token"
