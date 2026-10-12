# Gauntlet seats

Use this file to choose the seat table and the reviewers at run start, to write a seat table, and to launch seats.

## Seat table

The local file `~/.agents/local/gauntlet.md` holds one table with one row per seat and four columns:

| Seat | Kind | Args | Clear |
| --- | --- | --- | --- |
| orchestrator | claude | `--model <id> --effort <level>` | `/clear` |
| implementer | codex | `<hands-off flag> -m <id> -c model_reasoning_effort=<level>` | `/new` |
| reviewer-<name> | claude | `--model <id> --effort <level> <hands-off flag>` | `/clear` |

- Seat is `orchestrator`, `implementer`, or `reviewer-<name>`. The `reviewer-<name>` rows form the reviewer pool; the table may hold any number of them.
- Kind is a row of the CLI matrix below, and the herdr agent kind.
- Args is the text after `--` in the launch command.
- Clear is the command that clears that agent's context.

An alternate table is `~/.agents/local/gauntlet-<name>.md` with the same shape. The skill text names seats only and never holds a launch command.

## Run start

At each run start, before sending any seat brief, the orchestrator MUST take two settings from the run's kickoff when it states them, and otherwise MUST ask the user:

- Seat table: the default `gauntlet.md`, or the `<name>` of an alternate table.
- Reviewers: from the pool, by seat name `reviewer-<name>` or by count, minimum one. A count takes the first rows of the pool in table order. A bare number is always a count. Only the chosen reviewers are launched.

The orchestrator then runs `gauntlet/seats.sh [--table <name>] <reviewers>` once. It prints one line for the implementer and one for each chosen reviewer: seat, kind, args, and clear command, separated by tabs. The orchestrator is the session running this skill, so its row only has to exist; the orchestrator launches the reviewers, and with the tab engine the implementer, from those lines and records which table ran. A changed launch command applies to the next run.

A missing file, or a missing orchestrator, implementer, or chosen reviewer, stops the run with `SEAT MISSING <seat>`. A reviewer count larger than the pool stops the run with `SEAT MISSING reviewer`.

## Roles

The implementer is also the fixer: with the tab engine, one seat and one tab for the whole run.
The orchestrator is the session running this skill and is never cleared during a run.

## CLI matrix

| Kind | Model flag | Effort flag | Hands-off flag | Clear | Known model ids |
| --- | --- | --- | --- | --- | --- |
| claude | `--model <id>` | `--effort low\|medium\|high\|xhigh\|max` | `--dangerously-skip-permissions` | `/clear` | claude-opus-4-8, claude-fable-5-1, claude-sonnet-5 |
| codex | `-m <id>` | `-c model_reasoning_effort=low\|medium\|high\|xhigh\|max\|ultra` | `--yolo` | `/new` | gpt-5.6-sol, gpt-5.6-terra, gpt-5.6-luna, gpt-5.5, gpt-6-astra |
| cursor | `--model <id>` | suffix on the id: `-low`, `-medium`, `-high`, `-xhigh` | `--yolo` | `/clear` | cursor-grok-4.5-high, cursor-grok-4.6-high, gpt-5.6-sol-high |

gpt-5.5 stops at xhigh and gpt-5.6-luna at max. Every value comes from the CLI's own help or model list.
