---
name: cto
description: Run one CTO sweep. Rebuild the picture of every herdr tab, session, todo and pull request from sources, close todos only on proof, put judgment calls to the user in one batch, dispatch ready todos, and write the brief. Use when a prompt says to run the cto skill, sweep, or reset.
---

# CTO

The CTO runs in one long-lived tab. On every sweep it rebuilds its picture from sources, never from chat memory. The `cto-snapshot` binary reads the sources and judges each todo and session with rules and gates. The CTO acts on those judgments and writes the brief. Words follow `CONTEXT.md`.

## Local file

`~/.agents/local/cto.md` holds every machine value. Read it at the start of every sweep. It has one section per field:

- Pinned: tab and workspace labels that a sweep never questions.
- Profiles: per profile, its name, its agent CLI kind (`claude` or `codex`) and its transcript directory.
- Launch command: the exact shell command that starts the CTO's agent CLI.
- Vault map: per project, the vault root, and the `llm-wiki` command.
- Cron times: the sweep interval, the waking hours and the reset time.
- Taskwarrior: the absolute path of the Taskwarrior binary. Bare `task` can be a different tool.
- Thresholds: the confidence cut-offs. The starting values are provisional until the ground-truth run sets them.
- Rulings: standing instructions from the user, one bullet each, dated.

The tool reads its settings from fenced `toml` blocks in these sections and rejects an unknown field. The blocks join into one document, so put each bare key under a table:

```toml
[commands]               # task is required; herdr, gh and git default to their bare names
task = "<absolute path>"
[pinned]
tabs = ["<tab label>"]
workspaces = ["<workspace label>"]
[[profiles]]
name = "<profile name>"
kind = "claude"          # or "codex"
dir = "<transcript directory>"
[vaults]
"<owner/name, or the project directory>" = "<vault root>"
[thresholds]
floor = 0.6              # below this, every verdict escalates
low = 0.6                # low-stakes acts: an In progress line, a Queued rank
high = 0.8               # high-stakes acts: closing or rewriting a todo
window_hours = 6         # sessions written within this window are read
tail_bytes = 65536
excerpt_chars = 800
[gate]
log = "<gate-call log path>"
```

Two rulings have a fixed format:

- `Dispatch default: seat table <default or a table name>; reviewers <a count, or reviewer-<name> seats separated by commas>`. The install step writes it. Every kickoff states both settings, so a dispatch never waits on the user for them.
- `Assignee split: <the user's rule for +cos and +cto>`, followed by `Assignee split applied: <date>` once the first sweep has applied it.

The API key is never in this file. The tool reads `TYPESAFE_API_KEY` from the environment, else the OS keyring entry with service `typesafe` and user `api`. Without a key every gate escalates.

## Install

1. Build the tool: `go build -o ../bin/cto-snapshot .` from this skill's `snapshot` directory.
2. Store the API key once. On Linux: `secret-tool store --label=typesafe service typesafe username api`. On macOS: `security add-generic-password -s typesafe -a api -w`. Each command prompts for the key.
3. Create the local file from the sections above, including the `Dispatch default` ruling.
4. Create the CTO tab: `herdr workspace create --cwd <directory> --label cto --no-focus`. Take the root pane ID from the reply, run the launch command there with `herdr pane run <pane> "<launch command>"`, then `herdr agent rename <pane> cto`.
5. Add the crontab entries below, then work through `references/checklist.md`.

The crontab addresses the tab by its agent name. `PATH` must include the directory of `herdr`. The first line sweeps at the interval during waking hours. The second resets at the reset time. Each waits for the tab to settle, then sends only when it is `idle` or `done`. A `blocked` tab gets nothing, because the blocked tab is already the user's notification. After `/clear` the reset sends Enter once, because the command can stay staged in the composer; on an empty composer the Enter does nothing.

