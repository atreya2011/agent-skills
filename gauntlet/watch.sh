#!/usr/bin/env bash
set -euo pipefail
: "${@:?usage: watch.sh AGENT...}"
targets=("$@")
for target in "${targets[@]}"; do
  while ! herdr agent wait "$target" --until "done" --until idle --until blocked --timeout 300000 >/dev/null; do
    state=$(herdr agent get "$target") || { echo "SEAT UNCAPTURED $target" >&2; exit 1; }
    case "$state" in
      *agent_not_running*|*dead*|*vanished*) echo "SEAT DEAD $target" >&2; exit 1 ;;
      *working*|*running*) continue ;;
      *) echo "SEAT UNCAPTURED $target" >&2; exit 1 ;;
    esac
  done
  state=$(herdr agent get "$target") || { echo "SEAT UNCAPTURED $target" >&2; exit 1; }
  case "$state" in *agent_not_running*|*dead*|*vanished*) echo "SEAT DEAD $target" >&2; exit 1;; esac
  capture=$(herdr agent read "$target" --source recent-unwrapped --lines 400) || { echo "CAPTURE BLOCKED $target" >&2; exit 1; }
  [[ -n "$capture" ]] || { echo "CAPTURE BLOCKED $target" >&2; exit 1; }
  printf '%s\n' "$capture"
done
