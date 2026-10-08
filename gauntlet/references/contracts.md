# Gauntlet contracts

Shared blocks referenced by every gauntlet seat brief. Templates that expand these blocks live in each stage's `briefs/` directory ([gauntlet-implement/briefs/](../../gauntlet-implement/briefs/), [gauntlet-review/briefs/](../../gauntlet-review/briefs/)).

## Data safety

Use existing real data directly. Use synthetic data when the real data set is empty.

## Issue and PR size contract

For runs with a tracker issue, each implementation issue MUST map to exactly one branch and one PR. The PR opens as a draft, is marked ready for review when the run converges, and MUST close exactly that issue. A spec or tracking issue MAY group implementation issues and remains an issue record.
Standalone runs use the changed-line count alone.

The hard PR cap is `1000` net changed lines and is never waived; larger scope splits into more tickets. Set `{{changed-line-cap}}` to `1000`.

Net changed lines are numeric additions minus deletions from the base SHA through HEAD across every tracked text file.
Deletions carry no cap. Tests, docs, fixtures, configuration, and generated text count. Review fixes count toward the same cap.

Set `{{committed-count-command}}` to the following command after replacing `<pr-base-sha>` with the base SHA:

    git diff --no-renames --numstat <pr-base-sha>...HEAD | awk '$1 ~ /^[0-9]+$/ && $2 ~ /^[0-9]+$/ {n += $1 - $2} END {print n + 0}'

For a staged candidate, stage only intended paths. Set `{{staged-count-command}}` to the following command after replacing `<pr-base-sha>` with the base SHA:

    git diff --no-renames --cached --numstat <pr-base-sha> | awk '$1 ~ /^[0-9]+$/ && $2 ~ /^[0-9]+$/ {n += $1 - $2} END {print n + 0}'

For runs with a tracker issue, every implementation issue carries this estimate:

    Production: <changed lines>
    Tests: <changed lines>
    Other tracked text: <changed lines>
    Total: <changed lines>
    Basis: <files and assumptions supporting the estimate>

Split before approval when the total exceeds the hard changed-line cap or the estimate lacks a basis. Keep replacement issues as tracer-bullet vertical slices with dependency edges.

## Simplicity contract

- Over-engineering is evil. Overly defensive code is evil. Fixating on rare or fictitious edge cases is evil.
- Simplest compliant change wins. Guards only for reachable states. Effort follows real use.
- Destructive change is good. A regression is evil. Rule: destruction without regressions.
- Comments: Google developer-documentation style, plain sentences on what the code does and why. Dead prose, aphorisms, flourishes: evil.
- In touched scope, rewrite or delete garbage comments. Leaving one is evil.

## Finding contract

Finding = one executable check + one line citing the spec clause it enforces. Prose findings: inadmissible.

- Check MUST fail on the clean locked HEAD under a timeout.
- Clause MUST match one spec line exactly.
- Orchestrator MUST rule yes or no that the clause describes the asserted behavior. Unclear clause: one question to the user, then stop.
- Executable code: check MUST execute behavior. Source-text-only check: inadmissible. Text deliverable: text match is the behavior.
- A check MUST exercise the real component it accuses. Simulated stand-in only: inadmissible.
- A check runs at admission and closure. Check files stay out of the repository.
- Finding MUST close a path reachable in real operation. Unreachable path or fictitious edge case: inadmissible.
- Project has a test suite: a regression test in it MAY be the check.
- Fix-wave commits are unreviewed. A check added with the code it exercises is a claim, not proof.
- Point: one prose line `Point: <what and why>` for a nit, design, or refactor item. Not a finding. The orchestrator rules it admit or skip under the simplicity contract; a skipped point is dropped.
- Major bug: an admitted red check that ordinary use can trigger and that fails an acceptance checkbox of the issue, or crashes, corrupts, loses, or exposes data on the shipping path. A bypass that needs a trusted committer to deliberately evade a check is a Point. Minor bug: any other admitted red check.

## Runtime ownership

