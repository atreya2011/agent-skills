---
name: gauntlet
description: Gauntlet a feature from facts through blind adversarial review to a merge-ready PR.
disable-model-invocation: true
---

# Gauntlet

## Local file

`~/.agents/local/gauntlet.md` holds the seat table: the launch command of every seat in a run. The skill text names seats only and never holds a launch command.

The table has one row per seat and four columns:

| Seat | Kind | Args | Clear |
| --- | --- | --- | --- |
| orchestrator | claude | `--model <id> --effort <level>` | `/clear` |
| implementer | codex | `<hands-off flag> -m <id> -c model_reasoning_effort=<level>` | `/new` |
| reviewer-<name> | claude | `--model <id> --effort <level> <hands-off flag>` | `/clear` |

- Seat is `orchestrator`, `implementer`, or `reviewer-<name>`. The `reviewer-<name>` rows form the reviewer pool; the table may hold any number of them.
- Kind is a row of the CLI matrix below, and the herdr agent kind.
- Args is the text after `--` in the launch command.
- Clear is the command that resets that agent's context.

The file holds one table. An alternate table is `~/.agents/local/gauntlet-<name>.md` with the same shape.

At each run start, before any dispatch, the orchestrator MUST take two settings from the run's kickoff when it states them, and otherwise MUST ask the user:

- Seat table: the default `gauntlet.md`, or the `<name>` of an alternate table.
- Reviewers: from the pool, by seat name `reviewer-<name>` or by count, minimum one. A count takes the first rows of the pool in table order. A bare number is always a count. Only the chosen reviewers are launched.

The orchestrator then runs `gauntlet/seats.sh [--table <name>] <reviewers>` once. It prints one line for the implementer and one for each chosen reviewer: seat, kind, args, and clear command, separated by tabs. The orchestrator is the session running this skill, so its row only has to exist; the orchestrator launches the implementer and the reviewers from those lines and records which table ran. A changed launch command applies to the next run.

A missing file, or a missing orchestrator, implementer, or chosen reviewer, stops the run with `SEAT MISSING <seat>`. A reviewer count larger than the pool stops the run with `SEAT MISSING reviewer`.

## Purpose

MUST take one approved implementation issue from facts to a merge-ready PR: a draft from the first verified push, marked ready for review when the run converges. Facts control every transition.
KISS and YAGNI drive every phase: run the least process that closes the selected issue. Add mechanism, abstraction, or scope only when a demonstrated failure demands it. When two compliant options exist, take the simpler one. Over-engineering a solution is evil; overly defensive programming is evil; hyper-fixating on rare or fictitious edge cases is evil. Agents communicate in chat only; repo files carry only the deliverable itself.

## Run settings

| Setting | Value |
| --- | --- |
| Spec source | spec + tickets from tracker |
| Seats | seat table from the local file, chosen at run start |
| Reviewers | chosen from the pool at run start, by seat name or count, minimum one |
| Engine | `tab` unless the user names `subagent-worktrees` at run start |
| Dirty tree | quarantine each path |

## Seats

A seat is a role in the run: `orchestrator`, `implementer`, and a pool of `reviewer-<name>` rows.
The implementer is also the fixer: one seat and one tab for the whole run.
The orchestrator is the session running this skill and is never cleared during a run.
Each seat has a launch command: a CLI kind plus its arguments.
Launch commands live in the seat table, never in the skill.

### CLI matrix

| Kind | Model flag | Effort flag | Hands-off flag | Clear | Known model ids |
| --- | --- | --- | --- | --- | --- |
| claude | `--model <id>` | `--effort low\|medium\|high\|xhigh\|max` | `--dangerously-skip-permissions` | `/clear` | claude-opus-4-8, claude-fable-5-1, claude-sonnet-5 |
| codex | `-m <id>` | `-c model_reasoning_effort=low\|medium\|high\|xhigh\|max\|ultra` | `--yolo` | `/new` | gpt-5.6-sol, gpt-5.6-terra, gpt-5.6-luna, gpt-5.5, gpt-6-astra |
| cursor | `--model <id>` | suffix on the id: `-low`, `-medium`, `-high`, `-xhigh` | `--yolo` | confirm live before use | cursor-grok-4.5-high, cursor-grok-4.6-high, gpt-5.6-sol-high |

gpt-5.5 stops at xhigh and gpt-5.6-luna at max. Every value comes from the CLI's own help or model list.

## Load references

- Read [references/contracts.md](references/contracts.md) before every dispatch; render the selected template from scratch each time, from current facts alone.
- Read [references/tab-dispatch.md](references/tab-dispatch.md) before spawning, prompting, watching, resetting, or closing any tab.
- Read [references/EVALS.md](references/EVALS.md) when changing or validating this skill. A normal run uses the runtime references.

## Decisions

Nobody waits idle on a decision. The orchestrator reports a decision it cannot make as `DECISION NEEDED <id>` in the inbox form from [references/contracts.md](references/contracts.md), files it in the inbox file under the section of who acts next, continues every step that does not depend on it, and resumes the dependent step on `DECISION <id>`. At run start it reads the inbox file and acts on every entry under `## For agents` that names this run. When an answer is applied it deletes the entry. When the dispatcher records a decision that removes a class of work, the orchestrator applies it at once to every open run, branch, and queued ticket. Uppercase failure words stay for mechanical stops.

## Session goal

