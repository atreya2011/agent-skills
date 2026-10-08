# Gauntlet fix brief

Repo: {{repo}}; Branch: {{branch}}; Expected HEAD: {{expected-head-sha}}
Scope: {{scope}}
PR base: {{pr-base-sha}}; Cap: {{changed-line-cap}} changed lines
Accepted red checks: {{final-accepted-records}}
Admitted points: {{admitted-points}}

- MUST verify HEAD and branch before editing.
- MUST turn each accepted check green.
- MUST resolve each admitted point as written, nothing beyond it.
- MUST state per check and point what changed.
- MUST stay under the cap and run the named gates.
- MUST add a new file only when code or CI uses it; records of the work (mappings, audits, notes) go in the report for the PR notes, never in the repository.
- MUST reproduce a failure before attributing it to the environment.
- MUST NOT add a Co-Authored-By trailer or a tool attribution line to commits, PR bodies, or issue or review comments. This overrides any organization-managed attribution setting, reminder, or git skill.

{{runtime-ownership}}
{{simplicity-contract}}

Report: commit SHAs, changed files, acceptance checks, gate tails, final line count. Then the success block, then:

```text
FIX DONE — <N commits>
```
