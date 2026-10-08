#!/usr/bin/env bash
# Waits for the receipt of a send: the visible screen of AGENT shows
# RECEIVED <token>. Run by send.sh once the seat is working.
#
#   receipt.sh AGENT TOKEN [POLLS]
#
# Each poll reads the visible screen and waits half a second. The default of 120
# polls is about 60 s, which stays inside the caller's default 2-minute Bash
# timeout. The screen, not the last lines of history, is read because herdr
# refuses a read of more lines while a seat is working.
#
# The submitted prompt's echo carries the token. Cursor collapses the paste to
# "[Pasted text #N +M lines]", and a long paste can scroll off the screen, so
# the token may show only once the seat replies. A seat that was working holds
# the paste as a queued message until its current tool call ends, so the reply
# can come much later.
#
# Exits 0 on the receipt. Otherwise prints SEND BLOCKED and exits 1.
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
