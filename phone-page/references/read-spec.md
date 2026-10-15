# Read spec

Write read pages only as JSON specs and publish them with `phone-page read <spec> [id]`; never hand-write their HTML or asset declarations. The renderer validates the spec and uses the same [blocks](blocks.md), fonts and styling as question pages.

A read spec is one object with a string `title`, a mandatory non-empty `summary` that states the bottom line, and 1 to 10 `sections`. Every section is an object with a non-empty string `heading` and any optional shared `context`, `cards`, `table`, `flow` or `flows`, and `mermaid` fields. Sections have no `options`, and a read spec cannot contain `questions`. The summary follows the [shared sentence rules](blocks.md#sentence-rules).

```json
{
  "title": "Route status",
  "summary": "The direct route is ready because the checks passed.",
  "sections": [
    {
      "heading": "Result",
      "context": ["The direct route keeps the existing lifecycle."]
    },
    {
      "heading": "Checks",
      "table": {
        "columns": ["Check", "Result"],
        "rows": [["Validation", "Passed"], ["Browser", "Passed"]]
      }
    }
  ]
}
```

On a phone, blocks follow their heading in one column. At 960 px and above, each section has its own desktop row, with heading and context on the left, cards on the right, and the table and charts below across both columns; context-only sections stay centered. Read pages have no answer controls, progress or navigation bars, stored answer drafts, or question inbox entries. See [design.md](design.md) for the complete layout.
