---
name: implement-issue
description: Implement a GitHub issue end-to-end through a draft pull request, including commits, push, adversarial review, fixes, and PR updates. Use only when the user explicitly asks to implement or ship an issue to a PR, not for read-only issue discussion.
---

# implement-issue

Take a GitHub issue from "approved spec" to "open draft pull request with an
internal adversarial review pass" in one autonomous run. The issue body is the
contract — it has already been planned and approved, so this skill proceeds
straight through the happy path, running implementation without pausing to
re-plan or ask for sign-off. It branches, dispatches implementation, commits,
pushes, opens the draft PR, dispatches adversarial review, integrates review
fixes, updates the PR body, and leaves the PR ready for human review.

## Operating principles (read first)

- **The issue is the approved plan.** "Implement" means the thinking is done —
  execute the settled scope directly, skipping any re-litigation or planning gate.
- **Build the laziest thing that satisfies the issue.** The issue fixes *what* to
  build; you minimize *how*. Take the first rung that holds: stdlib over custom code,
  a native platform feature over a new dependency, an already-installed dependency
  over a new one, one line over fifty, the minimum code that works only after those
  fail. Build only abstractions the issue actually requires, add scaffolding only
  for what's needed right now, favor deletion over addition, keep the diff as
  short as it can be. Lazy means less code with equally solid logic — pick
  the form that is correct on edge cases. Always keep input validation at trust
  boundaries, error handling that prevents data loss, security, and accessibility
  fully intact. Mark a deliberate simplification or a known-ceiling shortcut (name
  the ceiling and its upgrade path) in a plain comment, following the repo's comment
  style, using whatever tag fits the surrounding code. This governs implementation
  only; it treats every issue requirement as fixed (questioning them would
  contradict the principle above).
- **Delete every fallback, legacy path, and dead guard — always.** After a fix or
  refactor, DELETE the old path; the result MUST be free of any fallback, compat
  shim, or legacy branch. Code and tests MUST cover only reachable, current
  behavior — checking for deleted, unreachable, or impossible behavior tests a
  tautology, since a closed enum/schema is limited to producing only its currently
  defined values. Tests MUST exercise live behavior only. Preserve a legacy or
  fallback path ONLY when the issue explicitly asks for it. Less code, fewer
  guards.
- **Run straight through on the happy path.** The user invoking the skill is the
  consent. Run straight through to an open PR.
- **Best-effort on blockers, always surfaced.** If you hit something outside what
  the issue covers (an undecided API shape, a test that keeps failing, or a base
  that resists merging), make the most reasonable assumption, keep going, and open
  a *ready* PR — but surface every assumption loudly in the PR body, so a reviewer
  is sure to see it. A documented guess beats a stalled run; an *undocumented*
  guess is a bug.
- **GitHub only.** This skill uses `gh`. On a non-GitHub remote, stop and say so.
- **End at an open PR.** Print its URL and stop there — merging is the user's call.
- **Draft first, review before handoff.** Open the PR as a draft before
  adversarial review. Keep it draft while blockers or incomplete validation remain.
  If review is clean and validation passes, mark it ready for review unless the
  user or repo convention explicitly asks for draft-only PRs.

## Inputs

The user passes a bare issue number (`123`) or a full issue URL. If neither is
given, run `gh issue list --state open` and let the user pick one — choosing
*which* issue is input selection, which stays consistent with running straight
through on the happy path rather than acting as a pipeline stop.

## Pipeline

### 1. Pre-flight (fail fast, before touching anything)

- **Host check.** `gh repo view --json nameWithOwner,url` — confirm the remote is
  github.com, or stop: this skill is GitHub-only.
- **Clean tree.** `git status --porcelain`. If non-empty, **abort** with a clear
  message ("working tree dirty — commit or stash first, then re-run"). Keep
  pre-existing uncommitted edits out of the issue's branch entirely; folding them
  in silently ships unrelated work and corrupts the diff a reviewer sees.
- **Up-to-date base.** Detect the default branch
  (`gh repo view --json defaultBranchRef -q .defaultBranchRef.name`; fall back to
  `git symbolic-ref refs/remotes/origin/HEAD`). `git fetch origin`. Branch from
  `origin/<default>` so the work sits on the latest base.
