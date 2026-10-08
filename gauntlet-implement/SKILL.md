---
name: gauntlet-implement
description: Stage and implement one approved issue or spec into committed, cap-checked code on its own branch. Send the implementer brief through a chosen engine: the implementer seat in a herdr tab, or parallel subagents in git worktrees. Use standalone to run an implementation stage directly, or as gauntlet's stage 5.
---

# gauntlet-implement

Take one approved implementation issue or spec from a base SHA to
committed, cap-checked code on its own branch.
Apply KISS and YAGNI to the build: make the least change that satisfies the spec. Prefer the standard library, native capabilities, and existing dependencies over new ones. Add only abstractions the spec requires. Over-engineering a solution is evil; overly defensive programming is evil; hyper-fixating on rare or fictitious edge cases is evil.

## Load references

Read [../gauntlet/references/contracts.md](../gauntlet/references/contracts.md)
for the size contract and shared blocks. Load only the chosen engine file
under `engines/` and the template under
[briefs/implementer.md](briefs/implementer.md); load the other engine file
only if the caller's engine setting changes mid-run.

## 1. Pre-flight

Check the working tree with
`git status --porcelain`. Apply the caller's dirty-tree setting to the paths
that check finds. Record every pre-existing path and quarantine it in the
implementer brief so every mutation leaves it untouched.

## 2. Record the base SHA and branch

Record the base SHA that contains the code under change. Create one branch
from that SHA, decoupled from any worktree. Record the
branch, base SHA, and hard changed-line cap. When the branch already exists, adopt it to resume.

## 3. Send through the chosen engine

Render the implementer brief from
[briefs/implementer.md](briefs/implementer.md), filling every field from
current facts — base SHA, branch, scope, cap. Give the engine the
complete rendered seat brief every time; render it fresh for each send instead
of reusing an earlier render. Send it through the chosen engine:

- **tab** — the implementer seat in a herdr tab; see
  [engines/tab.md](engines/tab.md).
- **subagent-worktrees** — parallel `Agent`-tool subagents in git worktrees; see
  [engines/subagent-worktrees.md](engines/subagent-worktrees.md).

## 4. Verify yourself

Run build, uncached tests, lint, and generated-artifact checks yourself in
the orchestrating session — treat the engine's own report as a claim to
confirm, not a substitute for verification. Treat any warning or dirty
generated output as a failure and send the branch back to the engine for
another pass.

## 5. Cap check

Run the committed-range count per
[../gauntlet/references/contracts.md](../gauntlet/references/contracts.md)
after every implementer commit batch and every fix wave, and report it to the
dispatcher each time. MUST stop with `DECISION NEEDED` when the total exceeds
the issue's estimate or the cap. Resume review and PR creation after the
branch returns under the cap.

## Report

State the branch, base SHA, HEAD, changed-line count against the cap, and
which engine ran. Hand a branch under the cap to the caller's next stage. With the tab engine, the implementer tab stays open; the review stage sends it every fix brief.
