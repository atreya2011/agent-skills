# Tab cleanup

Clean up idle or done agent tabs in every unpinned workspace. herdr's `done` is the same state as `idle`, not yet seen, so treat the two alike. Never question or close a working or pinned tab, a tab in a pinned workspace, or the tab running this sweep.

Before cleanup and immediately before every cleanup prompt or `herdr tab close`, refresh the live workspace, tab and agent lists. Resolve the sweep's current tab with `herdr pane current --current`, using `.result.pane.tab_id`, rather than inherited `$HERDR_TAB_ID`. If this identity read fails or has no tab ID, skip cleanup and report the failure under Not checked. Check the flow's eligibility against these fresh reads: the target tab is still `idle` or `done`, both tab and workspace are unpinned, and the tab is not the sweep's current tab. A prompt also requires a live agent with nonempty `agent` and `target` attached to that tab. The workspace flow additionally requires the workspace to remain `idle` or `done`. If a read fails or eligibility changes, keep the affected tabs open, report them and do not prompt or close them.

Before prompting any `claude` agent, set aside its input draft with `herdr agent send-keys <target> ctrl+s`; Claude restores it after the message is sent. If this fails, keep the target open, report the failure and send no prompt. If a prompt fails, times out or settles `blocked`, keep the target open and report it; never read an older reply as proof or proceed to closing.

## Workspace orchestrator

Use this flow for each unpinned workspace whose state is `idle` or `done` and which has an eligible tab labeled `orchestrator` with a live agent and nonempty `agent` and `target`. If the orchestrator is ineligible or is only a shell, skip this flow and consider the workspace's other eligible tabs individually. Once an orchestrator prompt has been attempted, its workspace's tabs stay in this flow; never question them individually after a prompt failure, a timeout, a blocked state or any other verdict.

1. Preserve any Claude draft and apply the fresh checks above, then send the orchestrator a chat message: `herdr agent prompt <target> --wait --timeout 120000 "AGENT MESSAGE (cto, not a human). Are your tabs still needed? Reply done only if your work is finished and every tab can close. If you finished any of these todos, name them: <the project's open +cto todos>."`
2. Read the reply with `herdr agent read <target> --source recent-unwrapped --lines 60` and pipe the final message into `bin/cto-snapshot reply`.
3. If the verdict is `act` and the label is `done`, the orchestrator has said done. That is proof: close the todos it named and annotate them with the printed `annotation`, then apply the fresh checks before closing each eligible tab with `herdr tab close <tab_id>`, the orchestrator's last. If any tab remains ineligible, the workspace stays open and is reported.
4. A timeout, a `blocked` state or any other verdict is "not done". Report the workspace with its state and age in Needs you. Close nothing.

## Individual agent tab

Use this flow for every eligible agent tab whose workspace's orchestrator prompt was not attempted in the workspace flow. This includes tabs in workspaces with no orchestrator, tabs whose orchestrator was skipped because it is ineligible or is only a shell, and idle or done tabs in workspaces that are not wholly idle.

1. Preserve any Claude draft and apply the fresh checks above, then send `herdr agent prompt <target> --wait --timeout 120000 "AGENT MESSAGE (cto, not a human). Is this tab still needed? Reply done only if your work is finished and this tab can close. If you finished any of these todos, name them: <the project's open +cto todos>."`
2. Read the reply with `herdr agent read <target> --source recent-unwrapped --lines 60` and pipe the final message into `bin/cto-snapshot reply`.
3. Only a verdict of `act` with the label `done` is proof: close the todos it named and annotate them with the printed `annotation`, then apply the fresh checks before closing that one tab with `herdr tab close <tab_id>`.
4. A timeout, a `blocked` state or any other verdict closes nothing. Report the tab with its state and age in Needs you.

Talk to agents by chat only. Never edit their files.
