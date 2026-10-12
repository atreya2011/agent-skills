# Phase 0: Audit, then freeze the manifest

Full program ceremony (audit, manifest, waves, gates) has real cost, so run Phase 0 only after Step 0 confirms the work needs a program.

1. Split the codebase into audit slices: routes, APIs and crons, database schema, shared libraries, scripts, docs. Adjust the slices to the repo. Spawn one audit child per slice, in parallel. Each child reports the units in its slice, a verdict per unit (in-scope, out-of-scope or UNKNOWN), dependencies and risks.
2. Find the project's real verification surface: test suites, verify or run skills, end-to-end suites, staging environments, backup and snapshot tooling. The gate contract is built from it.
3. Write one manifest at `docs/programs/<slug>/manifest.md`. Copy the structure of [manifest-template.md](manifest-template.md) for every new manifest. Give every unit a verdict and a wave, following the wave order below.
4. Write the gate contract for each wave from the verification surface found in step 2. A diff that looks right is not verification; the gate must run real checks. If the project has no meaningful regression surface, say so and have the user explicitly accept the weaker gate. Never silently weaken a gate to whatever checks happen to exist.
5. Ask the user to approve the manifest (waves, gates, UNKNOWN list) on a phone-page question page. Tag the restore point with `git tag program/<slug>/wave-0` and commit the manifest.

## Wave order

Order waves from safest to most irreversible. When an irreversible step runs, nothing may still depend on the old state; the earlier waves remove that risk. Adapt this canonical shape as needed:

| Wave | What | Risk |
|------|------|------|
| 0 | Audit + manifest + restore-point tag | none |
| 1 | Changes with no live dependents | near zero |
| 2 | Entanglement surgery on shared infrastructure | medium |
| 3–4 | The bulk of the code changes | medium |
| 5 | Production data changes ([irreversible-steps.md](irreversible-steps.md)) | high |
| 6 | Schema migrations | high |
| 7 | Dependency pruning + docs/knowledge update | low |
