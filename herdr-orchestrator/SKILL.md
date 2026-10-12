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
work. This skill owns tab lifecycle and CLI syntax.
[herdr-chat](../herdr-chat/SKILL.md) owns live-tab message format, replies and busy-tab behavior.

## Core loop

Read [references/herdr-cli.md](references/herdr-cli.md) before you interpret list output, pick a target, or look up command syntax.

1. Survey fresh every task using agent and workspace lists; include the tab list
   when plain shells matter too, per [references/herdr-cli.md §Lists and targets](references/herdr-cli.md#lists-and-targets).

2. Report one row per agent tab, ranked `blocked > done > working > idle`:

   ```
   ws# | tab_id | name | agent | status | cwd(basename) | what it's doing
   ```

   Fill the last column from `terminal_title_stripped`; when the title is too thin,
   read 40 lines from `recent`, per [references/herdr-cli.md §Agent commands](references/herdr-cli.md#agent-commands).

3. Keep the summary to one glance. Lead with tabs that need the user now (`blocked`,
   then `done`). Name the `agent` (claude vs codex) in every row.

## Find a tab

- By project — filter agent list where `cwd` contains the term.
- By agent — filter `agent == "claude"` or `agent == "codex"`.
- By activity — match on `terminal_title_stripped` first, then
  read 60 lines from `recent-unwrapped` for clean, soft-wrap-joined text,
  per [references/herdr-cli.md §Agent commands](references/herdr-cli.md#agent-commands).

## Act on a live tab

Read [references/live-tab.md](references/live-tab.md) before acting on a live tab or plain shell. Follow [herdr-chat](../herdr-chat/SKILL.md) before talking to another live agent.

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
- Default read-only. Confirm spawning first. Follow [herdr-chat](../herdr-chat/SKILL.md)
  before sending live-tab messages, including to busy tabs.
- Close only tabs you created; read [references/herdr-cli.md §Tab lifecycle](references/herdr-cli.md#tab-lifecycle) before closing. Leave everything else
  standing. Leave the server running — `server stop` kills every tab's process.
- Alert the user when something needs them:
  `herdr notification show "<title>" --body "<text>" --sound done|request`.
