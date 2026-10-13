# Manual checklist

Cron, tab setup and the notification need a live herdr, so check them once by hand after install. Tick each item.

- The build: `bin/cto-snapshot --local-file ~/.agents/local/cto.md` prints a JSON document and exits 0. Every source the machine has appears; anything missing appears in `unreadable`.
- The key: `echo "All merged. The tabs can be closed." | bin/cto-snapshot reply` prints a judgment whose reason is not "no API key". Run it from the CTO tab, because that is where the sweeps run.
- The tab: `herdr agent get cto` returns the CTO agent, and `herdr agent prompt cto "AGENT MESSAGE (user). Run the cto skill: sweep."` produces a brief.
- Cron fires: after the crontab is installed, the next scheduled time starts a sweep. `herdr agent read cto --source recent-unwrapped --lines 60` shows the cron prompt and the brief.
- A blocked CTO tab gets nothing: block the tab on a question, wait for a scheduled time, and confirm no prompt arrived.
- Each sweep: at every scheduled time the tab clears its context (a staged `/clear` is submitted by the Enter), restores the `cto` agent name, then runs a full sweep. The brief that follows uses current sources.
- The notification: start a tab that asks a question and leave it blocked. The next sweep raises one notification naming it, and the brief lists it under Needs you.
- Tab cleanup: follow [tab-cleanup.md](tab-cleanup.md) with live eligibility and current-pane identity checks before each prompt and close. An eligible agent orchestrator uses the workspace reply gate; otherwise eligible tabs are asked individually. Claude drafts are set aside successfully before prompting, and a failed, timed-out or blocked prompt closes nothing and never falls back from the workspace flow. Only an explicit done passing the gate permits closing. Working or pinned tabs, tabs in pinned workspaces and the sweep's current tab are never asked or closed.
- Dispatch: a ready todo gets a workspace, an orchestrator, a kickoff, a goal and a `run:` annotation, and the next sweep does not dispatch it again. A todo that is not ready is listed in Queued instead.
