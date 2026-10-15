#!/usr/bin/env bash
#
#
#
#
set -euo pipefail

agent=${1:?usage: receipt.sh AGENT TOKEN [POLLS]}
token=${2:?usage: receipt.sh AGENT TOKEN [POLLS]}
polls=${3:-120}
blocked() { echo "SEND BLOCKED $agent $1" >&2; exit 1; }
for _ in $(seq 1 "$polls"); do
  screen=$(herdr agent read "$agent" --source visible) || blocked "receipt read failed"
  grep -Fq "RECEIVED $token" <<<"$screen" && exit 0
  sleep 0.5
done
blocked "no receipt for $token"
