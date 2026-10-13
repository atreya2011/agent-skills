# Tab cleanup


Clean up idle or done agent tabs in every unpinned workspace. herdr's `done` is the same state as `idle`, not yet seen, so treat the two alike. Never question or close a pinned tab or workspace. Identify the CTO's own tab by `$HERDR_TAB_ID`; never question or close it in either flow.

## Workspace orchestrator

Use this flow for each unpinned workspace whose state is `idle` or `done` and which has a tab labeled `orchestrator`. If that orchestrator is pinned or is the CTO's own tab, skip this flow and consider the workspace's other eligible tabs individually. Once an orchestrator has been asked, its workspace's tabs stay in this flow; never question them individually after a timeout, a blocked state or any other verdict.

1. If the orchestrator is a `claude` agent, set aside any draft the user left in its input box with `herdr agent send-keys <target> ctrl+s`; Claude puts the draft back after the message is sent. Then send the orchestrator a chat message: `herdr agent prompt <target> --wait --timeout 120000 "AGENT MESSAGE (cto, not a human). Are your tabs still needed? Reply done only if your work is finished and every tab can close. If you finished any of these todos, name them: <the project's open +cto todos>."`
2. Read the reply with `herdr agent read <target> --source recent-unwrapped --lines 60` and pipe the final message into `bin/cto-snapshot reply`.
3. If the verdict is `act` and the label is `done`, the orchestrator has said done. That is proof: close the todos it named and annotate them with the printed `annotation`, then close each tab of the workspace that is `idle` or `done`, not pinned and not the CTO's own tab, with `herdr tab close <tab_id>`, the orchestrator's last. If a pinned tab or the CTO's own tab remains, the workspace stays open and is reported.
4. A timeout, a `blocked` state or any other verdict is "not done". Report the workspace with its state and age in Needs you. Close nothing.

## Individual agent tab

Use this flow for every agent tab whose state is `idle` or `done`, which is not pinned, whose workspace is not pinned, which is not the CTO's own tab and whose workspace's orchestrator was not asked in the workspace flow. This includes tabs in workspaces with no orchestrator, tabs whose orchestrator was skipped because it is pinned or is the CTO's own tab, and idle or done tabs in workspaces that are not wholly idle.

1. If the tab's agent is `claude`, preserve its input draft with `herdr agent send-keys <target> ctrl+s` as above. Then send `herdr agent prompt <target> --wait --timeout 120000 "AGENT MESSAGE (cto, not a human). Is this tab still needed? Reply done only if your work is finished and this tab can close. If you finished any of these todos, name them: <the project's open +cto todos>."`
2. Read the reply with `herdr agent read <target> --source recent-unwrapped --lines 60` and pipe the final message into `bin/cto-snapshot reply`.
3. Only a verdict of `act` with the label `done` is proof: close the todos it named and annotate them with the printed `annotation`, then close that one tab with `herdr tab close <tab_id>`.
4. A timeout, a `blocked` state or any other verdict closes nothing. Report the tab with its state and age in Needs you.

Talk to agents by chat only. Never edit their files.
