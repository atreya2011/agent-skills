---
name: thermo-nuclear-code-quality-review
description: Run an extremely strict maintainability review for abstraction quality, giant files, and spaghetti-condition growth. Use for a thermo-nuclear code quality review, thermonuclear review, deep code quality audit, or especially harsh maintainability review.
disable-model-invocation: true
---

# Thermo-Nuclear Code Quality Review

Use this skill for an unusually strict review focused on implementation quality, maintainability, abstraction quality, and codebase health.

Above all, this skill should push the reviewer to be **ambitious** about code structure. Move past local cleanup opportunities and actively search for "code judo" moves: restructurings that preserve behavior while making the implementation dramatically simpler, smaller, more direct, and more elegant.

## Core Prompt

Start from this baseline:

> Perform a deep code quality audit of the current branch's changes.
> Rethink how to structure / implement the changes to meaningfully improve code quality without impacting behavior.
> Work to improve abstractions, modularity, reduce Spaghetti code, improve succinctness and legibility.
> Be ambitious, if there is a clear path to improving the implementation that involves restructuring some of the codebase, go for it.
> Be extremely thorough and rigorous. Measure twice, cut once.

## Steps

1. Read [references/standards.md](references/standards.md) before auditing the diff. Apply its non-negotiable standards, and ask its review questions of every meaningful change.
2. Read [references/findings.md](references/findings.md) before writing a finding. It lists what to flag aggressively, the preferred remedies, and the review tone.
3. Report the findings in the user's review output format, `~/.agents/rules/review-output-format.md`. Within that flat ranking, favor a small number of high-conviction findings over a long list of cosmetic notes when larger structural issues exist. Put structural regressions and missed code-judo moves at the top of the ranking (`design`, `bug`, `dup`).
4. Read [references/approval-bar.md](references/approval-bar.md) before approving a change or leaving blocking feedback.
