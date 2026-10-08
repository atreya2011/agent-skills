# Engine: tab

Run the implementer seat from the seat table inside a herdr tab.

## Send

Follow
[../../gauntlet/references/tab-dispatch.md](../../gauntlet/references/tab-dispatch.md)
for the full spawn-through-close lifecycle:

1. Spawn a tab labeled `implementer`; start the implementer seat in it.
2. Confirm the tab sits at a settled, idle state before prompting it.
3. Deliver the full rendered implementer brief as the chat prompt, with its
   fix-verb receipt line at the top of the message.
4. Confirm the echoed receipt token, then watch for `done`, `idle`, or
   `blocked`.
5. Capture the final message with `agent read --source recent-unwrapped` immediately on completion. A truncated or missing capture reports `CAPTURE BLOCKED <source>` and stops.
6. Validate the schema and terminal line from
   [briefs/implementer.md](../briefs/implementer.md) before accepting the
   result.
7. Close the tab per
   [tab-dispatch.md](../../gauntlet/references/tab-dispatch.md) §Close tabs; the
   review stage clears it before each fix brief.

## On a blocked start

Read the tab and resolve a startup failure or updater prompt, then retry the
prompt once. Report `SEAT BLOCKED <seat>` and stop after a second startup
failure.
