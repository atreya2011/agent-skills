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

Survey and coordinate every agent tab, claude and codex, and help the user resume
work. This skill is the orchestration and resume layer; defer raw tab and workspace
primitives to the base `herdr` skill.

## Core loop

Read [references/herdr-cli.md](references/herdr-cli.md) before you interpret list output, pick a target, or look up command syntax.

1. Survey fresh every task:

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
   drill down with `herdr agent read <target> --source recent --lines 40`.

3. Keep the summary to one glance. Lead with tabs that need the user now (`blocked`,
   then `done`). Name the `agent` (claude vs codex) in every row.

## Find a tab

- By project — filter agent list where `cwd` contains the term.
- By agent — filter `agent == "claude"` or `agent == "codex"`.
- By activity — match on `terminal_title_stripped` first, then
  `herdr agent read <target> --source recent-unwrapped --lines 60`
  for clean, soft-wrap-joined text.

## Act on a live tab

Read [references/live-tab.md](references/live-tab.md) before you focus, prompt, wait on, send keys to or read a live tab, or run a command in a plain shell.

## Resume a dead session

Read [references/resume.md](references/resume.md) before you reopen a session whose live tab is absent.

## Remote agent over ssh

Read [references/remote-ssh.md](references/remote-ssh.md) before you open a tab for an agent on another machine.

## Rules

- Re-read IDs from list output before every action; treat them as live-session handles.
- Target tabs explicitly — a unique agent name, or the exact ID where a command
  requires one. The UI-focused tab belongs to the user.
- Sessions belong solely to the agents that created them. Open a session only in a
  fresh tab or workspace, routed by cwd.
- Default read-only. Spawning and sending are actions — confirm first, and ask
  before sending input to a `working` or `blocked` tab.
- Close only tabs you created (`herdr tab close <tab_id>`); leave everything else
  standing. Leave the server running — `server stop` kills every tab's process.
- Alert the user when something needs them:
  `herdr notification show "<title>" --body "<text>" --sound done|request`.