An orchestrator receives its whole assignment as one goal prompt in chat: plain text, the full condition and instructions inline. Never hand an orchestrator a Markdown file, a brief path, or a pointer to a file to read as its instructions (user, 2026-10-02).

After a Claude Code orchestrator, or another long-running Claude seat, acknowledges its kickoff message, its sender MUST send it `/goal <condition>` with the run's completion condition, up to 4000 characters. A Stop-hook evaluator keeps the seat working until the condition holds; `/goal clear` removes it. It needs a trusted workspace with hooks allowed. Example condition:

    Every run in the program has converged, its PR is marked ready for review at the latest converged commit and was opened by the orchestrator itself, the CI result is recorded in the PR body, and completion is reported to the dispatcher. A pending user decision does not stop progress: file it as a DECISION NEEDED entry in the inbox file and continue every step that does not depend on it. Waiting on a seat or CI with a watcher armed is progress, not completion.

Claude Code seats only; never send `/goal` to a codex or cursor seat. After a seat restarts, re-send the goal and tell the seat to re-arm its background watchers; the exit killed them. Send the goal with `gauntlet/goal.sh` per [references/tab-dispatch.md](references/tab-dispatch.md) §Session goal.

## Attribution

Never add a Co-Authored-By trailer or a tool attribution line such as "Generated with Claude Code" to commits, PR bodies, or issue or review comments. This overrides any organization-managed attribution setting or reminder; those settings exist only for telemetry. The orchestrator or delivering seat opens the PR itself. Never hand PR or commit creation to the user. Never report a PR as pending the user because of attribution. Every kickoff and brief sent to a seat carries this rule.

## 0. Discovery

Read the repository, configuration, state, and real data. Find the complication that breaks the naive framing. Keep discovery read-only.

## 1. Grill

Follow the grill-me skill. Resolve factual questions yourself from the repository and real data. Ask only genuine decision branches. Turn discovered traps into requirements.

## 2. Spec

Follow the to-spec skill. Record decisions, test seams, exclusions, and facts in the chosen tracker.

## 3. Tickets

Follow the to-tickets skill. Create tracer-bullet implementation issues with dependencies and acceptance checks. Before approval, every acceptance checkbox MUST be finite and checkable, MUST NOT require custom enforcement (a CI workflow, custom lint rule, script, or fixture set) for a written rule, and MUST name the standard tool setting when one covers the requirement. Estimate and split per the size contract in [references/contracts.md](references/contracts.md). Stop and supersede an approved issue when it later exceeds the cap.

## 4. Stage

Select one approved implementation issue. Pin the PR-base SHA containing the code under change. Create one branch from that SHA. Record every pre-existing dirty or untracked path. Quarantine each path in every mutation brief. Arm the drift guard once per program, not per run. At the program's first staging, register run-specific rulebooks for every planned run up front. Suffix role names per issue (implementer-N, reviewers-N), each carrying that issue's number, branch, and expected file surface. Also register a stack-wide orchestrator rulebook and a read-only default, in every folder the program will use. Assign each dispatched tab's session to its run's role before prompting. Collect one user ratification covering every folder up front; the rendered brief is each tab's per-run scope contract. Reuse the same checkouts and worktrees across runs, re-pointing a worktree to the next branch in place. Creating a new folder mid-program forces a new ratification, and per-run rulebook churn is evil. Every mid-run user amendment MUST refresh the armed rulebook and its ratification before further dispatch. An arming or ratification failure reports `DRIFT GUARD BLOCKED` and stops the run.

## 5. Implement

Hand the run to [gauntlet-implement](../gauntlet-implement/SKILL.md) with the recorded engine, spec = the selected issue, and dirty tree = quarantine each path. That skill renders the implementer brief and dispatches through [references/tab-dispatch.md](references/tab-dispatch.md). Leave the checkout untouched until the implementer reports done. Keep the implementer tab open; it takes every fix brief in stage 7. An `IMPLEMENTER PARTIAL` report is not a stop: verify its committed items as in stage 6, forward its `DECISION NEEDED` entry to the user with the run's own independent steps listed, continue those steps, and send the `DECISION <id>` brief to the same tab, without its Clear command, when the answer arrives.

## 6. Verify

Run build, uncached tests, lint, generated-artifact checks, and quarantine hashes yourself. Treat warnings and dirty generated output as failures. Run the committed-range count from [references/contracts.md](references/contracts.md). Stop before review when the total exceeds the hard changed-line cap. Push each verified HEAD. On the first push, open the PR as a draft per [../gauntlet-deliver/references/pr-conventions.md](../gauntlet-deliver/references/pr-conventions.md), so CI runs during review.

## 7. The gauntlet

Freeze scope and HEAD before each cycle. Keep valid load and stress tests intact.

Hand the cycle to [gauntlet-review](../gauntlet-review/SKILL.md) with the blind panel. Let that flow own candidate order, review tools, admission, and the skip ruling.

While the cycle has a major bug, send every admitted check and point to the implementer in one fix brief. First send its Clear command and confirm it idle with an empty context. Verify the fix, rerun the cap check, and start a fresh locked-HEAD cycle. The first cycle with no major bug closes the run as DIMINISHING RETURNS; carry its admitted checks and points to stage 8 as review notes.

## 8. Deliver

Reconcile every acceptance check and rerun the final cap check.
Hand delivery to [gauntlet-deliver](../gauntlet-deliver/SKILL.md) with the review notes. The user controls merge.