- **Already-shipped guard.** Check whether the issue already has an open PR before
  building anything: `gh pr list --state open --json number,headRefName,url,body`
  and match the issue (a linked PR, a `Closes #<n>` in a body, or an issue-numbered
  head branch). If one exists, **stop and report its URL** instead of opening a
  duplicate — the issue is already in flight, and a duplicate is exactly what
  leaves a second branch/worktree fighting the first.
- **Record the starting branch** so the user's main worktree can be restored at the
  end. All real work happens in dedicated worktrees — the main worktree stays put
  throughout.

### 2. Resolve the issue

`gh issue view <n> --json number,title,body,labels,url,assignees`. Read the body as
the spec. Note labels (they hint the commit/branch *type*: a `bug` label → `fix`, a
`feature`/`enhancement` label → `feat`, etc.).

**Inspect declared blockers before starting — this step is mandatory.** Many issue
bodies declare dependencies: a `## Blockers` section (`Blocked by:` hard, `Soft
blocker:` soft), a `Depends on` / `Blocked by` list, or a `blocked` label. Parse them
from the body and labels, then check each referenced issue's state
(`gh issue view <blocker> --json number,state,title`):

- **Open hard blocker** (`Blocked by` / `Depends on`) → **stop and report it**;
  start work only if the issue body explicitly waives it. Building on an
  unresolved hard dependency produces a PR that cannot land.
- **Open soft blocker** → proceed, but inspect each for overlap with this issue's
  scope and record any accepted overlap in the PR body's
  **`## Assumptions & unverified`** section.
- **Closed blocker** → note it and continue.

### 3. Plan the decomposition (silently — proceed on your own)

Split the issue into the **independent** parts it actually has — parts whose file
sets stay disjoint and whose code stands alone, independent of the others. Be
honest:

