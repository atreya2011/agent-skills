---
name: herdr-orchestrator
description: Survey all Claude and Codex tabs across herdr workspaces, report activity and blockers, locate tabs, or resume dead sessions. Use only when the user asks for an all-agent herdr overview, search, or recovery.
---

# herdr orchestrator

## Local file

`~/.agents/local/herdr-orchestrator.md` holds the user's org chart, per-seat launch
commands, and standing rules. Read it when it exists and apply it on top of the
procedure below; without it, run the procedure alone.

You run in the orchestrator's herdr tab. First run `herdr agent list`. If `HERDR_ENV` ≠ `1`,
say this task needs a herdr-managed tab and stop.

You survey and coordinate every agent tab — claude AND codex — and help the user
resume work. For raw tab/workspace primitives, defer to the base `herdr` skill;
this skill is the orchestration + resume layer on top. Written against herdr 0.7.5;
command syntax re-checked against 0.9.0.

## What herdr exposes

`herdr agent list` returns JSON, one object per detected agent; `tab_id` names its tab:

- `agent` — `claude`, `codex`, or another detected agent kind
- `agent_status` — `idle` `working` `blocked` `done` `unknown`
- `name` — the agent's unique name when it has one
- `cwd` — the tab's project directory
- `tab_id` — e.g. `wC:t3`; what `herdr tab` commands take
- `pane_id` — e.g. `wC:pQ`; needed only where a herdr command requires a pane ID
- `terminal_title_stripped` — the tab's own title; usually says what it's doing
- `workspace_id`, `focused`

`herdr agent` commands take a target: the agent's `name`, or its `pane_id` from
`agent list` when it has no name. `herdr tab list` covers every tab including
plain shells, with `label`, `number`, `agent_status` and `tab_id`;
`herdr tab get <tab_id>` returns one tab. `herdr workspace list` maps
`number` ↔ `workspace_id` ↔ `label` ↔ `agent_status`.

IDs (`wC`, `wC:t3`, `wC:pQ`) are opaque handles for the current live session.
Closed IDs retire permanently — a stale ID fails loudly instead of hitting a
neighbor. herdr injects `$HERDR_PANE_ID`, `$HERDR_TAB_ID`, `$HERDR_WORKSPACE_ID`
into every tab, so identify yourself from those instead of guessing.

Status meaning:

- `blocked` — an approval or question UI is waiting on the user. Highest urgency.
- `done` — finished in the background and nobody has SEEN it yet. Focusing the tab
  (or an agent focus command) marks it seen; CLI reads leave it unseen.
- `working` — actively running. Ask first before sending input.
- `idle` — ready for input and already seen.
- `unknown` — an agent may be present but herdr can't classify it (or it's a plain
  shell). Read the tab before concluding anything — `unknown` proves nothing.

## Core loop

1. Survey fresh every task, re-reading IDs from list output:

   ```bash
   herdr agent list
   herdr workspace list
   herdr tab list         # when plain shells matter too
   ```

2. Report one row per agent tab, ranked `blocked > done > working > idle`:

   ```
   ws# | tab_id | name | agent | status | cwd(basename) | what it's doing
   ```

   Fill the last column from `terminal_title_stripped`; when the title is too thin,
   drill down:

   ```bash
   herdr agent read <target> --source recent --lines 40
   ```

3. Lead with tabs that need the user now (`blocked`, then `done`). Always name the
   `agent` (claude vs codex) in every row.

## Find a tab

- By project — filter agent list where `cwd` contains the term.
- By agent — filter `agent == "claude"` or `agent == "codex"`.
- By activity — match on `terminal_title_stripped` first, then
  `herdr agent read <target> --source recent-unwrapped --lines 60`
  for clean, soft-wrap-joined text.

## Act on a LIVE tab

Read-only by default. Ask before sending input to a `working` or `blocked` tab.

Agent tabs take the agent surface, by name or by the pane ID from `agent list`:

```bash
herdr workspace focus <ws#>                       # bring the user to it
herdr agent prompt <target> "<text>" --wait --timeout 120000
herdr agent wait <target> --until done --timeout 120000
herdr agent send-keys <target> esc                # logical keys, validated
herdr agent read <target> --source recent-unwrapped --lines 120
```

