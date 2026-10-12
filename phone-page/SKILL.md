---
name: phone-page
description: Publish designed read pages and tappable question pages for replies that do not fit cleanly in chat on a phone.
---

# Phone page

Use a page when a reply to the user needs more than two sentences, a diagram, a table or any question. A chat reply that points to a page has at most two sentences plus the link, with no code block, table or list. Put every question to the user on a question page, never in reply text or the agent CLI's question dialog. Chat between agents is unaffected.

For read pages and question pages, open with the bottom line, then give the reasons. Write each bullet and sub-bullet as one sentence. Put numbered points in nested sub-bullets, never as numbering in bullet text. Show every process as a flow chart with bordered boxes and arrows, using branches for decisions: draw the chart on a read page or use the question spec's `flow` or `flows` field on a question page. Show parallel facts as a card grid and structured comparisons as a table instead of bullet prose. Content text is 17 px.

- A page covers one project; questions or content about different projects go on separate pages; the page title names the project.
- Give each question exactly what the user needs to decide, with no wall of text and no flow-only page:
  - the problem, what was tried, the result, and what needs deciding, then the recommendation with its one-line reason, as the first context bullets;
  - the facts it rests on in a `table`, with numbers and sources;
  - known limits as two or three one-line `cards`, or as a table row when there is only one;
  - the process as a `flow` or `flows` chart with short boxes;
  - each option's consequence in its description.
- Add nothing beyond those parts, and never cut a fact the decision depends on.
- Follow Grice's [maxim of quantity](https://en.wikipedia.org/wiki/Cooperative_principle#Maxim_of_quantity).
- Fix an open page in place with `phone-page ask <spec> <id>` or `phone-page publish <file> <id>`; never publish a replacement page for content the user already has a link to. Replacing a page discards its previous submission.

Read [references/design.md](references/design.md) before making a read page. Use `phone-page ask` for a question page; do not hand-write its form.

- Before writing any page, use `git ls-files` in the current repository to find files named `glossary.md` at any path, ignoring letter case, and read any matches.
- Use glossary terms with their exact spelling and meaning, and define each term on first use using the glossary's own definition.

## Local file

`~/.agents/local/phone-page.md` holds the store command and nothing else. Read it when it exists. Its complete format is one line containing an executable name from `PATH` or an absolute executable path:

```text
page-store
```

`PHONE_PAGE_LOCAL_FILE` overrides the path for tests. The command must implement the store contract below.

## Install

1. Make `phone-page/phone-page` executable.
2. Link `phone-page/phone-page` into a directory on `PATH`, such as `~/.local/bin`.
3. Link this `phone-page` directory from the checkout into every agent CLI's skills directory, then verify that each CLI lists the skill.
4. Create the local file with the chosen store command after that store is installed.
5. Run `phone-page assets <directory>` once for the store and confirm that the store serves every downloaded file at `/assets/<name>`.

## Commands

- `phone-page publish <file> [id]` publishes a read page, or replaces page `id` in place, and prints the store's one-line JSON object containing `id` and `url`.
- `phone-page ask <spec> [id]` validates a question spec, renders it, publishes it or replaces page `id` in place, and prints the same JSON object. When publishing a new page and `HERDR_TAB_ID` is nonempty, it passes that asking tab to the store; without it, the page has no asking tab. A replaced page keeps its id, url and asking tab.
- `phone-page delete <id>` deletes the page and removes its inbox line. Exit 4 means the page was already gone; its inbox line is still removed.
- `phone-page inbox` prunes and prints the local inbox of open question pages.
- `phone-page render <spec>` validates a question spec and prints the complete HTML page.
- `phone-page answers <id>` prints the latest submission. Exit 3 means an existing page has no submission; exit 4 means the page no longer exists.
- `phone-page assets <directory>` downloads and verifies every file in `assets.json` before installing it in the directory.

`publish`, `ask`, `delete` and `answers` preserve the store command's output and exit status.

After a successful `ask`, `~/.agents/local/inbox.md` receives `- <YYYY-MM-DD HH:MM> <spec title>: <page url>`; an `ask` with an id replaces the line whose url ends in that id, and `delete` removes it. `inbox` removes answered or deleted pages and keeps entries when the store cannot be read. These commands lock the inbox with `flock` on `~/.agents/local/inbox.md.lock`; install it if unavailable. `PHONE_PAGE_INBOX_FILE` overrides the path for tests.

## Question spec

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

At widths of 960 px and above, each question starts a new screen and uses as many parts as its measured content needs, followed by a **Check your answers** review screen. Questions with context, cards or a table fill the left column, then the right, with whole blocks in reading order: heading with context, cards, table, each chart, and options with Other, note and Skip; other questions use one column. The first screen also holds the masthead; later parts start with a compact question heading. A block taller than the available screen scrolls internally on its own screen. Resizing repacks the current question and preserves a focused answer field. A fixed bottom bar shows Previous, the position (including part numbers when needed) and Next, with Review answers on the final part and no Next on review. Left and Right arrow keys navigate unless focus is in a text field, radio, checkbox or table, or a modifier key is held; a screen change scrolls to the top and focuses its heading.

Below 960 px, all questions remain on one scrolling page, followed by a live **Your answers** panel. Both summaries show each question's header, question, selected labels, Other text, skipped or unanswered state, and any note prefixed with Note:. Change opens the question on desktop or scrolls to and focuses it on a phone. Submit follows the summary, centered on desktop and full width on a phone; pressing Enter from a desktop question first opens review. An empty selected Other response keeps the existing error and focuses its field, opening the part containing that field on desktop. The masthead shows the latest submission when the page opens, with any failure to check it shown alongside. Successful submission preserves the toast and shows `Sent at` with the local time, followed by `Reply done in the chat`, below Submit.

## Store contract

The configured store command accepts `publish <file> [tab-id]` and prints exactly one JSON line containing the new page's `id` and `url`. It accepts `update <id> <file>`, which replaces that page's content in place while keeping its id, url and asking tab, discards its submission, and prints the same JSON line. It accepts `delete <id>` and removes the page and its submission. It accepts `answers <id>` and prints the latest submission JSON, exits 3 when an existing page has no submission, or exits 4 when the page no longer exists. `update` and `delete` also exit 4 when the page no longer exists. Any other failure is nonzero.

The published page can make authenticated `GET` and `POST` requests to its own URL plus `/answers`, and can load the manifest files at its own origin under `/assets/<name>`. `GET <page URL>/answers` returns HTTP 404 when no submission exists. The store's content policy allows inline `script` and `style` elements on pages. A posted submission is a JSON object no larger than 64 KiB and replaces the prior submission. Only the user after login, and the store command, can reach any route.

When `cf-page-store listen` runs on the asking machine, a submitted answer wakes the recorded herdr tab without polling. On that prompt, run `phone-page answers <id>` and continue from the stored submission.

After publishing, reply in one line that says what the page contains and what the user should do, followed by its link. After the user replies `done`, run `phone-page answers <id>` and continue from that submission. When the user asks for a change to a page they already have, rebuild it in place with the same id and reply with the same link.
