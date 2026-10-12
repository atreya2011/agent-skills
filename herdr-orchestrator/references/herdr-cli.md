# What herdr exposes

Written against herdr 0.7.5; command syntax re-checked against 0.9.0.

## Lists and targets

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
- `working` — actively running. Ask first before sending input.
- `idle` — ready for input and already seen.
- `unknown` — an agent may be present but herdr can't classify it (or it's a plain
  shell). Read the tab before concluding anything — `unknown` proves nothing.

## Syntax and errors

Discover syntax by running a command group bare (`herdr tab`, `herdr agent`) —
bare `herdr` attaches the TUI, and probing a mutating command by omitting args
executes it (`workspace create` is valid with defaults).

CLI server errors arrive as JSON on stderr with exit 1; syntax errors exit 2.
