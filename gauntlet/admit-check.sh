#!/usr/bin/env bash
set -euo pipefail

timeout_seconds=${CHECK_TIMEOUT_SECONDS:-10}
sha=${1:?locked SHA}
check=${2:?check file}
clause=${3:?clause string}
spec=${4:?spec file}

git diff --quiet "$sha" -- . || { echo 'REJECTED dirty tree'; exit 1; }
[[ -z "$(git status --porcelain)" ]] || { echo 'REJECTED dirty tree'; exit 1; }
test "$(git rev-parse HEAD)" = "$sha" || { echo 'REJECTED unlocked HEAD'; exit 1; }
grep -Fqx -e "$clause" -- "$spec" || { echo 'REJECTED missing clause'; exit 1; }
[[ -x "$check" ]] || { echo 'REJECTED check is not executable'; exit 1; }
for _ in 1 2; do
  [[ -z "$(git status --porcelain)" ]] || { echo 'REJECTED dirty tree'; exit 1; }
  if timeout "$timeout_seconds" "$check" >/dev/null 2>&1; then rc=0; else rc=$?; fi
  if (( rc == 0 )); then
    echo 'REJECTED check passed'
    exit 1
  fi
  (( rc == 124 || rc == 126 || rc == 127 )) && { echo "REJECTED non-result exit $rc"; exit 1; }
  [[ -z "$(git status --porcelain)" ]] || { echo 'REJECTED dirty tree'; exit 1; }
done
echo ACCEPTED
