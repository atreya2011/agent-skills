#!/usr/bin/env bash
set -euo pipefail
target=${1:?usage: goal.sh AGENT}
block() { echo "GOAL BLOCKED $target" >&2; exit 1; }
condition=$(tr '\r\n' '  ' | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')
test -n "$condition" || block
(( ${#condition} <= 4000 )) || block
kind=$(herdr agent get "$target" | jq -r '.result.agent.agent // empty') || block
[[ "$kind" == claude ]] || block
herdr agent prompt "$target" "/goal $condition" --wait --until working --timeout 15000 >/dev/null || block
for _ in $(seq 1 30); do
  screen=$(herdr agent read "$target" --source visible) || block
  grep -Fq '/goal active' <<<"$screen" && exit 0
  sleep 0.5
done
block
