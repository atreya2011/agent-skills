# Gauntlet tabs and seat briefs

Use this file for herdr tab and agent spawn, seat brief delivery, receipt, watch, capture, clear, and close. Read shared contracts and role templates from [contracts.md](contracts.md) and each stage's `briefs/` directory. At each run start or resume, read the current [SKILL.md](../SKILL.md) and this file.

Seat brief delivery goes through `gauntlet/send.sh`: empty the composer with Escape only when the seat is idle or done (skipped for Codex, which exits on Escape when its composer is empty, and for a working or blocked seat, where Escape interrupts the turn or dismisses the dialog), paste, verify content, submit once, and confirm working. Claude and Cursor can stage and need one Enter; Codex submits directly. On failure print `SEND BLOCKED <agent> <reason>`. Waiting and capture go through `gauntlet/watch.sh`; `CAPTURE BLOCKED` stays for read failures.

## Spawn tabs

Create a tab in the current workspace. Keep each tab unsplit. Label every tab by seat. Name each agent `<seat>-<issue>`, such as `implementer-5`, because herdr agent names are unique per server; that name is `<agent>` in every herdr command and script below. Rename the controlling tab `orchestrator`. A pane id is looked up only where a herdr command requires one.

    herdr tab create --workspace <ws> --cwd <repo> --label <seat> --no-focus
    -> result.root_pane.pane_id
    herdr agent start <agent> --kind <Kind> --pane <pane_id> -- <Args>

Kind and Args come from the seat's row in the seat table; see the Local file section of [SKILL.md](../SKILL.md). `herdr agent start --kind claude` launches the canonical `claude` executable only.

Use a read-only allowlist where supported.

## Prepare seat briefs

Before sending any implementer, reviewer, or fix brief, the orchestrator MUST render the selected template. Rendering uses the gauntlet skill's `render-brief.sh` from its skill root. Send the output as the message. Fill every field from current command output. Render the complete block from scratch each time; treat any earlier rendered seat brief as stale, not a source.
A nonzero render exit MUST be reported as `RENDER BLOCKED <template>` and MUST stop the send.

Select one implementation issue for the run. Record its branch, base SHA, hard cap, and current changed-line count. Keep every seat brief, review range, fix wave, and delivery action scoped to that issue.
The seat brief is the tab's scope contract; deliver it in the tab's prompt. The repo's drift-guard scope serves the staged spec contract alone.

Confirm the agent has settled before prompting: `herdr tab list` shows the seat's tab with its `agent_status`, or use `herdr agent wait <agent> --until idle`. Treat a raw shell prompt as not yet a valid target.

Handle launch blockers before sending a seat brief:

- On startup failure, relaunch the agent once.
- On the Codex updater menu, choose the install-free option with Down then Enter.
- Re-run `herdr tab list` and confirm the seat's tab is idle. A second startup failure reports `SEAT BLOCKED <seat>` and stops the cycle for the user.

Before sending a rendered seat brief, verify its receipt line and required scope.

## Send receipts

Deliver the complete rendered seat brief as the prompt message. Put one of these receipt lines at the top, filling in a unique token per send:

    AGENT MESSAGE (orchestrator, not a human). Fix this task. Begin your response with exactly: RECEIVED <token>
    AGENT MESSAGE (orchestrator, not a human). Write your review. Begin your response with exactly: RECEIVED <token>

The `AGENT MESSAGE` prefix marks the sender as an agent. `send.sh` blocks a message whose first line lacks it.

Use the `Fix` line for the implementer, for both its implementer brief and every fix brief, and the `Write your review` line for reviewers. Send that line and the full seat brief together through `send.sh`.

Confirm delivery from the exact target receipt: `herdr agent read <agent> --source visible` MUST show the exact `RECEIVED <token>`; herdr refuses a longer read while the seat works. Composer contents remain staged until submission. Resolve an updater menu or startup failure and retry once. After a second delivery failure, or a rejected paste, report `SEND BLOCKED <agent>` and stop.

