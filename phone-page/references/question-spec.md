# Question spec

Write one JSON object with a string `title`, an optional `summary` that states the bottom line under the title, and 1 to 10 `questions`; publish it with `phone-page ask <spec> [id]`. Question pages add answer controls, stored drafts and a review screen to the [shared blocks](blocks.md).

Every question has a unique string `id`, a `header` of 1 to 12 characters, a string `question`, a Boolean `multiSelect`, and 2 to 6 `options`. It may also have the shared `context`, `cards`, `table`, `flow` or `flows`, and `mermaid` fields. The page renders context, cards, table, charts and options in that order. A question spec cannot contain `sections`.

Every option has a unique string `label`, a string `description`, and may set Boolean `recommended`. Recommended options come first; a question that recommends more than one option sets `multiSelect` to true, so the page shows checkboxes and marks each recommended option. Summary and option descriptions follow the [shared sentence rules](blocks.md#sentence-rules).

```json
{
  "title": "Choose a route",
  "summary": "Take the direct route because it changes the least.",
  "questions": [
    {
      "id": "route",
      "header": "Route",
      "question": "Which route should the work take?",
      "context": [
        "Take the direct route.",
        {
          "text": "It has the fewest moving parts.",
          "items": ["Use the existing path.", "Keep the current lifecycle."]
        }
      ],
      "cards": [
        {"title": "Existing path", "text": "It is ready without another dependency.", "icon": "route"},
        {"title": "Separate path", "text": "It needs its own lifecycle and owner.", "icon": "git-fork"}
      ],
      "table": {
        "columns": ["Route", "Cost"],
        "rows": [["Direct", "One change"], ["Separate", "One new path and its lifecycle"]]
      },
      "flows": [
        {
          "title": "Direct route",
          "steps": [
            "Make the smallest change.",
            {
              "text": "Did the checks pass?",
              "branches": ["Deliver the passing result.", "Repair the failing result."]
            },
            "Record the result."
          ]
        }
      ],
      "multiSelect": false,
      "options": [
        {
          "label": "Direct",
          "description": "Use the existing path and make the smallest change.",
          "recommended": true
        },
        {
          "label": "Separate",
          "description": "Create an independent path with its own lifecycle."
        }
      ]
    }
  ]
}
```

The page adds Other and a note to every question. Questions may be skipped. The latest submission replaces the previous one, while unsent choices stay in browser storage.

The rendered layout, review screens and controls follow [design.md](design.md).
