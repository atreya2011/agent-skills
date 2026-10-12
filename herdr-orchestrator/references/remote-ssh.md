# Remote agent over ssh

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
