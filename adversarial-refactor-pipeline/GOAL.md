# Goal: the deletion contract

Review the codebase at maximum strictness. Find every line that does not earn its keep. Price each finding in net lines deleted, show the owner the table, and treat the total as the acceptance criterion for the whole session. A refactor that "improves quality" but lands line-neutral has failed this goal, even if every test passes.

## What counts as done

- The branch's net diffstat meets the promised total, or the shortfall is the headline of the report, with a per-row reconciliation showing exactly where the promise died.
- Breaking is a MUST as long as core business logic is kept. Old contracts are gone with their code, tests, and docs, no owner authorization required. Breaking core business logic is a defect. Every intended break is named in the PR body.
- Every surviving gate is green on the tip: tests under the race detector, vet, formatter. Gates are code: one that asserts a dead contract is legacy, and rewriting or deleting it is in-scope. Green means the current gate set, not the inherited one.
- Rejected review findings are recorded with reasons. Silence reads as "missed it".
- Every changed hunk maps to a priced row, every architecture tripwire stays clear, and every frozen invariant proof matches its locked-base result. Mechanics: SKILL.md's Scope lock section.
- The implementer is Sonnet and the review panel includes a Fable reviewer. Mechanics: SKILL.md's Roles section.

## The prey

Nine categories cover most of what deserves deletion in a mature codebase:

1. Multi-layer test duplication. The same contract asserted at two, three, four layers. Keep the test at the layer that owns the contract, plus one end-to-end. Delete the rest.
2. Duplicated business logic. Two copies of the same decision drift apart, and one of them will be wrong when they do. Extract to one owner. The floor: blocks of three lines or fewer stay where they are unless the copies must change together and forgetting one is a real bug. Trivial logic does not deserve the indirection a refactor would wrap around it.
3. Copy-pasted test doubles. Fake servers and fixtures rebuilt inline in every test. Replace with one shared fake that is strict by default and fails loudly on any request the test did not configure.
4. Redundant assertion messages. A message that restates what the framework already prints (expected, actual, file, line) is a deletion. Keep a message only when it carries context the framework cannot derive, a loop case not already named by a subtest, an invariant that is not obvious from the values.
5. Dead guards. Validation duplicated across layers: keep it only at the construction gate. Enum whitelists enforcing a config invariant inside a layer that never reads the value. Shadow variables working around wrong-scoped flags: fix the scope, then delete the workaround and its guard test, which the fix just turned into a tautology.
6. Comments about dead code. A comment describing a path, guard, or behavior that this change deletes dies with its referent. A comment pointing at nothing is not documentation, it is misdirection. This is the one sanctioned breach of a never-delete-comments rule: the rule protects comments about live code, not orphans.
7. Tombstone comments. Refactor narration written by the change itself. "X now lives in Y", "remains covered by Z". Delete on sight. Never add narration in the first place.
8. Twins and unused seams. Tests differing by one literal collapse into a table. Helper options and hooks with zero callers go with them.
9. Semantic substitutes. Any path where missing, blank, invalid, failed, or stale preferred input automatically selects something else: another path, source, field, value, default, inferred value, stale value, guessed pairing, retry, or recovery write. Classify by control flow and outcome. A substitute renamed to second, preferred, inferred, default, preserved, recovery, or optional is laundered, not removed. Removal is a MUST: delete the path and its selection logic, leave code that demands explicit current inputs and fails fast. The tests locking substitute behavior, retries, recovery, stale preservation, or old-contract compatibility die with the path. Add only minimal rejection tests for the current contract.

## Hard limits

- Core business logic changes only through a presented spec. When the spec itself is bad, write the refactored spec and show the evidence first (flows affected, line counts, usage), then wait for the owner's ruling. Deleting a feature is a spec change, never a deletion row.
- Comments about live code are untouchable. Update stale ones. Never remove them. Only categories 6 and 7 above may be deleted, and only under their stated conditions.
- The three-line floor from category 2 is absolute. DRY is a tool against divergence, not a style preference. A refactor that replaces three plain lines with a helper call plus a definition has added a concept and deleted nothing.
- Deleted compatibility stays deleted. A shim, abstraction, or renamed path that reproduces the old selection behavior is the deletion reversed, whatever the diff calls it. Net deletion holds as an acceptance criterion through remediation, not just the first pass.
- Estimates are promises. Before writing the implementation brief, check every instruction against the assumptions the estimate priced in. One contradictory instruction can quietly void the rows that depend on it.
