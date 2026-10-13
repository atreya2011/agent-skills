# Sweep

Run these steps in order on every sweep, using the snapshot for judgments and chat for orchestration.


1. Read the local file. If it has an Import section, read and follow [import.md](import.md) now.
2. Run `bin/cto-snapshot`. If it exits nonzero, say so in one line and stop. It reads the `Assignee split` ruling as policy for the owner gate; do not assign an owner from the title alone.
3. Apply owner judgments in [Judgments](#judgments). Keep the UUID and description of every todo this sweep tags `+cos`. If the missing owner was its only question, remove `+ask`.
4. If this sweep tagged any todo `+cos`, send the CoS one chat message. First run `herdr agent send-keys cos ctrl+s`. Then run `herdr agent prompt cos --wait --timeout 120000` with this message, putting each todo on its own line: `AGENT MESSAGE (cto, not a human). New todos for you:\n<uuid> <description>\nAct on each within your standing rules and close it only on proof.` If the send fails, list it in the brief under Not checked.
5. Read `unreadable`. Each entry is a source the tool could not read. List them in the brief under Not checked. Never infer from silence: a tab, todo or project that depends on an unreadable source is reported as unchecked, and no todo changes on a bad read.
6. As a backstop, handle any [orchestrator messages](#orchestrator-messages) still in chat and unrecorded on their todos.
7. Apply the [judgments](#judgments).
8. Clean up tabs per [tab-cleanup.md](tab-cleanup.md).
9. Dispatch ready todos per [dispatch.md](dispatch.md).
10. Make [wiki writes](#wiki-writes).
11. If any tab has state `blocked`, raise one notification: `herdr notification show "Blocked tab" --body "<project>: <tab labels>" --sound request`.
12. Write the brief per [brief.md](brief.md).

## Orchestrator messages

An orchestrator can write to the CTO tab in chat; its message starts with `AGENT MESSAGE`, such as a `DECISION NEEDED` entry or a report that it finished. In the turn each message arrives, even outside a sweep, find the existing todo whose `run:` annotation names that orchestrator's workspace. Run `task <uuid> modify +ask` and annotate `ask: <the decision, as one plain question>`, with the orchestrator's own recommendation as `suggested:` when it gives one. Never create a todo for the message. Never close a todo or a tab on such a message. Completion is accepted only through the reply gate in [tab-cleanup.md](tab-cleanup.md).

## Judgments

The snapshot's `judgments` hold one entry per judged todo and per session. The order is rule, gate, judgment call. A rule decides with no model call. Otherwise a gate decides: a typed label with calibrated confidence. Each entry has a `verdict`. The CTO never reads transcripts itself and never overrides a verdict.

- Owner, gate `todo_owner`, verdict `act`, label `cto` or `cos`: for a still-pending todo with neither owner tag, add `+cto` or `+cos` and write the judgment annotation. The owner gate uses the existing confidence floor. A user label, a mixed-owner task, a missing source, a failed call or confidence below the floor becomes a judgment call; keep it unassigned. There is no `+user` owner tag. Do not close, rewrite or dispatch a todo from an owner judgment.
- Todo, verdict `act`, label `done`: `task <uuid> done`, then `task <uuid> annotate "<annotation>"`. Rules and gates both give an `annotation`; write it unchanged, so the verdict is recorded where the decision lands.
- Todo, verdict `act`, label `stale`: rewrite the description from the facts in the snapshot (`task <uuid> modify`) and annotate. Keep every URL, `commit:` and `session:` reference that the old description holds.
- Todo, verdict `act`, label `open`: change nothing. The todo is In progress or Queued.
- A proof rule or an `act` verdict from `todo_state` settles the state question. If it is tagged `+ask`, run `task <uuid> modify -ask`. An owner judgment removes `+ask` only when missing ownership was the todo's sole question.
- Todo, verdict `escalate`: this is a judgment call. Tag it `+ask` if it is not tagged yet. Never repeat a question: annotate `ask: <one plain question> | suggested: <label> | <annotation>` only when no earlier `ask:` annotation of the todo holds the same question. A question the user already answered is asked again only when a fact in the snapshot changed since the answer, such as the state of a linked issue or pull request, a new commit, or a different tab state. If `reason` says a source is unreadable, the todo has no assignee tag, or the gate failed, put that in the question and suggest nothing.
- Session: the label is the In progress line. An `escalate` verdict means "activity unknown".
- Todos tagged `+cos` belong to the CoS. Never change them.

Both todo gates receive the todo, current links, project tabs, linked wiki page text and matching handoff text. They never receive a transcript excerpt; `session_activity` still receives its transcript excerpt. `wiki:` notes match actual markdown files in the todo's mapped vault by whole path or filename with or without `.md`; punctuation and prose need no reference grammar. A note with no matching page or an ambiguous filename reports an error, and no partial context is judged. Oversized requests report the page or handoff that crosses the service input limit instead of truncating text.

The tool leaves the `ask:` and `cto ` annotations out of the gate's input, so an old question or verdict never feeds a new one. When the user answers a judgment call, apply the answer, remove `+ask` and annotate the todo with the answer.

## Wiki writes

Write a wiki ruling with `llm-wiki rule --vault <vault root> --on <page> "<text>"`, or edit the page directly, in the vault the vault map names for the project. Before any edit, read the target page now and confirm the claim still holds against today's vault. A stale proposal never applies. Never run ingest. Applying a proposal imported from the previous tracker is not part of this skill.
