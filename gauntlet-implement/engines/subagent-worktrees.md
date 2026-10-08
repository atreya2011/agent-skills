# Engine: subagent-worktrees

Run the implementer as one or more `Agent`-tool subagents, each hosted
in an ephemeral git worktree off the branch. Choose this engine when
an in-session subagent starts faster than a tab.

## 1. Decompose honestly

Split the spec into parts that are genuinely independent — disjoint file
sets, code that stands alone. Default to **one part (N=1)**. Split into N>1
only when the independence is real. Each part still lands as multiple
logical commits internally.

## 2. Branch first, worktrees second

The branch already exists from the base SHA step. Each worktree is an
ephemeral host added *after* branch creation. `git worktree add -b`
locks the branch to that worktree, so use the plain `add` form.

- **N = 1:** create or reuse a host worktree with `git worktree add <path>
  <branch>`, then start the subagent in that host with `<branch>` actually checked out.
  The caller MUST verify that branch precondition; the subagent commits
  directly so the branch advances from its current commit.
- **N > 1:** add an integration worktree on the branch itself, plus one
  worktree per part on its own part-branch
  (`git worktree add <path-i> -b <branch>-p<i> <branch>`). Start every
  part's subagent in a single message so they run in parallel; each commits
  locally on its part-branch only, leaving the push to the caller.

Give every worktree a fresh, collision-free path (`mktemp -d`, or an
issue-and-part-suffixed temp dir).

## 3. Implement

Tell every subagent: the spec is the contract; make small logical commits
following the repo's conventions. Also give it the Attribution bullet of the
Agent operating contract in
[../../gauntlet/references/contracts.md](../../gauntlet/references/contracts.md)
verbatim.

## 4. Integrate

- **N = 1:** integration is complete because commits already sit on the branch.
- **N > 1:** in the integration worktree, squash-merge each part-branch in
  dependency order (`git merge --squash <branch>-p<i>` then `git commit`).
  Resolve a conflict directly, or re-implement a part inline when it turns
  out interdependent after all.

Run the full test and lint suite on the integrated branch; start a fresh
subagent to fix what it can, and record any residual failure in the report.

## 5. Clean up

Check each worktree before removing it: `git -C <path> status --porcelain`.
Remove a clean worktree whose commits already landed on the branch
(`git worktree remove <path>`) without force. Surface an unclean worktree,
resolve or confirm its changes are disposable, and only then force-remove it.
Prune stale entries (`git worktree prune`), delete the part-branches
(`git branch -D <branch>-p<i>`), and keep the branch itself.

## 6. Fix wave

Under this engine no implementer tab exists, so each fix brief, deletion round, and trim from the review stage goes to a fresh `Agent`-tool subagent in a new host worktree on the branch, set up as in step 2 and removed as in step 5.
