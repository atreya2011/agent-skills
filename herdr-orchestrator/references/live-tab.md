# Act on a live tab

## Agent tabs

Read [herdr-cli.md](herdr-cli.md) §Lists and targets and §Agent commands before targeting, focusing, prompting, waiting, sending keys or reading. Use a 120000 ms timeout for prompts and done-state waits, the logical key `esc` for Escape, and 120 lines from `recent-unwrapped` for reads. Follow [herdr-chat](../../herdr-chat/SKILL.md) for live-tab messages, busy tabs and reply handling.

`agent prompt` submits text and Enter atomically and honors the tab's live
bracketed-paste mode; `--wait` returns at the first settled `idle`, `done`, or
`blocked`. A prompt that produces no state change within 5 s returns
`agent_prompt_stalled` — recover by reading the tab:

- Text staged **normal-rendered** in the input box → Enter was swallowed during
  paste processing. Submit it with the logical key `enter`, using Agent commands in [herdr-cli.md](herdr-cli.md).
- Input box **empty** → the TUI was still starting and dropped the text. Re-send
  once a read from `visible` shows the rendered prompt line.
- After a manual Enter, confirm the flip with `agent read` or a short
  working-state wait — an immediate settled-state wait can sample the
  pre-flip `idle` and return early.

`agent wait` returns `agent_not_running` promptly when the target tab closes,
so watchers fail fast.

## Plain-shell tabs

A tab that hosts a plain shell has no agent target, so its commands require a
pane ID: look it up in `herdr pane list` (the entry whose `tab_id` matches).

```bash
herdr pane run <pane_id> "<command>"              # text + Enter
herdr pane send-text <pane_id> "<text>"           # text, follow with send-keys Enter
```

For text or regex output waits, read [herdr-cli.md](herdr-cli.md) §Tab lifecycle before watching the pane; use a 30000 ms timeout.

## Scrollback

If raising `--lines` reveals nothing more, the agent is on the
terminal's alternate screen and those rows are unrecoverable. For lost replies, follow [herdr-chat §Steps](../../herdr-chat/SKILL.md#steps).
