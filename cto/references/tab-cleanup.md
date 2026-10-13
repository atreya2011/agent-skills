# Tab cleanup


Do this for each workspace that is not pinned and whose state is `idle` or `done`. herdr's `done` is the same state as `idle`, not yet seen, so treat the two alike. A workspace's orchestrator is its tab labeled `orchestrator`. A workspace with no such tab is reported with its state and left alone. A pinned tab or workspace is never questioned, and a pinned tab is never closed, even inside an unpinned workspace.

1. If the orchestrator is a `claude` agent, set aside any draft the user left in its input box with `herdr agent send-keys <target> ctrl+s`; Claude puts the draft back after the message is sent. Then send the orchestrator a chat message: `herdr agent prompt <target> --wait --timeout 120000 "AGENT MESSAGE (cto, not a human). Are your tabs still needed? Reply done only if your work is finished and every tab can close. If you finished any of these todos, name them: <the project's open +cto todos>."`
2. Read the reply with `herdr agent read <target> --source recent-unwrapped --lines 60` and pipe the final message into `bin/cto-snapshot reply`.
3. If the verdict is `act` and the label is `done`, the orchestrator has said done. That is proof: close the todos it named and annotate them with the printed `annotation`, then close each tab of the workspace that is `idle` or `done` and not pinned, with `herdr tab close <tab_id>`, the orchestrator's last. If a pinned tab remains, the workspace stays open and is reported.
4. A timeout, a `blocked` state or any other verdict is "not done". Report the workspace with its state and age in Needs you. Close nothing.

Talk to orchestrators by chat only. Never edit their files.
