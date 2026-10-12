---
name: gauntlet
description: Gauntlet a feature from facts through blind adversarial review to a merge-ready PR.
disable-model-invocation: true
---

# Gauntlet

## Local file

`~/.agents/local/gauntlet.md` holds the seat table: every seat's launch command. An alternate table is `~/.agents/local/gauntlet-<name>.md`. Read [references/seats.md](references/seats.md) before choosing the run's table and reviewers, writing a table, or launching a seat.

## Purpose

MUST take one approved implementation issue from facts to a merge-ready PR. Facts control transitions. Apply the simplicity contract in [references/contracts.md](references/contracts.md). Agents communicate in chat only; repo files carry only the deliverable itself.

## Load references

- Read [references/contracts.md](references/contracts.md) before sending any seat brief.
- Read [references/tab-dispatch.md](references/tab-dispatch.md) before spawning, prompting, watching, clearing, or closing any tab.
- Read [references/EVALS.md](references/EVALS.md) when changing or validating this skill. Check any change to the gauntlet skills against [references/VISION.md](references/VISION.md).

## Attribution

Read [references/contracts.md](references/contracts.md) §Attribution before sending kickoffs or creating commits, PRs, or comments.

## Decisions

Handle decisions per [references/contracts.md](references/contracts.md) §Inbox. At run start, act on every `## For agents` inbox entry that names this run.

## Session goal

Before sending a kickoff or `/goal` to a seat, read [references/tab-dispatch.md](references/tab-dispatch.md) §Session goal.

## 0. Discovery

Read the repository, configuration, state, and real data. Find the complication that breaks the naive framing. Keep discovery read-only.

## 1. Grill

Follow the grill-me skill. Resolve factual questions yourself from the repository and real data. Ask only genuine decision branches. Turn discovered traps into requirements.

## 2. Spec

Follow the to-spec skill. Record decisions, test seams, exclusions, and facts in the tracker.

## 3. Tickets

Follow the to-tickets skill. Read [references/contracts.md](references/contracts.md) §Issue and PR size contract before creating tickets.

## 4. Stage

Select one approved issue. Record the base SHA that contains the code under change and create one branch from it. Record and quarantine every pre-existing dirty or untracked path in every mutation brief. Read [references/drift-guard.md](references/drift-guard.md) before the first dispatch.

## 5. Implement

Hand the run to [gauntlet-implement](../gauntlet-implement/SKILL.md) with the engine (`tab` unless the user names `subagent-worktrees` at run start), spec = the selected issue, and dirty tree = quarantine each path. Leave the checkout untouched until the implementer reports done. An `IMPLEMENTER PARTIAL` report is not a stop; handle it per [references/contracts.md](references/contracts.md) §Inbox.

## 6. Verify

Run build, uncached tests, lint, generated-artifact checks, and quarantine hashes yourself. Treat warnings and dirty generated output as failures. Run the committed-range count from [references/contracts.md](references/contracts.md); stop before review above the hard cap. Push each verified HEAD. On the first push, open a draft PR per [../gauntlet-deliver/references/pr-conventions.md](../gauntlet-deliver/references/pr-conventions.md).

## 7. The gauntlet

Freeze scope and HEAD before each cycle, and keep valid load and stress tests intact. Run each cycle through [gauntlet-review](../gauntlet-review/SKILL.md) with the blind panel. After a fix, verify it, rerun the cap check, and start a fresh locked-HEAD cycle.

## 8. Deliver

Reconcile every acceptance check and rerun the final cap check. Hand delivery to [gauntlet-deliver](../gauntlet-deliver/SKILL.md) with the closing cycle's review notes. The user controls merge.
