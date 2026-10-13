# Local file


`~/.agents/local/cto.md` holds every machine value. Read it at the start of every sweep. It has one section per field:

- Pinned: tab and workspace labels that a sweep never questions.
- Profiles: per profile, its name, its agent CLI kind (`claude` or `codex`) and its transcript directory.
- Launch command: the exact shell command that starts the CTO's agent CLI.
- Vault map: per repository, checkout route or Taskwarrior project name, the vault root, and the `llm-wiki` command. Domain aliases cover todos with no repository link.
- Todo context: the handoff directory. Only Markdown handoffs that name the todo UUID (full or first eight characters) or a linked pull-request URL are included.
- Cron times: the sweep interval and the waking hours.
- Taskwarrior: the absolute path of the Taskwarrior binary. Bare `task` can be a different tool.
- Thresholds: the confidence cut-offs. The starting values are provisional until the ground-truth run sets them.
- Rulings: standing instructions from the user, one bullet each, dated.

The tool reads its settings from fenced `toml` blocks in these sections and rejects an unknown field. The blocks join into one document, so put each bare key under a table:

```toml
[commands]               # task is required; herdr, gh and git default to their bare names
task = "<absolute path>"
[pinned]
tabs = ["<tab label>"]
workspaces = ["<workspace label>"]
[[profiles]]
name = "<profile name>"
kind = "claude"          # or "codex"
dir = "<transcript directory>"
[vaults]
"<owner/name, or the project directory>" = "<vault root>"
[context]
handoff_dir = "<handoff directory>"
[thresholds]
floor = 0.6              # below this, every verdict escalates
low = 0.6                # low-stakes acts: an In progress line, a Queued rank
high = 0.8               # high-stakes acts: closing or rewriting a todo
window_hours = 6         # sessions written within this window are read
tail_bytes = 65536
excerpt_chars = 800
[gate]
log = "<gate-call log path>"
todo_context_enabled = false # enable only after a read-only known-answer evaluation passes
```

Two rulings have a fixed format:

- `Dispatch default: seat table <default or a table name>; reviewers <a count, or reviewer-<name> seats separated by commas>`. The install step writes it. Every kickoff states both settings, so a dispatch never waits on the user for them.
- `Assignee split: <the user's rule for +cos and +cto>`.

The handoff setting and rollout flag require the updated binary. Prepare them in a separate local file while evaluating; adding unknown fields to the live file before upgrading the installed binary makes the previous binary reject it.

The two todo gates remain disabled until `gate.todo_context_enabled` is true. Enable them only after a read-only known-answer evaluation has no confident wrong answer. A failed evaluation keeps both todo gates disabled and sends unresolved todos to the user; the session gate and proof rules continue.

The API key is never in this file. The tool reads `TYPESAFE_API_KEY` from the environment, else the OS keyring entry with service `typesafe` and user `api`. Without a key every gate escalates.