Report the implementer or the reviewer group as `running` after every target has its receipt and a confirmed working status. Validate the result directly when an agent reaches completion first.

## Session goal

Applies to Claude Code seats only, per the Session goal section of [SKILL.md](../SKILL.md). `send.sh` cannot carry a slash command: it requires the first line to start with `AGENT MESSAGE`. After the seat acknowledges its kickoff, send the goal through `gauntlet/goal.sh` with the condition on stdin:

    gauntlet/goal.sh <agent> <<'EOF'
    <condition>
    EOF

`goal.sh` flattens the condition to one line, sends `/goal <condition>`, and exits 0 only when the visible screen shows `/goal active`. An empty or over-4000-character condition, a non-Claude seat, a failed prompt, or no `/goal active` within about 15 seconds prints `GOAL BLOCKED <agent>` and exits 1. On a seat restart, follow the restart step in the Session goal section of [SKILL.md](../SKILL.md).

## Capture results

Capture output through `watch.sh` immediately after the completion wait succeeds. The recent buffer truncates long messages, so capture immediately and isolate the final assistant message, excluding the seat brief, receipt, and earlier tab text. A truncated or missing capture reports `CAPTURE BLOCKED <source>` and stops.

For the implementer, search for the runtime-valued terminal line: `IMPLEMENTER DONE — [0-9]+ commits?`, `IMPLEMENTER PARTIAL — [0-9]+ commits?`, `IMPLEMENTER BLOCKED — [^<]`, or `FIX DONE — [0-9]+ commits?`. Each pattern needs text that the seat brief's template leaves as a placeholder (`<N>`, `<reason>`), so it cannot match the template on the screen. For reviewers, locate the final assistant message from the completed turn. Use the longest matching assistant message.

## Clear a tab

Send the seat's Clear command from the seat table with `herdr agent prompt <agent> "<Clear>"`, without `--wait`. The command is local, so the seat never works: `--wait` returns `agent_prompt_stalled` for a claude seat, and `--until idle` times out for a codex seat, which ends `done`.

Read `herdr agent read <agent> --source visible`. If the composer still holds the command, send `herdr agent send-keys <agent> Enter` once. After `/new` a codex seat first asks "Where should the new conversation run?"; send Enter the same way to keep "Current checkout". Then read the screen every second for about 15 seconds until the earlier turns are gone and the composer is empty; otherwise stop with `SEAT BLOCKED <seat>`.

After `/clear` a claude seat shows its header, the `/clear` line and an empty composer, and `herdr tab list` reports its tab idle. After `/new` a codex seat shows its header, a welcome line and the empty composer `› Ask Codex to do anything`, and `herdr tab list` reports its tab `done`. For a cursor seat, confirm live before use.

Send the fresh rendered seat brief only after the composer is empty, applying the send receipt protocol above. The orchestrator never clears itself during a run.

## Watch completion

For reviewers, `watch.sh` owns the settled-state wait. Treat `blocked` as a question to answer, not a failure.

For the implementer, confirm completion with a runtime-valued regex requiring:

    herdr pane wait-output <pane_id> --regex "<done regex>" --timeout <milliseconds>

Require digits, as in `FIX DONE — [0-9]+ commits?`: a pattern for `IMPLEMENTER DONE` alone matches the template text in the seat brief. Run `herdr agent wait` alongside it for blocked or idle states; treat timeouts as outer bounds.

## Close tabs

Close a spawned tab with `herdr tab close <tab_id>`, taking the id from `herdr tab list`, only after its output is captured, it passes [gauntlet-review/SKILL.md](../../gauntlet-review/SKILL.md) §Validate results, and an independent runtime check proves zero owned resources. The implementer tab stays open until delivery. Keep blocked tabs open for cleanup, and list every spawned tab label and closure state in the final report.

## Decisions

`IMPLEMENTER PARTIAL` is a settled state for `watch.sh`, like done. Capture it, verify the committed items, return an entry with a missing field to the seat unfiled, and file a complete entry in the inbox file. Send the answer through `send.sh` as [contracts.md](contracts.md) §Inbox says.
