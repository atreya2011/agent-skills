# Install

Build and configure the CTO tab, then install one crontab entry that clears context before every sweep. Check the setup once by hand.


1. Build the tool: `go build -o ../bin/cto-snapshot .` from this skill's `snapshot` directory.
2. Store the API key once. On Linux: `secret-tool store --label=typesafe service typesafe username api`. On macOS: `security add-generic-password -s typesafe -a api -w`. Each command prompts for the key.
3. Create the local file from [local-file.md](local-file.md), including the `Dispatch default` ruling.
4. Create the CTO tab: `herdr workspace create --cwd <directory> --label cto --no-focus`. Take the root pane ID from the reply, run the launch command there with `herdr pane run <pane> "<launch command>"`, then `herdr agent rename <pane> cto`.
5. Add the crontab entry below, then work through [checklist.md](checklist.md).

The crontab addresses the tab by pane ID because `/clear` starts a new agent session and drops its agent name. `PATH` must include the directory of `herdr`. The line runs at the sweep interval during waking hours. It waits up to 600000 ms for the tab to settle, then sends only when it is `idle` or `done`. A `blocked` tab gets nothing, because a blocked tab is already waiting on the user. Before the sweep prompt, the line sends `/clear` and Enter once, because the command can stay staged in the composer; on an empty composer the Enter does nothing. It then waits five seconds and restores the `cto` agent name, ignoring a rename failure so the sweep still runs. Before each message the line presses `ctrl+s`, which sets aside any draft the user left in the input box; Claude puts the draft back after the message is sent.

```
*/<interval in minutes> <first hour>-<last hour> * * * herdr agent wait <pane> --timeout 600000; herdr agent get <pane> | grep -Eq '"agent_status":"(idle|done)"' && herdr agent send-keys <pane> ctrl+s && herdr agent prompt <pane> "/clear" && herdr agent send-keys <pane> Enter && sleep 5 && (herdr agent rename <pane> cto || true) && herdr agent send-keys <pane> ctrl+s && herdr agent prompt <pane> "AGENT MESSAGE (cron). Run the cto skill: sweep."
```
