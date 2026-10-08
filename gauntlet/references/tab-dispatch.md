# Gauntlet tab dispatch

Use this file for herdr 0.7.5 tab and agent spawn, brief delivery, receipt, watch, capture, reset, and close. Read shared contracts and role templates from [contracts.md](contracts.md) and each stage's `briefs/` directory. At each run start or resume, read the current [SKILL.md](../SKILL.md) and this file.

Brief delivery goes through `gauntlet/send.sh`: clear the composer with Escape only when the seat is idle or done (skipped for Codex, which exits on Escape when its composer is empty, and for a working or blocked seat, where Escape interrupts the turn or dismisses the dialog), paste, verify content, submit once, and confirm working. Claude and Cursor can stage and need one Enter; Codex submits directly. On failure print `SEND BLOCKED <target> <reason>`. Waiting and capture go through `gauntlet/watch.sh`; `CAPTURE BLOCKED` stays for read failures.

## Spawn tabs

Create a tab in the current workspace. Keep each tab unsplit. Label every tab by seat. Rename the controlling tab `orchestrator`. A pane id is looked up only where a herdr command requires one.

    herdr tab create --workspace <ws> --cwd <repo> --label <seat> --no-focus
    -> result.root_pane.pane_id
    herdr agent start <seat> --kind <Kind> --pane <pane_id> -- <Args>

Kind and Args come from the seat's row in the seat table; see the Seats section of [SKILL.md](../SKILL.md). `herdr agent start --kind claude` launches the canonical `claude` executable only. Launch a wrapper with `herdr pane run <pane_id> "<wrapper command>"` and automatic agent detection, then address it by seat name afterward.

Use a read-only allowlist where supported.

## Prepare briefs

Before every implementer, reviewer, or fix dispatch, the orchestrator MUST render the selected template. Rendering uses the gauntlet skill's `render-brief.sh` from its skill root. Send the output as the dispatch message. Fill every field from current command output. Render the complete block from scratch each time; treat any earlier rendered brief as stale, not a source.
A nonzero render exit MUST be reported as `RENDER BLOCKED <template>` and MUST stop the dispatch.

Select one implementation issue for the run. Record its branch, pinned PR base, hard cap, and current changed-line count. Keep every brief, review range, fix wave, and delivery action scoped to that issue.
The brief is the tab's scope contract; deliver it in the tab's prompt. The repo's drift-guard scope serves the staged spec contract alone.

Confirm the agent has settled before prompting: `herdr agent get <seat>` or `herdr agent wait <seat> --until idle`. Treat a raw shell prompt as not yet a valid target.

Handle launch blockers before dispatch:

- On startup failure, relaunch the agent once.
- On the Codex updater menu, choose the install-free option with Down then Enter.
- Re-run `herdr agent get <seat>` and confirm the idle state. A second startup failure reports `SEAT BLOCKED <seat>` and stops the cycle for the user.

Before dispatching a rendered brief, verify its receipt line and required scope.

## Dispatch receipts

Deliver the complete rendered brief as the prompt message. Put one of these receipt lines at the top, filling in a unique token per dispatch:

    AGENT MESSAGE (orchestrator, not a human). Fix this task. Begin your response with exactly: RECEIVED <token>
    AGENT MESSAGE (orchestrator, not a human). Write your review. Begin your response with exactly: RECEIVED <token>

The `AGENT MESSAGE` prefix marks the sender as an agent. `send.sh` blocks a message whose first line lacks it.

Use the `Fix` line for the implementer, for both its implementer brief and every fix brief, and the `Write your review` line for reviewers. Send that line and the full brief together through `send.sh`.

Confirm delivery from the exact target receipt: `herdr agent read <seat>` MUST echo the exact `RECEIVED <token>`. Composer contents remain staged until submission. Resolve an updater menu or startup failure and retry once. After a second delivery failure, or a rejected paste, report `DISPATCH BLOCKED <seat>` and stop.

Report the implementer or the reviewer group as `running` after every target has its receipt and a confirmed working status. Validate the result directly when an agent reaches completion first.

## Session goal

Applies to Claude Code seats only, per the Session goal section of [SKILL.md](../SKILL.md). `send.sh` cannot carry a slash command: it requires the first line to start with `AGENT MESSAGE`. After the seat acknowledges its kickoff, send the goal through `gauntlet/goal.sh` with the condition on stdin:

    gauntlet/goal.sh <seat> <<'EOF'
    <condition>
    EOF

`goal.sh` flattens the condition to one line, sends `/goal <condition>`, and exits 0 only when the visible screen shows `/goal active`. An empty or over-4000-character condition, a non-Claude seat, a failed prompt, or no `/goal active` within about 15 seconds prints `GOAL BLOCKED <seat>` and exits 1. On a seat restart, follow the restart step in the Session goal section of [SKILL.md](../SKILL.md).

## Capture results

Capture output through `watch.sh` immediately after the completion wait succeeds. The recent buffer truncates long messages, so capture immediately and isolate the final assistant message, excluding the brief, receipt, and earlier tab text. A truncated or missing capture reports `CAPTURE BLOCKED <source>` and stops.

For the implementer, search for the runtime-valued terminal signal (`IMPLEMENTER DONE — [0-9]+` or `FIX DONE — [0-9]+`). For reviewers, locate the final assistant message from the completed turn. Use the longest matching assistant message.

## Reset a tab

Send the seat's Clear command from the seat table with `herdr agent prompt <seat> "<Clear>" --wait --until idle`. Confirm the empty composer with `herdr agent read <seat> --source visible`. If the command remains staged, send `herdr agent send-keys <seat> Enter` once and check again. Send the fresh rendered brief only after the composer is empty, applying the dispatch receipt protocol above. The orchestrator never resets itself during a run.

## Watch completion

For reviewers, `watch.sh` owns the settled-state wait. Treat `blocked` as a question to answer, not a failure.

For the implementer, confirm completion with a runtime-valued regex requiring:

    herdr pane wait-output <pane_id> --regex "<done regex>" --timeout <milliseconds>

Require digits such as `FIX DONE — [0-9]+`. Run `herdr agent wait` alongside it for blocked or idle states; treat timeouts as outer bounds.

## Close tabs

Close a spawned tab only after its output is captured, it passes [gauntlet-review/SKILL.md](../../gauntlet-review/SKILL.md) §Validate results, and an independent runtime check proves zero owned resources. The implementer tab closes only at delivery. Keep blocked tabs open for cleanup, and list every spawned tab label and closure state in the final report.

## Decisions

`IMPLEMENTER PARTIAL` is a settled state for `watch.sh`, like done. Capture it, verify the committed items, return an entry with a missing field to the seat unfiled, and file a complete entry in the inbox file. Send the `DECISION <id>` brief through `send.sh` to the same tab without its Clear command; the seat needs its context to resume.
