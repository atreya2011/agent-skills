# Approval Bar

Reserve approval for changes that clear the structural bar below, regardless of how correct the behavior seems.
The bar for approval is:

- structural quality stays intact, free of clear regression
- every visible opportunity to make the implementation dramatically simpler gets taken
- any file-size growth stays justified
- special-case branching stays free of obvious spaghetti-growth
- abstractions stay direct, free of obviously hacky or magical constructs that make the code harder to reason about
- wrappers, casts, and optionality stay minimal and justified, keeping the real design clear
- architecture boundaries stay respected, with canonical helpers reused instead of duplicated
- obvious decomposition opportunities that would materially improve maintainability get taken

Treat these as presumptive blockers unless the author can justify them clearly:

- the PR preserves a lot of incidental complexity when there is a plausible code-judo move that would delete it
- the PR pushes a file from below 500 lines to above 500 lines
- the PR adds ad-hoc branching that makes an existing flow more tangled
- the PR solves a local problem by scattering feature checks across shared code
- the PR adds an unnecessary abstraction, wrapper, or cast-heavy contract that makes the design more indirect
- the PR duplicates an existing helper or puts logic in the wrong layer when there is a clear canonical home

When those conditions are unmet, leave explicit, actionable feedback and push for a cleaner decomposition.
