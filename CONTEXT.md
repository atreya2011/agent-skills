# Agent skills

Skills that let one user run and track many agent sessions in herdr. This glossary fixes the words the skills, briefs and issues use.

## Language

### People and agents

**User**:
The person the skills serve and report to.
_Avoid_: CEO, captain, operator

**CTO**:
The agent that tracks every session, todo and PR; this repo's `cto` skill.
_Avoid_: PM, watchdog, fleet manager

**CoS**:
The agent for people and admin work. Assignee of `+cos` todos. Not in this repo.
_Avoid_: assistant, secretary

**Agent CLI**:
A program that runs a coding agent in a terminal, such as claude, codex or cursor-agent.
_Avoid_: harness, binary, tool

**Profile**:
A named configuration of one agent CLI with its own settings directory and login.
_Avoid_: account, wrapper, identity

**Orchestrator**:
The agent that owns a workspace's tabs and answers for them. A run's own coordinating agent is a "gauntlet orchestrator".
_Avoid_: master, owner, lead

### Sessions and terminals

**Session**:
One agent CLI conversation with a transcript. The herdr server process is a "herdr server", never a session.
_Avoid_: chat, tab

**Tab**:
The herdr terminal a session runs in. Agent states (idle, working, blocked, done, unknown) are herdr's and are used unchanged. Skills list, read, prompt and close tabs; a pane ID is looked up only where a herdr command requires one.
_Avoid_: pane, window

**Workspace**:
herdr's grouping of tabs. Not a project.

**Pinned**:
A tab or workspace named in the CTO's local file that a sweep never questions.
_Avoid_: keep list, protected, exempt

**Idle**:
herdr's agent state for a tab waiting for input, always stated with an age.
_Avoid_: abandoned, dead, stale

### Work

**Project**:
A repository root or working directory that sessions work in. The grouping key of a brief.
_Avoid_: workspace

**Domain**:
The Taskwarrior `project` field: a broad area such as work or personal. A todo can carry a domain and a project.

**Todo**:
One Taskwarrior item.
_Avoid_: task, item, ticket

**Stale**:
A todo whose text no longer matches its sources.

**Proof**:
A fact from a source that closes or updates a todo without asking the user: a merged PR, a closed issue, a commit on the default branch, or the orchestrator saying done.
_Avoid_: evidence

**Judgment call**:
A todo change that lacks proof. It is tagged `+ask` and put to the user in a batch.
_Avoid_: decision, question

**Gate**:
A typed decision with calibrated confidence over a fixed label set, taken between rule and judgment call: it acts above a threshold and escalates below it or on unknown.
_Avoid_: classifier, check, filter

**Assignee**:
The agent a todo is tagged to, `+cos` or `+cto`.
_Avoid_: owner, holder, desk

**Handoff**:
Retagging a todo to a different assignee.
_Avoid_: transfer, migrate, reassign

**Import**:
The one-time load of todos and rulings from a previous tracker.

**Ready**:
A todo that links a GitHub issue labeled `ready-for-agent`; the only kind the CTO may dispatch on its own.
_Avoid_: actionable, specced

**Dispatch**:
The CTO starting a run for a ready todo.
_Avoid_: start, spawn, launch

**Ruling**:
A standing instruction from the user that the CTO keeps in its local file.
_Avoid_: rule, preference, policy

**Run**:
One gauntlet execution: one issue to one PR.
_Avoid_: job, pipeline, task

**Seat**:
A role in a run's seat table, mapped to a launch command: the orchestrator, the implementer or a reviewer.

**Seat brief**:
The rendered instructions one seat receives for one step of a run, named by its template, such as implementer brief or fix brief.
_Avoid_: brief

**Check**:
An executable test that fails on the locked commit.
_Avoid_: evidence, gate

**Finding**:
One check plus one quoted spec clause.
_Avoid_: evidence, report

### Sources

**Source**:
Any system the snapshot reads: herdr, git, GitHub, Taskwarrior, transcripts, vaults.
_Avoid_: data source, feed

**Transcript**:
The on-disk record of a session, kept per profile; what the snapshot reads for activity.
_Avoid_: session log, history, chat log

**Vault**:
One wiki root, as llm-wiki defines it.

**Ingest**:
llm-wiki's nightly load of new sources into a vault.

**Wiki ruling**:
A page correction appended with llm-wiki's `rule` command.

### Pages

**Page**:
A private HTML document published through a store and opened from a link.

**Read page**:
A page that presents flows, diagrams, plans or status without collecting answers.

**Question page**:
A page that presents every question to the user as tappable options and posts one submission.

**Question spec**:
The validated JSON input from which phone-page renders a question page.

**Submission**:
The latest set of answers posted by a question page. A new submission replaces the previous one.

**Store**:
The private service that publishes pages, serves their assets and keeps their latest submissions.

**Store command**:
The command phone-page calls to publish a page or read its latest submission.

**Asset manifest**:
The versioned list of page assets, source addresses and checksums that a store serves from its own origin.

### Output

**Snapshot**:
The JSON the gathering tool emits: facts, and the rule and gate judgments on them. The tool only reads.
_Avoid_: state, dump, fleet view

**Brief**:
The four-section output the CTO writes from a snapshot: Needs you, Done, In progress, Queued. Always a full picture, never a delta. Needs you holds blocked tabs and judgment calls.
_Avoid_: digest, report

**Sweep**:
One scheduled CTO run that produces a brief.
_Avoid_: loop, tick, cycle

**Reset**:
The morning context clear followed by a full sweep.
_Avoid_: rebuild

**Notification**:
herdr's `notification show`, raised only when a tab is blocked.
_Avoid_: toast, alert, ping

### Portability

**Machine value**:
Any string that ties text to a person, an organization or a machine: a name, email, hostname, IP, home directory, wrapper or profile name, organization repo or internal tool. Public agent CLI names, model ids and flags are not machine values.
_Avoid_: config, site

**Launch command**:
The exact shell command that starts an agent CLI for a seat or for the CTO. A machine value only when it names a wrapper or profile.
_Avoid_: binary, invocation

**Installed skill**:
A skill added by a skill installer and recorded in its lock file; never part of this repo.
_Avoid_: third-party skill, external skill

**Local file**:
The untracked file `~/.agents/local/<skill>.md` that holds a skill's machine values. Gauntlet's seat table is a section of its local file.
_Avoid_: seat file, config file

**Portability check**:
The scan that fails the repo on any machine value.
_Avoid_: banned-string test, lint, hygiene

**Extra patterns**:
Additional portability-check patterns read from the local file, never tracked.