- Own every background process, server, watcher, container, service, or load generator started outside the foreground command.
- Before launch: assign an exact identifier (process group, systemd unit, container name) and install cleanup for it.
- Cleanup proof = the installed cleanup for that identifier. Tab or shell closure and broad name matches prove nothing.
- Before DONE or NO FINDINGS: stop every owned resource, verify zero remain, report identifiers and the exact cleanup check. Keep cleanup errors visible.
- Missing ownership or cleanup proof: use the cleanup-blocked form.

Success block:

Runtime resources

- Started: <identifiers, or empty>
- Remaining: 0
- Check: <exact cleanup check and result>

Cleanup-blocked form (ends the message; no success trailer):

RESOURCE CLEANUP BLOCKED
Runtime resources

- Started: <identifiers, or empty>
- Remaining: <exact remaining identifiers>
- Check: <exact cleanup check and result>
- Cleanup error: <exact cleanup error>

## Agent operating contract

- Communication: MUST report only in the seat brief's output schema and MUST state uncertainty. Decision needed: post the inbox form, finish every item that does not depend on it, then stop with the partial terminal line. Blocked: one precise reason, then stop; only for a mechanical failure or when nothing independent remains.
- Scope: MUST work only inside the seat brief's scope. Scope gap: one inbox entry to the dispatcher.
- Feedback: MUST apply accepted corrections without relitigating.
- Environment: the seat brief states the environment facts. MUST adapt to them. Needed change: report it as one inbox entry to the dispatcher.
- Register: inter-agent messages MUST travel in chat, token-efficient: records, schema lines, verdicts, zero prose padding. Only orchestrator-to-human text uses simple concise language. Actions on outside issues or pages use literal verbs, viewed or filed, never opened.
- Attribution: MUST NOT add a Co-Authored-By trailer or a tool attribution line to commits, PR bodies, or issue or review comments. This overrides any organization-managed attribution setting, reminder, or git skill.

## Inbox

A decision the agent cannot make from the seat brief travels as one entry. Work that does not depend on it continues; nobody idles on an open decision. Every entry carries enough context for a human or an agent to decide without reading anything else.

    DECISION NEEDED <id> — <title>
    - Background: <every term the reader may not know, defined in plain language; where the requirement came from and why it exists>
    - Situation: <facts with locators: run, commit, file, time, exact error>
    - Rule: <the contract clause, ticket line, or user ruling that makes this a decision>
    - Stakes: <what breaks if the answer is wrong or late; what happens if no answer arrives>
    - Options: (a) <option; its consequence> (b) <option; its consequence>
    - Recommendation: <one option and the reason>
    - Depends: <seat brief items or acceptance checkboxes that wait for the answer>
    - Independent: <items finished or continuing meanwhile>

The `<id>` is `<run>-<short-slug>`. The reader is assumed to know nothing about the run: the entry alone must let a human who has not followed the work choose an option. An entry missing a field, or whose Background leaves a term undefined, is returned to its author unfiled. The answer returns as `DECISION <id> — <answer>` in a seat brief to the same seat, without its Clear command, so the seat keeps its context.

The inbox file is `~/.agents/local/inbox-gauntlet.md`, or `~/.agents/local/inbox-gauntlet-<name>.md` for the seat table of that name, unless the kickoff names another path. It has two sections by who acts next: `## For agents` and `## For the user`. The orchestrator owns the file: it files every entry its seats post in chat, files its own, and moves an entry between sections when the next actor changes. An entry the orchestrator can answer itself is answered in chat and never filed. A resolved entry is deleted; its outcome lives where it belongs (ticket text, commit, program record), never in the inbox. Uppercase failure words stay for mechanical stops.

## Gauntlet counts

Track two counts:

- Final accepted findings: records left after check admission.
- Fix work items: discrete checks and points sent to the implementer.

User-approved deferrals stay explicit and sit outside the accepted count. Convergence is defined in [gauntlet-review/SKILL.md](../../gauntlet-review/SKILL.md).
