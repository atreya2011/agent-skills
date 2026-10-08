# Agent skills

Skills for running and tracking coding-agent sessions in herdr, kept free of machine values so that a clone works on any machine. `CONTEXT.md` is the glossary every skill, brief and issue uses; decisions are recorded in `docs/adr/`.

## Skills

- `adversarial-refactor-pipeline`: Run an explicit chat-only refactor pipeline through herdr tabs with one implementer, a mixed-model review panel, repeated review rounds, deletion-contract reconciliation, and a pull request.
- `be-concise`: Answer concisely in plain language without unnecessary code details.
- `find-unknowns`: Map unknown unknowns before work by exploring the codebase and domain in parallel, teaching the findings, and writing a persistent map under docs/unknowns.
- `gauntlet`: Take one approved issue from facts through blind adversarial review to a merge-ready pull request, with its seats read from a seat table in the local file.
- `gauntlet-deliver`: Deliver a converged gauntlet branch as a pull request marked ready for review, with the review notes of the closing cycle.
- `gauntlet-implement`: Stage and implement one approved issue into committed, cap-checked code on its own branch, through the implementer tab or parallel subagents in git worktrees.
- `gauntlet-review`: Run blind review cycles against a locked HEAD with executable checks and review points until a cycle has no major bug.
- `herdr-orchestrator`: Survey all Claude and Codex tabs across herdr workspaces, report activity and blockers, locate tabs, or resume dead sessions.
- `implement-issue`: Implement a GitHub issue end-to-end through a draft pull request, including commits, push, adversarial review, fixes, and PR updates.
- `major-refactor`: Run an explicit managed program for a large or irreversible codebase change, with audits, a committed manifest, risk-ordered waves, subagents, and approval gates.
- `no-slop-writing`: Draft or rewrite the author's public prose in a natural human voice built from a concrete scene, mismatch, artifact, or failure.
- `thermo-nuclear-code-quality-review`: Run an extremely strict maintainability review for abstraction quality, giant files, and spaghetti-condition growth.
- `unslop`: Edit prose to remove AI tells and restore a natural human voice.
- `where-are-we`: Reconstruct how the current session started and where it stands now by parsing its own transcript, then report it as a short readable narrative (ASD-STE100 mechanics, blunt outcome-first voice).

## Local files

A skill that needs machine values (see `CONTEXT.md`) reads them from `~/.agents/local/<skill>.md`, a file this repo never tracks. Such a skill opens with a `## Local file` section that names the path, says what the file holds, and tells the agent to read it when it exists. To set up a new machine, create each file named by such a section. Skills with a local file: `adversarial-refactor-pipeline`, `gauntlet`, `herdr-orchestrator`. `gauntlet` reads its seat table from `gauntlet.md`, or from `gauntlet-<name>.md` when a run names an alternate table.

## Portability check

`./portability-check.sh` scans the content of every tracked file, every tracked path name and every tracked symlink target, plus any paths given as arguments, for absolute home-directory paths on Unix (`/home/<user>`, `/Users/<user>`) and on Windows (`C:\Users\<user>`), email addresses and IPv4 addresses. It exits 1 listing `path:line:match` for a content or symlink-target hit and `path:name:match` for a path-name hit, and 0 when clean. When `~/.agents/local/portability-patterns.txt` exists, its lines are added as extra patterns: one extended regular expression per line, matched case-insensitively; a `w:` prefix makes the pattern a word pattern; `#` lines and blank lines are ignored; a trailing carriage return is dropped, so a CRLF file works. A word pattern — the built-in IPv4 pattern and every `w:` pattern — matches only where the character on each side is neither a letter nor a digit, so it still hits a token that `_` or `-` joins to another word, and never hits inside a longer word; those boundary characters are part of the reported match. `PORTABILITY_PATTERNS_FILE` overrides that path.

`./portability-check.sh --commit <sha>` scans one commit instead of the worktree: the content, path names and symlink targets in its tree, and its message, with the commit in front of every reported line. The pre-push hook runs it for every commit a push would add, so a machine value that a later commit removed still blocks the push.

`./portability-check.test.sh` runs the check's tests against the fixtures under `tests/fixtures/`. The tracked scan skips exactly the fixture paths that the test names, so any other file added there is scanned like the rest of the tree.

CI runs the check with the built-in patterns only, and runs its tests. To run it before every push, run this once after cloning:

```sh
git config core.hooksPath .githooks
```

That setting replaces a global `core.hooksPath` for this repo, so each hook in `.githooks/` first runs the hook of the same name from the global path, passing its arguments and standard input through, and only then runs this repo's own logic. `pre-commit` and `commit-msg` do nothing else; they exist so that the global hooks keep running.

The same rule covers commit messages, pull-request text and issues: none may contain a machine value.

## Gauntlet tests

`tests/gauntlet/seats.test.sh`, `tests/gauntlet/close-decision.test.sh` and `tests/gauntlet/send.test.sh` test `gauntlet/seats.sh`, `gauntlet/close-decision.sh` and the agent marker check of `gauntlet/send.sh`. Each runs without a local file or a herdr server, and CI runs them in the `gauntlet` job; the fixtures under `tests/gauntlet/` hold no machine values.
