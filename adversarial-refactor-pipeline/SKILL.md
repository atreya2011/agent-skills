---
name: adversarial-refactor-pipeline
description: Run an explicit chat-only refactor pipeline through herdr tabs with one implementer, a mixed-model review panel, repeated review rounds, deletion-contract reconciliation, and a pull request. Invoke with /adversarial-refactor-pipeline.
disable-model-invocation: true
---

# Adversarial refactor pipeline, chat only

Run a large refactor without editing a single file yourself. One implementer agent makes every code change. A panel of reviewers from different model families tries to prove the work wrong. You consolidate, verify, relay, and keep score. Everything moves through tab chat. No instruction files, no shared scratch documents.

The separation is the point. An orchestrator who also edits code starts trusting its own diffs. An implementer who talks to reviewers directly starts negotiating. Keep the roles apart and route everything through the middle.

The acceptance criterion for the session lives in GOAL.md next to this file. Read it first.

## Local file

`~/.agents/local/adversarial-refactor-pipeline.md` holds the launch command for each role: the implementer and every reviewer seat. Read it before launching any agent; every launch uses the role's command exactly as written there.

## Roles

- Orchestrator (you). Writes briefs, verifies claims, consolidates findings, decides rejections, runs gates, ships. Never edits code.
- Implementer. One Sonnet agent in its own new tab. Owns the branch, commits one finding at a time, reruns gates after each, never pushes.
- Review panel. Two or more agents, each in its own new tab, including at least one Fable reviewer, and otherwise from different vendors or model families where possible. Different families fail differently. Overlap between their findings is your confirmation signal; their disagreements are where the judgment calls live. Read-only by instruction: no edits, no commits, no checkouts.

## The brief

Every dispatch is one self-contained chat message. It carries:

- Full task context. The receiving agent has none of yours.
- Hard rules restated every time: comment policy, the three-line floor on DRY, commit format, no amend, which gates to run and when.
- Intended behavior changes listed as exclusions, so reviewers do not rediscover them as bugs.
- A unique done signal per round (IMPL-DONE-R2, REVIEW-DONE-OPUS-R1). Never reuse one. Old signals persist in scrollback, and the brief itself echoes the signal, so detect completion by a count increasing, not by a single match.
- For reviewers: the output contract (flat ranked list, severity tags), the instruction to verify every candidate against the code before reporting, and "skip records are first-class".
- On substitute contracts: the substitute definition from GOAL.md and the sweep's surface list, copied in full. A reviewer briefed to grep names will certify the renames.
- Every dispatch carries the scope lock verbatim: base revision, priced rows, frozen invariants, tripwires. See Scope lock below.

Before dispatching, check every instruction in the brief against the assumptions the contract priced in. An instruction that negates a priced assumption voids those rows before any work starts, and nothing downstream will warn you.

## Scope lock

Freeze the base revision the moment loop step 1's findings table is priced — the SHA the table is scouted against, fixed for the whole session. It moves only when the owner approves a new priced baseline, never because the implementer rebased.

Every row in that table names the files it touches and the behavior it covers; that pairing is the changed-file allowlist. A behavior change with no row naming it is scope drift, not an intended break — intended breaks are only the ones an owner-approved priced row explicitly lists. An unnamed break found downstream goes back through pricing before it ships, same as any other row.

Before dispatch, also fix the frozen core invariants: the calculations, schemas, or contracts nothing in this session may alter. Attach a runnable proof to each — a test target, a golden fixture, a sample calculation — and capture its baseline result against the locked revision.

Architecture tripwires stop work outright, regardless of which row is open: persistence formats, public schemas, synchronization, versioning, and core calculations. None of these move unless a priced row explicitly says so — a tripwire lifts only for the row that names it, never on an implementer's initiative, never as a side effect of a fix for something else. Scope expands only through a new owner-approved priced row, priced and reviewed like every other row.

After every implementer commit, before anything else moves forward: map each changed hunk to the one row it belongs to, rerun the proofs for any frozen invariant the commit touched, and reconcile the running diffstat against the promised total. A commit passes only when every hunk maps to a row, no tripwire fires, and every frozen proof still matches its base result. The moment one of those fails, stop — do not brief the next finding, do not let the panel see the commit, do not scope the drift down to make it fit. Resume from the last passing commit in a fresh worktree; the drifted work goes back through pricing as its own row before it runs again.

## Dispatch mechanics

Terminal tabs are hostile, and every race here is silent. Skip none.

