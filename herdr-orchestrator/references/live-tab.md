# Act on a live tab

## Agent tabs

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

## Plain-shell tabs

A tab that hosts a plain shell has no agent target, so its commands require a
pane ID: look it up in `herdr pane list` (the entry whose `tab_id` matches).

```bash
herdr pane run <pane_id> "<command>"              # text + Enter
herdr pane send-text <pane_id> "<text>"           # text, follow with send-keys Enter
herdr pane wait-output <pane_id> --match "<text>" --timeout 30000
herdr pane wait-output <pane_id> --regex "<pattern>" --timeout 30000
```

## Scrollback

If raising `--lines` reveals nothing more, the agent is on the
terminal's alternate screen and those rows are unrecoverable. Fallback: ask the
agent to write its full answer to a file and read that.