- Most issues are a single concern → **one part (N=1)**. That is the normal case.
- Only split when parts are genuinely independent (e.g. "add endpoint A" and "add
  unrelated endpoint B"). **Split only when the independence is real** —
  manufacturing a split to force parallelism just creates integration conflicts
  and wastes the parallel machinery.

Each part still produces *multiple logical commits* internally (e.g. schema → wiring
→ tests); decomposition is about independent units of work, commits are about
reviewable slices within a unit.

### 4. Name and create the branch

- **Detect the repo's branch convention** from existing branches
  (`git branch -r`). Match what you see. For example, a repo using
  `feat/105-active-flow-history` and `refactor/62-split-by-responsibility` wants
  `<type>/<issue>-<slug>`; one using `feat/issue-104-cash-path-guards` wants
  `<type>/issue-<issue>-<slug>`.
- **Fallback** for a repo whose branch history is inconsistent: `<type>/<issue>-<slug>`,
  where `<type>` comes from the issue labels/title and `<slug>` is a short
  kebab-case form of the title.
- **Create the branch first, as a first-class branch decoupled from any worktree:**
  `git branch <issue-branch> origin/<default>`. Keep branch creation independent of
  any worktree — using `git worktree add -b` instead couples the two, and that
  coupling is what leaves the branch locked (`gh pr checkout` failing, re-runs
  erroring "branch already exists") when a worktree lingers. The branch must
  outlive every worktree.
- **If the branch already exists** (local `git rev-parse --verify <issue-branch>`,
  remote `git ls-remote --heads origin <issue-branch>`, or held by a worktree in
  `git worktree list`): the already-shipped guard ruled out an open PR, so it is a
  stale or aborted attempt. Reuse it to resume, or create a suffixed branch
  (`<issue-branch>-v2`); if a stale worktree holds it, free it with
  `git worktree remove --force` after confirming that worktree is clean and pushed.
  Surface whichever path you take.

### 5. Implement (parallel sonnet subagents in worktrees)

Worktrees are **ephemeral hosts for subagent work**, created off the
already-existing issue branch — they host the work; branch creation happens
earlier, in step 4.

- **N = 1:** add one worktree checked out to the **existing** issue branch
  (`git worktree add <path> <issue-branch>` — note: omits `-b`, since the branch
  already exists). Dispatch one sonnet subagent (`Agent` tool, `model: "sonnet"`) to
  implement there. It writes the code, runs the tests/lint relevant to its change,
  and makes the logical commits **directly on the issue branch**. Nothing to
  integrate afterward.
- **N > 1:** first add the **integration worktree** — one worktree checked out to the
  issue branch itself (`git worktree add <integration-path> <issue-branch>`), the
  place the parts come together in step 6. Then create one worktree per part, each on
  its own part-branch **off the issue branch**
  (`git worktree add <path-i> -b <issue-branch>-p<i> <issue-branch>`), and dispatch the
  sonnet subagents **in a single message so they run in parallel**. Each implements
  its part, runs its relevant tests/lint, and commits its logical commits on its
  part-branch. Reserve pushing for the main agent — subagents commit locally only.

Use a unique, collision-free worktree path per worktree (a `mktemp -d` path or an
issue-and-part-suffixed temp dir), freshly generated each time so it cannot already
exist.

Tell every subagent: the issue body is the spec; make small logical commits that
follow this repo's commit conventions (see §8); document any assumption it had to
make; return a summary of files changed, commits made, assumptions, and test status.
Apply the build-minimal operating principle: minimal diff; stdlib / native / an
already-installed dependency before a new one; only abstractions the issue actually
requires; validation, error handling, and security kept fully intact. Apply the
delete-legacy-paths principle too: delete the paths a fix obsoletes, leaving only
current paths and tests that cover live behavior.

### 6. Integrate and verify

- **N = 1:** nothing to integrate — the commits are already on the issue branch.
- **N > 1:** in the **integration worktree** on the issue branch (created in step 5),
  **squash-merge each part-branch into the issue branch** in dependency order:
  `git merge --squash <issue-branch>-p<i>` then `git commit` with a message describing
  that part. Each part collapses to one clean logical commit on the issue branch.
  Disjoint file sets squash cleanly; on a conflict, resolve it, or if a part turns out
  to be interdependent after all, re-implement the conflicting slice inline instead
  of forcing a broken merge.
- **Run the full test + lint suite** on the integrated branch (detect commands from
  the repo: `package.json` scripts, `Makefile`, `go test ./...`, CI config, project
  CLAUDE.md). Honor the zero-warnings expectation: try to get it clean. Dispatch a
  sonnet subagent to fix failures it can fix, holding to the build-minimal and
  delete-legacy-paths principles.
- **Residual failures stay non-blocking** (per the best-effort rule) — record them
  for the PR body. Cite fresh evidence generated for this run (`go test -count=1`,
  a real run), rather than a cached `ok`.

### 7. Clean up

Remove every worktree this run created — but **check before you force**, per the
check-before-mutating rule. For each, run
`git -C <path> status --porcelain`. If it is clean and its commits are already on the
issue branch (step 6's squash-merge captured part work; N=1 commits land there
directly), `git worktree remove <path>` succeeds without force. If it reports
uncommitted or untracked changes, **stop and surface them** — resolve the work
first, either integrating it or explicitly confirming the changes are disposable,
and only then run `git worktree remove --force <path>`. Run
`git worktree prune` to clear entries whose directories are already gone (it only
drops missing ones). Delete the part-branches (`git branch -D <issue-branch>-p<i>`).
The main worktree stayed in place throughout; confirm it still sits on the recorded
starting branch. **Keep the issue branch** — it is a first-class branch (step 4), so it
survives worktree removal unlocked, ready for `gh pr checkout` and re-runs.

### 8. Commit & PR conventions (discover fresh per repo)

This is a global skill — it adapts to each repo and the rules in effect for the
session rather than imposing one house style:

- **Read** the repo's `CLAUDE.md`, `AGENTS.md`, `CONTRIBUTING.md`, and a sample of
  recent history (`git log --format='%s%n%b' -20`) to learn the commit-message
  format (conventional-commit types? lowercase? ticket IDs? a body style?).
- **Honor any organization-level rules already in your context** — e.g. a required
  author/committer identity split, mandatory trailers (`Co-Authored-By`), or content
  that is *forbidden* (such as session links). When unsure, match what the most
  recent commits actually do.
- Commit messages describe what was done, staying free of project-plan structure
  such as phase/round/step labels.

### 9. Push and open the draft PR

- `git push -u origin <issue-branch>`.
- **PR body style — find it, follow it:**
  1. Look for a template: `.github/PULL_REQUEST_TEMPLATE.md`,
     `.github/pull_request_template.md`, `.github/PULL_REQUEST_TEMPLATE/*.md`,
     `docs/`, or repo root (case-insensitive). If found, fill it in.
  2. **A missing template → infer the style** from recent merged PRs:
     `gh pr list --state merged --limit 10`, then read a few bodies with
     `gh pr view <n> --json title,body`. Match their structure, section headings,
     and title format.
- **Title** follows the repo convention (commonly the conventional-commit style of
  the lead commit, e.g. `feat: model 2015 family-loan obligations`).
- **Body** must include, in addition to whatever the template/convention dictates:
  - A short summary of what changed and why (grounded in the issue).
  - `Closes #<issue>` so merging auto-closes the issue.
  - **`## Assumptions & unverified`** — a prominent section near the top, present
    only when you made assumptions or left verification incomplete. List each guess,
    each unresolved decision, and any failing/ skipped checks. This is the safety
    valve for the run-straight-through principle; keep it prominent, always.
- While internal review is pending, include **`## Internal review`** with the
  current state: `pending adversarial review`, `blockers under repair`, or
  `adversarial review passed`.
- `gh pr create --base <default> --head <issue-branch> --title "..." --body "..."`.
  Open it as a **draft**.

### 10. Dispatch adversarial review

After the draft PR exists, dispatch an adversarial review subagent. Tell it:

- Read the live PR body, PR diff, linked issue body, current branch head, and
  validation evidence.
- Use the repo's PR review skill or review rules if present.
- Flag over-engineering as findings: unrequested abstractions, a new dependency where
  stdlib / a native feature / an already-installed one suffices, dead scaffolding, or
  a diff larger than the issue needs.
- Flag legacy/fallback slop as findings: a fallback or compat path left after a fix, a
  guard checking for deleted or unreachable behavior, a test asserting deleted
  behavior — these MUST be deleted outright.
- Limit itself to reading and reporting.
- Treat draft status as a neutral state unrelated to implementation quality; the
  main agent marks the PR ready after a clean review and final PR-body update.
- Return blockers first, with exact file paths, lines, issue criteria, expected
  behavior, accept/reject examples where useful, and verification commands.
- Return `LGTM` only if the PR satisfies the issue body, repo contracts, generated
  artifact expectations, validation, and review-thread state.

Treat the review as a gate before final handoff. A draft PR without adversarial
review is incomplete.

### 11. Merge review fixes

If the adversarial review returns blockers:

- Create a review-fix worktree on a temporary branch off the issue branch.
- Dispatch a sonnet implementation subagent to fix only those blockers, holding to
  the build-minimal and delete-legacy-paths principles.
- Require small logical commits and fresh validation evidence.
- Merge the review-fix branch back into the issue branch. Prefer a normal merge or
  squash merge that preserves a clear review-fix commit, keeping the original
  implementation commits intact.
- Push the updated issue branch.
- Re-run the adversarial review loop until every blocker is resolved, or until a
  blocker turns out to be resolvable only with a user decision.

If a blocker needs a user decision, keep the PR draft, record the decision needed in
`## Assumptions & unverified`, and report the PR URL plus the blocker.

### 12. Update the PR body and readiness state

Refresh the PR body after the final review pass:

- Replace stale implementation claims with the final changed surface.
- Record validation commands actually run and their results.
- Record the adversarial review result.
- Keep `## Assumptions & unverified` if any assumption, skipped check, failing
  check, or unresolved blocker remains.
- Remove `## Assumptions & unverified` only when it is empty.

If every blocker is resolved and validation passes, mark the PR ready for review
unless the user or repo convention says to leave all PRs as drafts:

```bash
gh pr ready <pr>
```

### 13. Report

Print the PR URL. State the test/lint status plainly (pass, or which checks failed).
**End the run here, at the open PR — merging is the user's call.**

## Guardrails recap

- Abort on a dirty working tree.
- GitHub remotes only.
- Inspect the issue's declared blockers; stop on an open hard blocker unless the
  issue body explicitly waives it.
- Surface every assumption in the PR body — every guess must be documented; an
  undocumented guess is the one failure mode this skill exists to prevent.
- Open a draft PR before adversarial review.
- Run adversarial review before final handoff.
- Integrate review fixes before marking a PR ready.
- End every run at an open PR; merging is the user's call.