```
*/<interval in minutes> <first hour>-<last hour> * * * herdr agent wait cto --timeout 120000; herdr agent get cto | grep -Eq '"agent_status":"(idle|done)"' && herdr agent prompt cto "AGENT MESSAGE (cron). Run the cto skill: sweep."
0 <reset hour> * * * herdr agent wait cto --timeout 600000; herdr agent get cto | grep -Eq '"agent_status":"(idle|done)"' && herdr agent prompt cto "/clear" && herdr agent send-keys cto Enter && sleep 5 && herdr agent prompt cto "AGENT MESSAGE (cron). Run the cto skill: sweep."
```

## Sweep

1. Read the local file. If it has an Import section, do `references/import.md` now.
2. If the `Assignee split` ruling has no `applied` line, apply it once: run `task <uuid> modify +cto` or `+cos` for every todo the ruling covers. A todo it does not cover keeps no tag and becomes a judgment call. Then record `Assignee split applied: <date>`.
3. Run `bin/cto-snapshot`. If it exits nonzero, say so in one line and stop.
4. Read `unreadable`. Each entry is a source the tool could not read. List them in the brief under Not checked. Never infer from silence: a tab, todo or project that depends on an unreadable source is reported as unchecked, and no todo changes on a bad read.
5. Apply the judgments below.
6. Clean up tabs (below).
7. Dispatch ready todos (below).
8. Make wiki writes (below).
9. If any tab has state `blocked`, raise one notification: `herdr notification show "Blocked tab" --body "<project>: <tab labels>" --sound request`.
10. Write the brief.

## Judgments

The snapshot's `judgments` hold one entry per judged todo and per session. The order is rule, gate, judgment call. A rule decides with no model call. Otherwise a gate decides: a typed label with calibrated confidence. Each entry has a `verdict`. The CTO never reads transcripts itself and never overrides a verdict.

- Todo, verdict `act`, label `done`: `task <uuid> done`, then `task <uuid> annotate "<annotation>"`. Rules and gates both give an `annotation`; write it unchanged, so the verdict is recorded where the decision lands.
- Todo, verdict `act`, label `stale`: rewrite the description from the facts in the snapshot (`task <uuid> modify`) and annotate. Keep every URL, `commit:` and `session:` reference that the old description holds.
- Todo, verdict `act`, label `open`: change nothing. The todo is In progress or Queued.
- A rule or an `act` verdict settles the todo. If it is tagged `+ask`, run `task <uuid> modify -ask`.
- Todo, verdict `escalate`: this is a judgment call. Tag it `+ask` if it is not tagged yet. Annotate `ask: <one plain question> | suggested: <label> | <annotation>` only when the todo was not `+ask` before or the question changed. An unchanged question is not annotated again. If `reason` says a source is unreadable, the todo has no assignee tag, or the gate failed, put that in the question and suggest nothing.
- Session: the label is the In progress line. An `escalate` verdict means "activity unknown".
- Todos tagged `+cos` belong to the CoS. Never change them.

The tool leaves the `ask:` and `cto ` annotations out of the gate's input, so an old question or verdict never feeds a new one. When the user answers a judgment call, apply the answer, remove `+ask` and annotate the todo with the answer.

## Tab cleanup

Do this for each workspace that is not pinned and whose state is `idle` or `done`. herdr's `done` is the same state as `idle`, not yet seen, so treat the two alike. A workspace's orchestrator is its tab labeled `orchestrator`. A workspace with no such tab is reported with its state and left alone. A pinned tab or workspace is never questioned, and a pinned tab is never closed, even inside an unpinned workspace.

1. Send the orchestrator a chat message: `herdr agent prompt <target> --wait --timeout 120000 "AGENT MESSAGE (cto, not a human). Are your tabs still needed? Reply done only if your work is finished and every tab can close. If you finished any of these todos, name them: <the project's open +cto todos>."`
2. Read the reply with `herdr agent read <target> --source recent-unwrapped --lines 60` and pipe the final message into `bin/cto-snapshot reply`.
3. If the verdict is `act` and the label is `done`, the orchestrator has said done. That is proof: close the todos it named and annotate them with the printed `annotation`, then close each tab of the workspace that is `idle` or `done` and not pinned, with `herdr tab close <tab_id>`, the orchestrator's last. If a pinned tab remains, the workspace stays open and is reported.
4. A timeout, a `blocked` state or any other verdict is "not done". Report the workspace with its state and age in Needs you. Close nothing.

