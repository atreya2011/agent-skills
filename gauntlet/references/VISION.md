# Vision

This file is the acceptance policy for the gauntlet skills.
Compare current work against it.

## Identity

The gauntlet is a four-skill family: `gauntlet`, `gauntlet-implement`, `gauntlet-review`, and `gauntlet-deliver`, with the bash tools under `gauntlet/`.
It takes one approved issue from implementation to a merge-ready PR through adversarial review.
Review must never degrade work already proven correct.
The user is the final authority, and installing the skills happens only on the user's explicit approval.
A change the user directly orders lands without reviewer sign-off.

## Findings are executable checks

A finding is one executable check plus one exact quoted clause from the spec it enforces (`gauntlet/admit-check.sh`).
The quoted clause comes from the contract files, and the check proves itself within the check timeout.
The check fails on the clean locked commit.
The failure is reachable in real use from the shipping entrypoint.
Mock-only and simulated reproductions are inadmissible (`gauntlet/references/EVALS.md`).
Prose-only findings are inadmissible.
A check that fails from a missing tool proves nothing.
A check MAY run on any copy of the repo pinned to the locked commit.

## Check files stay out of the repository

Checks run at admission and at closure, and their files are never committed.

## The loop decides when to stop

A review point is a red check or one line of prose for a nit, design, or refactor item (#2).
The orchestrator rules every point admit or skip under the simplicity contract; a skipped point is dropped.
A major bug is defined in the Finding contract of [contracts.md](contracts.md).
While a cycle has a major bug, every admitted point is fixed by the implementer.
The first cycle with no major bug closes the run as `DIMINISHING RETURNS`, with every chosen reviewer captured (#2).
That cycle's admitted points go into the PR body under Review notes, and the user decides each one.
Nothing changes mid-cycle without a user order.

## Tools fail closed

Every render, send, watch, and teardown step proves success or blocks with an uppercase failure word.
Delivery uses the exact converged commit, and any mismatch is `DELIVERY BLOCKED`.
A decision is not a failure: the agent posts `DECISION NEEDED <id>` with background, situation, rule, stakes, options with consequences, recommendation and what depends on it, finishes every independent item, and stops with a partial line.
The inbox file holds only open entries, in two sections by who acts next; a resolved entry is deleted.
When the target moves after convergence, only the closure checks rerun on the new base.
Tabs and worktrees close only after `gauntlet/landed.sh` returns landed proof.

## Guidance stays minimal and affirmative

Rules are stated positively.
Deletion is the default fix for prose.
KISS and YAGNI gate every fix and implementation, and a fix changes only what its admitted check or point names.

## Scope and non-goals

In scope: one branch, one selected issue, one PR per run, opened as a draft and marked ready for review when the run converges.
Out of scope: commit protocols, sandboxes, and generated held-out test cases.
Also out of scope: semantic deduplication of findings, new roles, and gates beyond the ones the skills name.
A stop flag raised by the drift guard is cleared only by the user.

## Alignment tests

- A finding without an executable check is rejected at admission.
- A check that passes on the clean locked commit is inadmissible.
- A cycle with no major bug ends the run, and its admitted points appear in the PR body.
- A delivery whose commit differs from the converged commit blocks.
- A proposal to add a role, gate, or sandbox is out of scope regardless of merit.
- An agent that stops on a decision while independent items remain unfinished fails the contract.
- An inbox entry without background, situation, rule, stakes, options with consequences and a recommendation is returned unfiled; a reader who has not followed the run must be able to decide from the entry alone.
