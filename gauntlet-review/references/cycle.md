# Cycle records and fix briefs

Read this file before writing a cycle admission list or dispatching a fix brief.

## Admission list

Write each ruled item as one line of the cycle's admission list, a scratch file outside the repository, with fields separated by tabs: `major` or `minor`, the path of the file that holds the check, the clause; or `point` or `skip`, the text. A check may span many lines; the script puts each file's content into its output. An acceptance checkbox that fails on the locked HEAD enters the list as a `major` line, with a scratch file that holds the failing command as its check file and the checkbox text as its clause. Then run `gauntlet/close-decision.sh <locked-head>` with the admission list on standard input. The first line it prints is the decision.

## Fix brief

On `FIX` the cycle has a major bug. Send every admitted check and point to the implementer in one fix brief: fill `final-accepted-records` with the `- Check:` items and `admitted-points` with the `- Point:` items, or `none`. With the tab engine, first send the implementer its Clear command and confirm the empty context per [tab-dispatch.md](../../gauntlet/references/tab-dispatch.md) §Clear a tab (idle or done). When the same acceptance checkbox produces a major bug in two consecutive cycles, stop fixing and send `DECISION NEEDED` to the dispatcher about the checkbox itself.
