# Program: <program-name> — <one-line goal>

Started: <YYYY-MM-DD> · Restore-point tag: `program/<slug>/wave-0`

<!-- Copy this file to docs/programs/<slug>/manifest.md and fill in every <placeholder>. -->

## Program status

<!-- A fresh session must be able to answer "where are we" from this block alone. -->

- Current wave: `<N>`
- Overall state: `<planning | wave-N-in-progress | awaiting-ruling | awaiting-approval | complete>`
- Last updated: `<YYYY-MM-DD>`

## Units

<!-- One row per audited unit. Verdict and Wave are frozen after Phase 0 approval;
     only a recorded ruling may change them. -->

| Unit | Verdict | Wave | Status | Notes |
|------|---------|------|--------|-------|
| `src/legacy-auth/**` | out-of-scope | — | done | Slated for a later program; do not touch. |
| `src/api/routes/orders.ts` | in-scope | 3 | pending | Depends on `db/schema/orders.sql` landing in wave 6. |
| `scripts/backfill_customer_ids.py` | UNKNOWN | — | blocked | Audit found no owner; needs a ruling before scheduling. |

## Wave plan

| Wave | Scope summary | Risk | Gate | Status | PR link |
|------|---------------|------|------|--------|---------|
| 0 | Audit + manifest + restore-point tag | none | Manifest reviewed and approved by owner | done | — |
| 1 | Changes with no live dependents | near zero | `<test command>` | pending | — |
| 2 | Entanglement surgery on shared infra | medium | `<test command + manual smoke steps>` | pending | — |
| 5 | Production data changes | high | Dry-run inventory approved + re-verified counts | pending | — |
| 6 | Schema migrations | high | Migration dry-run on staging snapshot + rollback tested | pending | — |

## Gate contract

<!-- Concrete, per-wave verification discovered in Phase 0. Name real commands,
     not aspirations. If a wave's real verification surface is thin, say so and
     record the owner's explicit acceptance below — never silently degrade a gate. -->

- Wave 1: `<e.g. npm test -- --scope=routes>`
- Wave 2: `<e.g. npm run e2e:smoke>` + manual check of `<flow>`
- Wave 5: dry-run inventory script `<path>`, row-count diff reviewed by owner
- Wave 6: `<migration tool>` dry-run against staging snapshot dated `<date>`

**Weak-gate acceptance:** `<none recorded | Wave N gate is weak because <reason>;
accepted by <owner> on <date>>`

## Rulings log

<!-- Append-only. Never re-ask a question already ruled on here. -->

| Date | Unit / question | Ruling by owner | Consequence |
|------|------------------|------------------|-------------|
| `<YYYY-MM-DD>` | `<unit>`: `<question>` | `<owner name>`: `<decision>` | `<what changed in the manifest>` |

## Approvals log

<!-- Append-only. Every wave go/no-go, destructive inventory, and PR merge gets a line. -->

| Date | What was approved | Approved by |
|------|--------------------|-------------|
| `<YYYY-MM-DD>` | `<e.g. Wave 0 manifest (waves, gates, UNKNOWN list)>` | `<owner name>` |
| `<YYYY-MM-DD>` | `<e.g. Wave 5 destructive inventory: 1,204 rows in orders_legacy>` | `<owner name>` |

## Escalations

<!-- Open items awaiting the next wave-boundary batch. Clear an entry once ruled
     and logged above; do not leave stale entries. -->

- `<unit>`: `<what contradicted the manifest>` — options: `<option A>` / `<option B>`