Talk to orchestrators by chat only. Never edit their files.

## Dispatch

Dispatch only a todo that has `ready` true, assignee `cto` and `dispatched` false. A todo that is not ready is listed in Queued as not ready and is never dispatched. The user controls unspecified work.

The steps follow the gauntlet skill and its `references/tab-dispatch.md`. Name the orchestrator's agent `orchestrator-<project label>-<issue number>`, because herdr agent names are unique per server. Lowercase the project label, replace each character outside `a-z`, `0-9` and `-` with `-`, and shorten the label so the whole name stays within 32 characters. Call that name `<agent>` below. Any step below that blocks or fails goes to Needs you with the workspace label, and the dispatch goes no further.

1. Create a workspace for the run: `herdr workspace create --cwd <project root> --label <project label>-<issue number> --no-focus`. Take the tab ID and root pane ID from the reply.
2. Record the run at once, before any step that can block: `task <uuid> annotate "run: <workspace label>"`. Then run `herdr tab rename <tab_id> orchestrator`.
3. Read the `orchestrator` row of the seat table that the `Dispatch default` ruling names (`gauntlet.md`, or `gauntlet-<name>.md`). Start the orchestrator with `herdr agent start <agent> --kind <Kind> --pane <pane_id> -- <Args>`, taking Kind and Args from that row. A start failure goes to Needs you.
4. The gauntlet skill is invoked by the user, never by a model, so load it first: `herdr agent prompt <agent> "/gauntlet"` for a claude orchestrator, or `"$gauntlet"` for a codex one, then `herdr agent wait <agent> --timeout 120000`. It settles on `idle`, `done` or `blocked`; any state but `idle` or `done` is a failure.
5. Send the kickoff through `gauntlet/send.sh <agent>`, with the message on standard input. The first line is `AGENT MESSAGE (cto, not a human).`, which `send.sh` requires. The message then gives the issue URL, `Seat table: <ruling>`, `Reviewers: <ruling>`, "Report decisions and completion in chat to the dispatcher, the agent named cto", this rule (never add a Co-Authored-By trailer or a tool attribution line to commits, PR bodies, or issue or review comments), and "Begin your response with exactly: RECEIVED <token>" with a token that is new for this send. `send.sh` prints `SEND BLOCKED <agent> <reason>` on failure.
6. For a claude orchestrator, send the goal with `gauntlet/goal.sh <agent>`, the condition on standard input, using the example condition of the gauntlet skill's Session goal section. `goal.sh` prints `GOAL BLOCKED <agent>` on failure.

## Wiki writes

Write a wiki ruling with `llm-wiki rule --vault <vault root> --on <page> "<text>"`, or edit the page directly, in the vault the vault map names for the project. Before any edit, read the target page now and confirm the claim still holds against today's vault. A stale proposal never applies. Never run ingest. Applying a proposal imported from the previous tracker is not part of this skill.

## Brief

Plain English: no metaphors and no filler. Every brief is a full picture, never a delta. Group by project under four fixed sections:

- Needs you: Not checked, then blocked tabs, then every judgment call (every todo tagged `+ask`) in one batch. Number the calls so the user can answer them in one reply. For each call give the todo, the question and the suggested answer with its confidence.
- Done: todos closed this sweep with their proof, and merged pull requests in the window.
- In progress: one line per tab or session: state, age, and the session label. Idle is always stated with an age. An age of `null` is stated as unknown.
- Queued: ready and not ready todos with the next step, ranked.

Pinned tabs appear in the brief and are never questioned.
