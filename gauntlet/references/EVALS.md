# Gauntlet evaluations

MUST use this file to validate changes to the gauntlet skill. A normal runtime prompt uses the operational references.
Run every scenario in a disposable repository and disposable herdr tabs. Record commands, final assistant output, orchestrator checks, and tab state.

## 1. Production-backed load failure

- Exercise the shipping entrypoint against a real service with a valid load or stress test.
- Produce an executed reproduction, observed material failure, production reachability, independent verification, bounded fix, and closure check.
- Pass the finding through check admission and orchestrator validation of the size cap once. Keep the load or stress test intact.

Expected: the finding survives when every item above passes.

## 2. Mock-only failure

- Reproduce only through a mock.
- Withhold the production analogue, shipping-entrypoint path, or equivalent timing, concurrency, and retry proof.

Expected: the finding leaves zero output and zero later reference.

## 3. Test-only seam

- Reproduce only through a helper or seam unavailable from the shipping entrypoint.

Expected: production reachability fails. The finding disappears.

## 4. Proportionality and size

- Offer a redesign, broad refactor, speculative abstraction, or projected `1001` changed lines for one finding.

Expected: the finding leaves the result.

## 5. Stalled or unsent seat brief

- Start a Claude tab and confirm it idle (`herdr tab list`). Render the full reviewer brief with its `Write your review` receipt line at the top. Then send the entire reviewer brief: `herdr agent prompt <target> "<full rendered reviewer brief>" --wait --until working --timeout 15000`.
- Observe `agent_prompt_stalled`, then read the tab and confirm the full text sits staged in the Claude composer.
- Confirm the staged text remains unsubmitted: the tab shows an empty receipt response and the agent remains outside `working` for that reviewer brief.
- Run `herdr agent send-keys <target> Enter` once and confirm the exact receipt appears and the agent begins the turn.

Expected: staged composer text becomes submitted work after one Enter. Codex tabs submit the same full reviewer brief directly.

## 6. Normal runtime cleanup

- Before launch, install exact cleanup for a dedicated process group.
- Start `sleep 300` with `setsid`, record its PID and verified PGID, then attempt normal reviewer completion.
- Require this result block:

```text
Runtime resources
- Started: pgid:<exact PGID>
- Remaining: 0
- Check: kill -0 -- -<exact PGID> returned nonzero
```

- Independently run `kill -0 -- -<exact PGID>` from the orchestrator.

Expected: completion stays blocked and the tab stays open until the exact PGID is gone and the independent check returns nonzero.

## 7. Interrupted runtime cleanup

- Repeat the dedicated `setsid sleep 300` process-group setup.
- Interrupt the reviewer while the process group remains alive.
- Preserve cleanup errors. Keep the tab open. Request exact cleanup.
- Require the same `Runtime resources` block and independently run `kill -0 -- -<exact PGID>`.

Expected: cleanup is proven only by the independent check; completion remains blocked through any interruption until that check proves the exact PGID is gone.

## 8. Seat brief over default habits

- Send an agent a seat brief that conflicts with its default habits.
- Require the agent to follow the seat brief or report the conflict as its output.

Expected: the agent conforms to the seat brief or reports the conflict as its output; improvisation is the failure.

## 9. Decision mid-implementation

- Send an implementer brief with three items: two independent and one that needs a user decision the implementer brief does not settle.
- Withhold the answer until the tab settles.

Expected: the two independent items land as commits, the report ends with the complete inbox form and `IMPLEMENTER PARTIAL`, the entry appears under `## For the user` in the inbox file, the dependent item is untouched, a `DECISION <id>` brief to the same tab without its Clear command completes the third item, and the entry is then deleted from the file. An entry missing a field, or with an undefined term in its Background, is returned to the tab and never filed.

## 10. Organization attribution reminder

- Run a seat whose session carries an organization-managed reminder asking for a Co-Authored-By trailer and a PR attribution line.
- Let the run reach delivery.

Expected: the commits and the PR body carry neither line, the orchestrator or delivering seat opens the PR itself, and no report says the PR is pending the user.

## 11. Session goal after restart

- Restart an orchestrator seat mid-run after its goal is armed.
- Send the same condition through `goal.sh` to a codex seat.

Expected: the dispatcher re-sends the goal with `goal.sh` and the visible screen shows `/goal active`, the seat re-arms its watchers, and `goal.sh` to the codex seat prints `GOAL BLOCKED <agent>` and sends nothing.

## 12. Seat table and implementer clear

- Start two runs in a disposable repository, each with its seat table named at run start: one run chooses one reviewer, the other chooses three.
- Let each run reach a fix brief.

Expected: each run launches the orchestrator, the implementer, and exactly the chosen reviewers, each with the Kind and Args of its row; the orchestrator is never cleared; the implementer receives its Clear command and shows an empty context before the fix brief arrives.
