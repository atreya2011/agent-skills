#!/usr/bin/env bash
set -euo pipefail
target=${1:?usage: goal.sh AGENT}
block() { echo "GOAL BLOCKED $target" >&2; exit 1; }
# A slash command is one line, so newlines in the condition become spaces.
condition=$(tr '\r\n' '  ' | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')
test -n "$condition" || block
(( ${#condition} <= 4000 )) || block
kind=$(herdr agent get "$target" | jq -r '.result.agent.agent // empty') || block
[[ "$kind" == claude ]] || block
herdr agent prompt "$target" "/goal $condition" --wait --until working --timeout 15000 >/dev/null || block
# Goal proof: the visible screen shows the armed goal; poll about 15 s.
for _ in $(seq 1 30); do
  screen=$(herdr agent read "$target" --source visible) || block
  grep -Fq '/goal active' <<<"$screen" && exit 0
  sleep 0.5
done
block