1. Boot before brief. After launching an agent in a new tab, wait, then confirm the tab reports an agent before sending anything. Text sent to a booting TUI lands nowhere, and nothing tells you.
2. Paste is a race. Long text stages in the composer without submitting. Sequence: send text, short sleep, Enter, check status. If the agent is not working, send a single space, Enter, then Enter again. The space keystroke unsticks bracketed paste.
3. Trust the context meter, not the transcript. Staged composer text renders wrapped in the transcript area and looks submitted. Proof of submission is the context percentage rising or the status flipping to working.
4. Ghost text lies. Agent TUIs stage plausible prompt suggestions in the composer, and they read exactly like a pending command the owner typed. Identical text appearing in freshly created tabs is the tell. Never submit it. Never treat it as user input.
5. Monitors die at the tool timeout. Write watch loops as self-reporting windows under the cap and chain them across wakeups. Require two consecutive non-working reads before declaring an agent stopped. Status transitions flicker.
6. Launch exactly what was asked. Exactly one agent per new tab. The launch profile and its permission flags are part of the request: launch each role with its command from the local file, nothing swapped in and nothing added. A generic agent or an added bypass flag is the substitute pattern the contract hunts, applied to your own dispatches.

## The semantic sweep

The substitute category in GOAL.md is hunted by behavior. A sweep walks control flow and classifies each path by its outcome under that definition, regardless of what the code calls it. Vocabulary scans certify renames.

A sweep covers every surface a substitute survives on: runtime code, tests, docs, skills, proto and API definitions, generated consumers, config, workflows, runbooks, and live PR claims. An absolute claim (zero substitutes left) extends only as far as the surfaces actually swept, and one failed spot-check invalidates it.

## The loop

1. Scout the repo yourself and produce the findings table with per-row line prices. That table is the contract — freeze it as the scope lock's base revision the moment it's priced. On substitute contracts, dispatch two independent semantic sweeps first and reconcile every confirmed path into the table before pricing.
2. Brief the implementer. Monitor to completion, running the scope-lock gate after every commit lands.
3. Verify independently. Run every gate yourself on the tip. Read the diffstat. Spot-read the diffs of the riskiest commits. The implementer's green is a claim, not evidence. For proto or API rows, green additionally means every producer and consumer updated, artifacts regenerated, and lint plus breaking checks run against the shipping baseline, not an intermediate shape the branch itself already discarded.
4. Brief the panel in parallel. Fresh contexts every round: one new tab per agent.
5. Consolidate. Two reviewers finding the same defect independently is near-proof. A solo finding gets cross-examined against the code before you accept it. Reject with written reasons. A rejection without a reason comes back next round wearing a new severity tag.
6. Relay confirmed fixes to the implementer as a numbered brief under the same hard rules. Include the rejections so they land in the PR body. Sweep each remediation diff the same way as the original finding: the fix's control flow decides, not its names.
7. Repeat until a round yields zero new confirmed defects. On substitute contracts a round is clean only after a fresh sweep of the fixed tip finds nothing. Trivia following a round with a real bug is diminishing returns. A comment tweak does not justify another full panel round; verify that diff yourself.
8. Reconcile the diffstat against the contract, row by row, before anything ships. A miss above roughly a third of the headline number stops the ship and goes back to the implementer as a completion pass. Run the scope-lock gate once more across the complete branch: every hunk mapped to a row, no tripwire fired, every frozen proof matching its base result.
9. Push, open the PR. The body lists what changed, the intended behavior changes, the review rounds with finding counts, the rejected findings with reasons, and gate status on the tip. Every absolute claim in the body ships with its evidence: the exact scoped searches, the control-flow audit, the changed-versus-base classification, the diffstat, full gates on the tip, and the live remote and PR state.

## Failure modes to check before every ship

- The brief contradicted the contract. Any instruction that preserves what a row priced for deletion voids that row silently.
- Review hardening ate the deletion. Strict fakes and new contract tests add lines back. Track the running net against the promise, not per-commit wins.
- The shared helper outgrew what it replaced. A dedup that lands net positive is a design miss. Slim it or inline it.
- Over-DRY. A helper extracted from a block under the three-line floor traded three plain lines for a call plus a definition. Inline it back.
- The fix laundered the substitute. Remediation renamed the path and kept the selection behavior. Sweep the fix's control flow exactly as the original; names carry no information.
- Reviewer lens mismatch. A reviewer applying library rules to a binary repo produces confident false bugs. State the repo's nature in the brief and reject accordingly.
- The refactor narrated itself. Sweep the diff for added comments that describe the change instead of the code, and for surviving comments whose referent the change deleted. Both classes die before the owner finds them.
- One unmapped hunk invalidates every later clean round. A round that finds nothing new does not retroactively map a drifted hunk from an earlier commit; the branch stays unshippable until it's mapped or reset.
