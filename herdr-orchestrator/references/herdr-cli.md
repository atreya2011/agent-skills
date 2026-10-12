# What herdr exposes

Written against herdr 0.7.5; command syntax re-checked against 0.9.0.

## Lists and targets

`herdr agent list`, `herdr workspace list`, and `herdr tab list [--workspace <id>]` survey live agents, workspaces and tabs.

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

## IDs

IDs (`wC`, `wC:t3`, `wC:pQ`) are opaque handles for the current live session.
Closed IDs retire permanently — a stale ID fails loudly instead of hitting a
neighbor. herdr injects `$HERDR_PANE_ID`, `$HERDR_TAB_ID`, `$HERDR_WORKSPACE_ID`
into every tab, so identify yourself from those instead of guessing.

Parse new IDs from JSON responses (`workspace create` →
`result.workspace/tab/root_pane`, `tab create` → `result.tab/root_pane`).

## Status meaning

- `blocked` — an approval or question UI is waiting on the user. Highest urgency.
- `done` — finished in the background and nobody has SEEN it yet. Focusing the tab
  (or an agent focus command) marks it seen; CLI reads leave it unseen.
- `working` — actively running. Follow [herdr-chat §Steps](../../herdr-chat/SKILL.md#steps) before sending input to a busy tab.
- `idle` — ready for input and already seen.
- `unknown` — an agent may be present but herdr can't classify it (or it's a plain
  shell). Read the tab before concluding anything — `unknown` proves nothing.

## Syntax and errors

Discover syntax by running a command group bare (`herdr tab`, `herdr agent`) —
bare `herdr` attaches the TUI, and probing a mutating command by omitting args
executes it (`workspace create` is valid with defaults).

CLI server errors arrive as JSON on stderr with exit 1; syntax errors exit 2.

## Agent commands

Read this section before prompting, waiting, sending logical keys or reading an agent. Resolve `<target>` under Lists and targets. Message format, busy-tab handling and lost replies belong to [herdr-chat](../../herdr-chat/SKILL.md).

```bash
herdr workspace focus <ws#>
herdr agent prompt <target> "<text>" --wait --timeout <milliseconds>
herdr agent wait <target> [--until <state>] --timeout <milliseconds>
herdr agent send-keys <target> <key>
herdr agent read <target> --source <source> --lines <count>
```

Prompting uses `--wait` to await the reply; omit it for local slash commands that never enter working state. Wait without `--until` for a settled state, or name `done`, `working` or `idle` when the calling workflow requires it. Keys include `esc`, `enter` and `ctrl+s`. Read sources include `recent`, `recent-unwrapped` and `visible`; unwrapped joins soft wraps. The calling workflow supplies the timeout and line count.

## Tab lifecycle

Read this section before creating, starting, closing or watching a tab. Re-read live IDs before acting, take a new tab's root pane from the create response, and close only tabs you created. Never stop the server; it kills every tab's process. The calling workflow supplies launch arguments, labels and cleanup gates.

```bash
herdr tab create --workspace <ws> --cwd <repo> --label <label> --no-focus --env REPLY_SIZE_CHECK=off
herdr agent start <name> --kind <kind> --pane <pane_id> -- <args>
herdr tab close <tab_id>
herdr pane wait-output <pane_id> --match "<text>" --timeout <milliseconds>
herdr pane wait-output <pane_id> --regex "<pattern>" --timeout <milliseconds>
```

Keep each agent in its own tab. Set `REPLY_SIZE_CHECK=off` for a Claude session reporting to another agent. Workspace selection and session resumption follow [resume.md](resume.md); remote launches follow [remote-ssh.md](remote-ssh.md).
