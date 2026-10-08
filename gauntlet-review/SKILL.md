---
name: gauntlet-review
description: Run executable-check review against a locked HEAD.
---

# Gauntlet review

Read this file, the tab-dispatch reference, the shared contracts, the blind panel, and the current seat briefs before each cycle. Every chosen reviewer MUST be captured as settled; otherwise stop with `SEAT UNCAPTURED <seat>` and block the close.

## Cycle

Lock HEAD and hold the cap. Run every acceptance checkbox on the locked HEAD. Render reviewer briefs and deliver each with `gauntlet/send.sh`. Wait and capture every seat with `gauntlet/watch.sh`. Admit each check with `gauntlet/admit-check.sh` under the finding contract and mark it major or minor. Rule each prose point admit or skip under the simplicity contract and the issue scope; a skipped point is dropped.

Write each ruled item as one line of the cycle's admission list, a scratch file outside the repository, with fields separated by tabs: `major` or `minor`, the path of the file that holds the check, the clause; or `point` or `skip`, the text. A check may span many lines; the script puts each file's content into its output. An acceptance checkbox that fails on the locked HEAD enters the list as a `major` line, with a scratch file that holds the failing command as its check file and the checkbox text as its clause. Then run `gauntlet/close-decision.sh <locked-head>` with the admission list on standard input. The first line it prints is the decision.

On `FIX` the cycle has a major bug. Send every admitted check and point to the implementer in one fix brief: fill `final-accepted-records` with the `- Check:` items and `admitted-points` with the `- Point:` items, or `none`. First send the implementer its Clear command and confirm the empty context per [tab-dispatch.md](../gauntlet/references/tab-dispatch.md) §Clear a tab (idle or done). When the same acceptance checkbox produces a major bug in two consecutive cycles, stop fixing and send `DECISION NEEDED` to the dispatcher about the checkbox itself.

Before cycles 3, 6, 9, and every later multiple of 3, send the implementer a deletion round: one brief that only deletes tautological tests, defensive code, and tests for cases a user can never produce. Then lock the new HEAD and run the cycle.

After the fix wave, perform the narrow anti-overfit diff check. Hardcoded expected values or special-cased inputs reject the wave. Report the committed-range count to the dispatcher after every cycle. Growth past the issue's estimate or the cap stops the wave with `DECISION NEEDED`; the cap is never waived, and the implementer trims first. One deletion-only agent runs only if that trim fails. A failed deletion returns the issue to Tickets.

## Close

The first cycle with no major bug closes the run as DIMINISHING RETURNS, the script's decision when no admitted item is major. Every chosen reviewer is captured and every acceptance checkbox passes on the locked HEAD. That cycle's admitted checks and points are not fixed; hand the script's `## Review notes` list to delivery. Reviewers never declare DIMINISHING RETURNS.

## Validate results

Capture every settled seat's final output and independently verify runtime cleanup before closing tabs or worktrees. Missing or unreadable captures block closure.