`agent prompt` submits text and Enter atomically and honors the tab's live
bracketed-paste mode; `--wait` returns at the first settled `idle`, `done`, or
`blocked`. A prompt that produces no state change within 5 s returns
`agent_prompt_stalled` — recover by reading the tab:

- Text staged **normal-rendered** in the input box → Enter was swallowed during
  paste processing. Submit it: `herdr agent send-keys <target> enter`.
- Input box **empty** → the TUI was still starting and dropped the text. Re-send
  once the prompt line is rendered (`agent read --source visible`).
- After a manual Enter, confirm the flip with `agent read` or a short
  `agent wait --until working` — an immediate settled-state wait can sample the
  pre-flip `idle` and return early.

`agent wait` returns `agent_not_running` promptly when the target tab closes,
so watchers fail fast.

A tab that hosts a plain shell has no agent target, so its commands require a
pane ID: look it up in `herdr pane list` (the entry whose `tab_id` matches).

```bash
herdr pane run <pane_id> "<command>"              # text + Enter
herdr pane send-text <pane_id> "<text>"           # text, follow with send-keys Enter
herdr pane wait-output <pane_id> --match "<text>" --timeout 30000
herdr pane wait-output <pane_id> --regex "<pattern>" --timeout 30000
```

Scrollback caveat: if raising `--lines` reveals nothing more, the agent is on the
terminal's alternate screen and those rows are unrecoverable. Fallback: ask the
agent to write its full answer to a file and read that.

## Resume a DEAD session (live tab absent)

Sessions belong solely to the agents that created them. Open sessions only in a fresh
tab or workspace, routed by cwd.
Placement rule — decide by the session's cwd:

- If a workspace was created at that cwd (its `identity_cwd`) → open a **new tab** in
  that workspace (keeps same-project work together).
- Otherwise → open a **new workspace** (its own tab).

Match only on each workspace's `identity_cwd` (its creation dir, from
`~/.config/herdr/session.json`) — a tab's live cwd can drift after a `cd` into a
subdir. Normalize both sides (`realpath`, strip trailing slash) and, when
several workspaces share the cwd, pick the lowest-numbered **live** one so the choice
is deterministic. `agent start` requires a pane ID, so take the new tab's root
pane ID from the create response, then start the agent:

```bash
REPO=$(python3 -c 'import os,sys;print(os.path.realpath(sys.argv[1]).rstrip("/"))' "<path>")
# lowest-numbered live workspace whose identity_cwd == REPO; empty => none
WS=$(python3 - "$REPO" <<'PY'
import json,os,sys,subprocess
repo=sys.argv[1]
sess=json.load(open(os.path.expanduser("~/.config/herdr/session.json")))["workspaces"]
live={w["workspace_id"]:w["number"] for w in
      json.loads(subprocess.check_output(["herdr","workspace","list"]))["result"]["workspaces"]}
def norm(p): return os.path.realpath(p).rstrip("/") if p else ""
hits=sorted((w["id"] for w in sess if w["id"] in live and norm(w.get("identity_cwd"))==repo),
            key=lambda i: live[i])
print(hits[0] if hits else "")
PY
)

if [ -n "$WS" ]; then
    P=$(herdr tab create --workspace "$WS" --cwd "$REPO" --no-focus \
        | python3 -c 'import sys,json;print(json.load(sys.stdin)["result"]["root_pane"]["pane_id"])')
else
    P=$(herdr workspace create --cwd "$REPO" --no-focus \
        | python3 -c 'import sys,json;print(json.load(sys.stdin)["result"]["root_pane"]["pane_id"])')
fi
```

Start the agent with a validated `agent start` — it targets the new tab's root pane
by ID, verifies the detected agent kind, and returns only once the agent is ready for
input (30 s default timeout). Names match `[a-z][a-z0-9_-]{0,31}`, stay unique
among live agents, and clear when the occupant exits. Native agent args go after `--`.

