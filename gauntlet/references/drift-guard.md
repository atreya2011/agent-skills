# Gauntlet drift guard

Use this file when staging a run, before its first dispatch.

Arm the drift guard once per program, not per run. At the program's first staging, register run-specific rulebooks for every planned run up front. Suffix role names per issue (implementer-N, reviewers-N), each carrying that issue's number, branch, and expected file surface. Also register a stack-wide orchestrator rulebook and a read-only default, in every folder the program will use.

Assign each dispatched tab's session to its run's role before prompting. Collect one user ratification covering every folder up front; the rendered brief is each tab's per-run scope contract.

Reuse the same checkouts and worktrees across runs, re-pointing a worktree to the next branch in place. Creating a new folder mid-program forces a new ratification, and per-run rulebook churn is evil.

Every mid-run user amendment MUST refresh the armed rulebook and its ratification before further dispatch. An arming or ratification failure reports `DRIFT GUARD BLOCKED` and stops the run.
