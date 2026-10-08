#!/usr/bin/env bash
set -euo pipefail
worktree=${1:?worktree path}; branch=${2:?target branch}
git -C "$worktree" diff --quiet || { echo 'UNLANDED dirty tree'; exit 1; }
[[ -z "$(git -C "$worktree" status --porcelain)" ]] || { echo 'UNLANDED dirty tree'; exit 1; }
head=$(git -C "$worktree" rev-parse HEAD)
if ! git -C "$worktree" merge-base --is-ancestor "$head" "$branch"; then
  base=$(git -C "$worktree" merge-base "$head" "$branch")
  while IFS= read -r -d '' path; do
    git -C "$worktree" ls-tree -r --name-only "$branch" -- "$path" | grep -Fqx -- "$path" || { echo "UNLANDED $head not merged into $branch"; exit 1; }
  done < <(git -C "$worktree" -c core.quotePath=false diff --name-only -z "$base" "$head")
fi
echo LANDED
