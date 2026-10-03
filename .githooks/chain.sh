#!/usr/bin/env bash
# Sourced by every hook in this directory. Pointing core.hooksPath at .githooks
# hides the hooks the user installed globally, so each hook here runs its global
# namesake first and its own logic after.

# Runs the hook named like the calling hook from the global core.hooksPath, when
# that path is set and holds an executable hook of that name, with the calling
# hook's arguments and standard input. Returns the global hook's status.
chain_global_hook() {
  local dir hook
  dir=$(git config --global --type=path core.hooksPath || true)
  [[ -n "$dir" ]] || return 0
  hook="$dir/${0##*/}"
  # A global path that points back at this directory would recurse.
  [[ -f "$hook" && -x "$hook" && ! "$hook" -ef "$0" ]] || return 0
  "$hook" "$@"
}
