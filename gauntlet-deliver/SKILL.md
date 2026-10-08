---
name: gauntlet-deliver
description: Deliver a converged gauntlet branch.
---

# Gauntlet delivery

Delivery MUST use the exact latest converged commit. Any mismatch is `DELIVERY BLOCKED`. Before closing tabs or removing worktrees, run `gauntlet/landed.sh` and proceed only on its landed proof.

Read the acceptance checks and verify the base SHA and cap. Deliver exactly one PR for the selected issue. The PR closes that issue and references parent and dependency issues. Mark the draft PR ready for review and follow [references/pr-conventions.md](references/pr-conventions.md), including the review notes from the closing cycle. Report the branch, SHA, changed-line count, and tab closure state.
