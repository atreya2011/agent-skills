# Child Prompt Contract

Every child subagent starts with zero shared context. It has not seen the parent's
plan, the conversation history, or any prior wave. Anything the parent needs the
child to know — scope, boundaries, escalation rules, report format — must be stated
in the prompt itself, or it simply does not exist for that child. A child that
improvises scope because the prompt was thin is a prompt-writing failure, not a
child failure.

## Every child prompt MUST include

1. **The one unit of work**, stated narrowly. One unit, not "handle wave 3" —
   name the specific files/modules this child owns.
2. **Pointer to the manifest** (`docs/programs/<slug>/manifest.md`) plus the
   relevant row(s)/section pasted inline. The manifest is the source of truth;
   the child does not re-derive scope or re-litigate a verdict.
3. **Explicit file boundary**: the exact files/globs the child may touch, plus a
   statement that everything else is out of bounds. This is what makes parallel
   children safe — disjoint boundaries need no coordination between children.
4. **The escalation contract, verbatim**:
   > If reality contradicts the manifest — stop your unit, report the
   > contradiction in your final message, do not improvise or expand scope.
5. **Report format for the final message**: what changed, files touched,
   verification run, escalations (empty list if none).
6. **UNKNOWN handling**: if you hit something the manifest marks UNKNOWN, or
   something it doesn't cover at all, stop and report — do not guess.

## Prompt templates

### Audit child

```
Audit <slice name, e.g. "API routes"> for the <program name> program.

Scope: <directories/globs to inspect>.
For each unit you find, report: path, a verdict (in-scope / out-of-scope /
UNKNOWN), what depends on it, and any risk you see.

Do not modify any files. Do not decide scope for anything outside <slice name> —
report what you find, the parent synthesizes the manifest.

If you cannot determine a verdict, mark it UNKNOWN and say what's missing
(owner input, more context, a running system to check against). Do not guess.

Report format: a table of unit / verdict / dependencies / risk, plus a short
list of anything you could not resolve.
```

### Wave execution child

```
Unit: <exact unit from the manifest, e.g. "src/api/routes/orders.ts">.

Manifest: docs/programs/<slug>/manifest.md — relevant excerpt:
<paste the unit's row + any adjacent notes/rulings verbatim>

File boundary: you may only edit <exact files/globs>. Everything else is out
of bounds — do not touch it even if it looks related.

Task: <the specific change to make on this unit>.

Escalation: if reality contradicts the manifest — stop your unit, report the
contradiction in your final message, do not improvise or expand scope.

If you hit something the manifest marks UNKNOWN or doesn't cover, stop and
report — do not guess.

Report format: what changed, files touched, verification run, escalations
(empty if none).
```

### Gate-runner child

```
Run the wave <N> gate for the <program name> program. Nothing else.

Manifest: docs/programs/<slug>/manifest.md — gate contract for wave <N>:
<paste the exact gate contract text for this wave>

Task: execute every step in the gate contract exactly as written. Do not skip
a step because it looks redundant. Capture real evidence (command output,
counts, screenshots as applicable) — "looks fine" is not a pass.

Escalation: if reality contradicts the manifest — stop, report the
contradiction in your final message, do not improvise or expand scope.

Report format: pass/fail per gate step with evidence, overall verdict,
escalations (empty if none).
```

### Scoped fix child

```
A gate failed for wave <N>. Fix exactly what the gate caught. Nothing else.

Manifest: docs/programs/<slug>/manifest.md — relevant excerpt:
<paste the unit's row + the gate failure detail>

File boundary: you may only edit <exact files/globs implicated by the
failure>. Do not touch unrelated files, even ones in the same wave.

Task: <the specific defect the gate reported> — fix it, then re-run
<the specific gate step that failed> to confirm.

Escalation: if reality contradicts the manifest — stop your unit, report the
contradiction in your final message, do not improvise or expand scope.

Report format: what changed, files touched, gate re-run result, escalations
(empty if none).
```

## Model selection

| Child type | Model |
|------------|-------|
| Audit child | `sonnet` |
| Wave execution child — mechanical edit | `sonnet` |
| Wave execution child — entanglement surgery / judgment-heavy | omit `model` (session model) |
| Gate-runner child | `sonnet` |
| Scoped fix child — mechanical | `sonnet` |
| Docs/knowledge-update child (final wave) | `sonnet` |
