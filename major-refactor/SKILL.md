---
name: major-refactor
description: Run an explicit managed program for a large or irreversible codebase change, with audits, a committed manifest, risk-ordered waves, subagents, and approval gates. Invoke with /major-refactor.
disable-model-invocation: true
---

# Change Program

Run a large, risky change as a managed program: audit, freeze scope, order work by risk, run children in parallel, gate every wave, and ask the user only for decisions that need an owner. Strict roles and a committed manifest keep the whole program in view. Adapted from Ryan Carson's 40-agent migration pattern.

## Step 0: Does this need a program?

Run a program only when the work has an irreversible step (production data, schema migrations, published API changes, anything without a cheap undo) or exceeds one session (more than about 100 files, more than 3 independent workstreams, or several days). Otherwise say so and use ordinary subagents; declining is a valid outcome.

## Step 1: Find program state

Program state lives only in the committed `docs/programs/<slug>/manifest.md`. On every invocation, run `ls docs/programs/*/manifest.md`. If a manifest matches, read it, report the wave in progress, pending rulings and failed gates, and resume from the next incomplete step; never rebuild state from conversation memory. Otherwise run Phase 0.

## Roles

The main session is the parent orchestrator: it plans, spawns children, reviews their reports, sequences waves, updates the manifest, and escalates to the user. It never edits product code; children do every audit, code change, gate run and doc update. Read [references/child-prompts.md](references/child-prompts.md) before spawning any child.

## Phase 0: Audit, then freeze the manifest

Read [references/phase-0.md](references/phase-0.md) before Phase 0 and [references/manifest-template.md](references/manifest-template.md) before writing the manifest.

The manifest is the source of truth; no child re-decides scope. Mark what the audit cannot resolve UNKNOWN, get a ruling instead of guessing, and record it in the manifest.

## Executing a wave

1. Create the branch `program/<slug>/wave-N`.
2. Spawn one child per unit. Parallel children use `isolation: "worktree"` and disjoint file boundaries.
3. Children merge back to the wave branch. Review each report and update unit status in the manifest.
4. A gate child runs the wave's gate contract. A failed gate gets a scoped fix child, then a rerun. Never mark a wave complete without its gate result in the manifest.
5. Open one PR per wave to the default branch; the user's PR approval is the human gate. Commit the manifest's wave status, gate log and approvals log with the wave.

Read [references/irreversible-steps.md](references/irreversible-steps.md) before any destructive or irreversible step.

## Escalation

Children stop and report when reality contradicts the manifest. Let unaffected units continue. At wave boundaries, ask rulings one question at a time on a phone-page question page; batch only when the user requests all remaining questions or runs cto. Log rulings in the manifest. Move waiting units to later waves and note why.

## Final wave

After the change ships, children update CLAUDE.md, AGENTS.md, README, repo skills and any agent knowledge base to describe the new architecture. Then mark the program complete in the manifest.
