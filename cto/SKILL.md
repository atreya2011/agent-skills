---
name: cto
description: Run one CTO sweep. Rebuild the picture of every herdr tab, session, todo and pull request from sources, close todos only on proof, put judgment calls to the user in one batch, dispatch ready todos, and write the brief. Use when a prompt says to run the cto skill, sweep, or reset.
---

# CTO

The CTO runs in one long-lived tab. On every sweep it rebuilds its picture from sources, never from chat memory. The `cto-snapshot` binary reads the sources and judges each todo and session with rules and gates. The CTO acts on those judgments and writes the brief. Words follow [CONTEXT.md](../CONTEXT.md).

## Local file

`~/.agents/local/cto.md` holds every machine value. Read it at the start of every sweep. Before configuring or interpreting its fields, read [references/local-file.md](references/local-file.md).

## Read-only evaluation

Before evaluating or enabling the todo gates, read [references/evaluation.md](references/evaluation.md). Keep both disabled until a read-only known-answer evaluation passes; a confident wrong answer keeps both disabled.

## Install

Before building the tool, configuring the tab or installing cron, read [references/install.md](references/install.md). After install, run the manual checks in [references/checklist.md](references/checklist.md).

## Orchestrator messages

Handle every orchestrator message in the turn it arrives, even outside a sweep. Before acting, read [references/sweep.md](references/sweep.md#orchestrator-messages).

## Sweep

1. Read the local file. If it has an Import section, read and follow [references/import.md](references/import.md) now.
2. Run `bin/cto-snapshot`. If it exits nonzero, say so in one line and stop. Do not assign an owner from the title alone.
3. Read [references/sweep.md](references/sweep.md) before applying owner judgments; keep the UUID and description of every todo newly tagged `+cos`.
4. Send the CoS one chat handoff for those todos, following that reference. Report send failures under Not checked.
5. List `unreadable` entries under Not checked. Report dependent tabs, todos and projects as unchecked; never infer from silence or change a todo on a bad read.
6. As a backstop, handle any orchestrator messages still in chat and unrecorded on their todos per [references/sweep.md](references/sweep.md#orchestrator-messages).
7. Apply the remaining [judgments](references/sweep.md#judgments). Never read transcripts yourself or override a verdict. Never change `+cos` todos.
8. Before asking orchestrators or closing tabs, read [references/tab-cleanup.md](references/tab-cleanup.md). Never question pinned tabs or workspaces, or close pinned tabs.
9. Before dispatching ready todos, read [references/dispatch.md](references/dispatch.md).
10. Before wiki edits, read [references/sweep.md](references/sweep.md#wiki-writes). Never run ingest.
11. If any tab is blocked, raise one notification as specified in [references/sweep.md](references/sweep.md#sweep).
12. Before writing the full brief, read [references/brief.md](references/brief.md).
