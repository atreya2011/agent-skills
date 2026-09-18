---
name: major-refactor
description: Run an explicit managed program for a large or irreversible codebase change, with audits, a committed manifest, risk-ordered waves, subagents, and approval gates. Invoke with /major-refactor.
disable-model-invocation: true
---

# Change Program

Run a large risky change the way a good program manager would: audit, freeze scope,
sequence by risk, parallelize safely, gate everything, and interrupt the human only
for decisions that genuinely need an owner. Adapted from Ryan Carson's 40-agent
migration pattern (steal the patterns, not the tooling).

The failure mode this skill exists to prevent is **context collapse**: the thing
doing the work gradually loses the plot of the overall program. The cure is a strict
split of roles plus durable state in the repo.

## Step 0: Does this need a program?

Full program ceremony (audit phase, manifest, waves, gates) has real cost. Before
anything else, check:

- Does the work include an irreversible step (prod data, schema migration, published
  API change, anything without a cheap undo)?
- Is the scope too large for one session (roughly: >100 files, or >3 independent
  workstreams, or multi-day)?

If **neither** is true, say so plainly and do the work with ordinary subagents
instead. Declining is a first-class outcome of this skill, not a failure.

## Step 1: Find program state (re-entrancy)

This skill is re-entrant. Sessions die; the manifest doesn't. The **only** source of
program state is `docs/programs/<slug>/manifest.md`, committed to the repo.

On every invocation:

1. Look for an existing manifest matching the requested program
   (`ls docs/programs/*/manifest.md`).
2. **Manifest exists** → read it, report current state to the user (wave in
   progress, pending rulings, failed gates), and resume from the next incomplete
   step. Never reconstruct state from conversation memory; the manifest wins.
3. **No manifest** → this is a new program. Run Phase 0.

## Roles: the parent writes zero code

The main session is the **parent orchestrator**. Its only jobs: plan, spawn child
subagents, review their output, sequence waves, update the manifest, and escalate to
the user. It never edits product code itself. Every piece of actual work — audits,
code changes, gate runs, doc updates — is done by a child spawned via the Agent
tool. This keeps the parent's context clean enough to hold the whole program.

Match the child's model to the task: `model: "sonnet"` for audits, mechanical
edits, gate-running, and doc updates; the session's own model (omit `model`) for
entanglement surgery and judgment-heavy work. Most of a program's work is
mechanical; paying top-tier rates for it is waste.

## Phase 0: Audit, then freeze the manifest

1. Split the codebase into audit slices (routes, APIs/crons, DB schema, shared
   libraries, scripts, docs — adjust to the repo). Spawn one child per slice, in
   parallel. Each child reports: units in its slice, a verdict per unit
   (in-scope / out-of-scope / UNKNOWN), dependencies, and risks.
2. Discover the project's **real verification surface**: test suites, verify/run
   skills, e2e suites, staging environments, backup/snapshot tooling. This becomes
   the raw material for the gate contract.
3. Synthesize everything into one manifest at `docs/programs/<slug>/manifest.md`
   using `references/manifest-template.md`. Assign every unit a verdict and a wave.
4. Author the **gate contract** per wave from the verification surface found in
   step 2. If the project has no meaningful regression surface, say so out loud and
   have the user explicitly accept the weaker gate. Never silently degrade a gate
   to "whatever happened to exist".
5. Present the manifest to the user for approval (waves, gates, UNKNOWN list) via
   AskUserQuestion. Tag the restore point (`git tag program/<slug>/wave-0`), commit
   the manifest.

Two rules give the manifest its power:

- **No worker re-litigates scope.** Every child prompt states: the manifest is the
  source of truth. Forty sessions deriving their own opinions about scope will
  disagree; one frozen manifest cannot.
- **UNKNOWN means stop and ask.** Anything the audit couldn't resolve is marked
  UNKNOWN with an instruction: don't guess, get a ruling. Rulings are recorded in
  the manifest so later sessions inherit them.

## Waves: sequence by risk, not convenience

Order waves from safest to most irreversible. The canonical shape (adapt as needed):

| Wave | What | Risk |
|------|------|------|
| 0 | Audit + manifest + restore-point tag | none |
| 1 | Changes with no live dependents | near zero |
| 2 | Entanglement surgery on shared infrastructure | medium |
| 3–4 | The bulk of the code changes | medium |
| 5 | Production data changes (inventory protocol) | high |
| 6 | Schema migrations | high |
| 7 | Dependency pruning + docs/knowledge update | low |

The point of the ordering: by the time an irreversible step runs, nothing can still
depend on the old state — the risk has already been engineered out of it.

## Executing a wave

1. Create a wave branch (`program/<slug>/wave-N`).
2. Spawn children for the wave's units. Parallel children run with
   `isolation: "worktree"` and **explicitly disjoint file boundaries** stated in
   each prompt. Disjoint boundaries are what make parallelism safe — no merge
   conflicts, no coordination between children. Use the child prompt contract in
   `references/child-prompts.md`.
3. Children merge back to the wave branch; the parent reviews each child's report
   and updates unit status in the manifest.
4. Run the wave's gate (see below). A failed gate spawns a **scoped fix child** —
   fix exactly what the gate caught, re-run the gate.
5. Open **one PR per wave** to the default branch. The user's PR approval is the
   human wave gate. Update the manifest (wave status, gate log, approvals log),
   commit it with the wave.

## Gates

A gate is a child session whose only job is to run the wave's gate contract from
the manifest and report pass/fail with evidence. Gates exist because "the diff looks
right" is not verification — Carson's gate caught a broken customer flow that review
missed. Never mark a wave complete without its gate result recorded in the manifest.

## Irreversible steps: approve the list, not the idea

For any destructive or irreversible step (prod data, schema, deletions at scale):

1. **Restore point first**: git tag; plus snapshot/backup verification appropriate
   to the backend (a child audits backup inventory: count, recency, retention).
2. **Dry-run inventory**: the child produces the exact list of what it will touch —
   row counts, file lists, object IDs. Numbers, not intentions.
3. **Human approves the inventory** via AskUserQuestion. The user approves a
   specific list, not a vague plan.
4. **Execute exactly that inventory.** Reality diverging from the approved
   inventory mid-run is an escalation, not a judgment call.
5. **Re-verify counts** after execution; record them in the manifest.

## Escalation: children report, the parent absorbs

Every child prompt includes the escalation contract: if reality contradicts the
manifest (a "safe" module has a live dependent, a verdict looks wrong), **stop your
unit, report, don't improvise**. Individual workers never make scope decisions.

The parent batches rulings at the **wave boundary**: let unaffected units continue,
collect every needed ruling, ask the user once per wave via AskUserQuestion. Record
each ruling in the manifest's rulings log. Units blocked by an unresolved ruling
demote to a later wave — note the entanglement, move on.

## Final wave: teach the system what changed

After the change ships, spawn children to update CLAUDE.md, AGENTS.md, README,
repo skills, and any agent knowledge base to describe the new architecture. In an
agent-driven codebase, documentation is load-bearing infrastructure: every future
session starts from what these files say, and stale instructions actively sabotage
future work. Then mark the program complete in the manifest.

## References

- `references/manifest-template.md` — copy this structure for every new manifest.
  Read it before Phase 0 synthesis.
- `references/child-prompts.md` — required contract for every child prompt (scope,
  boundaries, escalation, report format). Read it before spawning wave children.
