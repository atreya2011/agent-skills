# Shared page blocks

Read sections and questions use the same optional context, cards, table and flow fields, validated once and rendered by the same components. Choose the blocks that carry the facts the reader needs; the page type adds its own heading, layout and controls.

## Context

`context` is an array of one or more non-empty strings or objects `{"text": "<one sentence>", "items": ["<one sentence>", ...]}` with a non-empty `text` and one or more non-empty nested sub-bullets. The first bullet states the bottom line; remaining bullets give the reasons. Put numbered points in `items` without numbering their text.

## Cards

`cards` holds 2 to 6 objects `{"title": "<one sentence>", "text": "<one sentence>", "icon": "<lucide name>"}` with non-empty titles and text. The optional `icon` is a lower-case Lucide name with hyphens, such as `book-open`. Cards show parallel facts, sources or constraints.

## Table

`table` is `{"columns": ["<text>", ...], "rows": [["<text>", ...], ...]}` with 2 to 4 non-empty string column headers and 1 to 8 rows of one string cell per column. Use it for structured comparisons, including numbers and sources.

## Flow charts

`flow` is an array of two or more steps. A step is a non-empty string or an object `{"text": "<one sentence decision>", "branches": ["<one sentence outcome>", ...]}` with a non-empty `text` and 2 to 4 non-empty branches. Alternatively, `flows` holds 1 to 4 objects `{"title": "<title>", "steps": [<flow steps>]}` with a non-empty title of up to 40 characters. A question or section uses `flow` or `flows`, not both.

Charts use the existing CSS renderer: bordered boxes and arrows, with decision outcomes side by side or stacked beside a shared connector on narrow screens. They are not numbered lists or code blocks.

## Sentence rules

The summary, context bullets and sub-bullets, card titles and text, table columns and cells, flow titles and steps, branch outcomes and question option descriptions each contain one sentence. Validation rejects a sentence break at `.`, `!` or `?` followed by whitespace and an uppercase letter or digit; decimals such as `0.60` and versions such as `v1.3.0` stay valid. Context bullets and sub-bullets cannot start with list numbering such as `1.` or `2)`. Errors name the offending text.

See [question-spec.md](question-spec.md) and [read-spec.md](read-spec.md) for each page type's own fields and [design.md](design.md) for presentation.