- **claude** — history in `~/.claude/projects/<slug>/*.jsonl`, where `<slug>` is the
  cwd with every non-alphanumeric char replaced by `-`.

  ```bash
  herdr agent start <name> --kind claude --pane "$P" -- --continue   # most recent in that cwd
  herdr agent start <name> --kind claude --pane "$P" -- --resume     # picker
  # list candidates: ls -t ~/.claude/projects/<slug>/*.jsonl
  ```

- **codex** — index in `~/.codex/session_index.jsonl` (`id`, `thread_name`,
  `updated_at`). A session's cwd lives inside its log at
  `~/.codex/sessions/YYYY/MM/DD/*.jsonl`.

  ```bash
  herdr agent start <name> --kind codex --pane "$P" -- resume --last   # most recent
  herdr agent start <name> --kind codex --pane "$P" -- resume          # picker
  herdr agent start <name> --kind codex --pane "$P" -- resume <uuid>   # exact session
  ```

Then hand it work: `herdr agent prompt <name> "<task>" --wait`.

## Remote agent over ssh

An ssh tab can host an agent running on another machine. Set the `HERDR_AGENT`
hint on the ssh process so local detection applies that agent kind's screen
patterns; the tab then joins `agent list` like any local agent (verified on
0.7.5: claude and codex over ssh). Open it in a new tab; `pane run`,
`pane wait-output` and the first `agent rename` require the new tab's root pane ID,
so take it from the create response:

```bash
P=$(herdr tab create --workspace "$HERDR_WORKSPACE_ID" --no-focus \
    | python3 -c 'import sys,json;print(json.load(sys.stdin)["result"]["root_pane"]["pane_id"])')
herdr pane run "$P" "HERDR_AGENT=claude ssh -t <host> 'zsh -ilc \"cd <dir> && claude --continue\"'"
herdr pane wait-output "$P" --regex "Welcome|shortcuts" --timeout 60000
herdr agent rename "$P" <name>
```

After the rename the full agent surface works by name — prompt, wait, read,
send-keys — with lifecycle tracking (`idle`/`working`/`done`) and the remote
TUI's title updates flowing back through ssh. Remote facts to apply:

- Launch through an interactive login shell (`zsh -ilc "…"`) so the remote
  `.zprofile` and `.zshrc` load — PATH (homebrew, `~/.local/bin`) and env then
  match a real terminal. A bare ssh command line loads neither.
- An expired-token API error on a turn means the ssh-launched instance resolved
  a stale credential store; run `/login` inside that session once to refresh it.
- Confirm the input box is rendered (`agent read --source visible`) before the
  first prompt; text typed during TUI startup is dropped silently and surfaces
  as `agent_prompt_stalled`.
- Surveys report the LOCAL tab cwd; read the remote directory from the tab
  title or screen.
- Budget generous `--timeout` values — tunnel latency adds to model time.
- `agent start` launches local executables only; start remote agents with
  `pane run` + `agent rename` as above. Codex works identically (verified):
  `HERDR_AGENT=codex ssh -t <host> 'zsh -ilc codex'`.

## Rules

- Re-read IDs from list output before every action; treat them as live-session
  handles. Parse new IDs from JSON responses (`workspace create` →
  `result.workspace/tab/root_pane`, `tab create` → `result.tab/root_pane`).
- Target tabs explicitly — a unique agent name, or the exact ID where a command
  requires one. The UI-focused tab belongs to the user.
- Open sessions only in a fresh tab or workspace, routed by workspace
  `identity_cwd` (normalized, lowest-numbered live match): new tab if one shares the
  cwd, else a new workspace.
- Default read-only. Spawning and sending are actions — confirm first, and ask
  before touching a `working` tab.
- Close only tabs you created (`herdr tab close <tab_id>`); leave everything else
  standing. Leave the server running — `server stop` kills every tab's process.
- Discover syntax by running a command group bare (`herdr tab`, `herdr agent`) —
  bare `herdr` attaches the TUI, and probing a mutating command by omitting args
  executes it (`workspace create` is valid with defaults).
- Alert the user when something needs them:
  `herdr notification show "<title>" --body "<text>" --sound done|request`.
- Distinguish claude vs codex in every report using the `agent` field.
- One-glance summaries. Lead with who needs the user now.
- CLI server errors arrive as JSON on stderr with exit 1; syntax errors exit 2.
