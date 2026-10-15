#!/usr/bin/env bash

chain_global_hook() {
  local dir hook
  dir=$(git config --global --type=path core.hooksPath || true)
  [[ -n "$dir" ]] || return 0
  hook="$dir/${0##*/}"
  [[ -f "$hook" && -x "$hook" && ! "$hook" -ef "$0" ]] || return 0
  "$hook" "$@"
}
