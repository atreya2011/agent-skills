# Question spec

Write one JSON object with a string `title`, an optional one-sentence `summary` that states the bottom line under the title, and 1 to 10 questions. Every question has a unique string `id`, a `header` of 1 to 12 characters, a string `question`, an optional `context` array of 1 or more items, optional `cards`, an optional `table`, an optional `flow` array of 2 or more steps or an optional `flows` array of 1 to 4 titled charts, a Boolean `multiSelect`, and 2 to 6 options. The first context bullet states the bottom line; any remaining bullets give the reasons. Every option has a unique string `label`, a string `description`, and may set Boolean `recommended`; recommended options come first, and a question that recommends more than one option sets `multiSelect` to true, so the page shows checkboxes and marks each recommended option. The page renders context, cards, table, charts and options in that order.

Each context item is a one-sentence string or an object `{"text": "<one sentence>", "items": ["<one sentence>", ...]}` with one or more nested sub-bullets. Put numbered points in `items` without numbering their text. `cards` holds 2 to 6 objects `{"title": "<one sentence>", "text": "<one sentence>", "icon": "<lucide name>"}`, where `icon` is optional and lower-case with hyphens; the page shows them as a grid of two columns when the card is wide enough. `table` is `{"columns": ["<text>", ...], "rows": [["<text>", ...], ...]}` with 2 to 4 columns and 1 to 8 rows of one string cell per column. Each flow step is a one-sentence string or an object `{"text": "<one sentence decision>", "branches": ["<one sentence outcome>", ...]}` with 2 to 4 branches. `flows` holds objects `{"title": "<up to 40 characters>", "steps": [<flow steps>]}`, and a question uses `flow` or `flows`, not both. Each chart has a small label, bordered boxes and down arrows; decision outcomes sit side by side or stack beside a shared connector when the screen is too narrow.

The summary, context bullets, sub-bullets, card titles and texts, table columns and cells, flow titles, flow steps, branch outcomes and option descriptions each contain one sentence. Validation rejects a sentence break at `.`, `!` or `?` followed by whitespace and an uppercase letter or digit; decimals such as `0.60` and versions such as `v1.3.0` stay valid. Context bullets and sub-bullets cannot start with list numbering such as `1.` or `2)`. Errors name the offending text.

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
