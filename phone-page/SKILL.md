---
name: phone-page
description: Publish designed read pages and tappable question pages for replies that do not fit cleanly in chat on a phone.
---

# Phone page

Use a page when a reply to the user needs more than two sentences, a diagram, a table or any question. A chat reply that points to a page has at most two sentences plus the link, with no code block, table or list. Put every question to the user on a question page, never in reply text or the agent CLI's question dialog. Chat between agents is unaffected.

For read pages and question pages, open with the bottom line, then give the reasons. Write each bullet and sub-bullet as one sentence. Put numbered points in nested sub-bullets, never as numbering in bullet text. Show every process as a flow chart with bordered boxes and arrows, using branches for decisions: draw the chart on a read page or use the question spec's `flow` field on a question page. Content text is 17 px.

- A page covers one project; questions or content about different projects go on separate pages; the page title names the project.

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

- `phone-page publish <file>` publishes a read page and prints the store's one-line JSON object containing `id` and `url`.
- `phone-page ask <spec>` validates a question spec, renders it, publishes it and prints the same JSON object. When `HERDR_TAB_ID` is nonempty, it passes that asking tab to the store; without it, the page has no asking tab.
- `phone-page inbox` prunes and prints the local inbox of open question pages.
- `phone-page render <spec>` validates a question spec and prints the complete HTML page.
- `phone-page answers <id>` prints the latest submission. Exit 3 means an existing page has no submission; exit 4 means the page no longer exists.
- `phone-page assets <directory>` downloads and verifies every file in `assets.json` before installing it in the directory.

`publish`, `ask` and `answers` preserve the store command's output and exit status.

After a successful `ask`, `~/.agents/local/inbox.md` receives `- <YYYY-MM-DD HH:MM> <spec title>: <page url>`. `inbox` removes answered or deleted pages and keeps entries when the store cannot be read. Both commands lock the inbox with `flock`; install it if unavailable. `PHONE_PAGE_INBOX_FILE` overrides the path for tests.
When a new page replaces an open page, delete the old page's line from the inbox while holding the same exclusive lock on `~/.agents/local/inbox.md.lock`.

## Question spec

Write one JSON object with a string `title` and 1 to 10 questions. Every question has a unique string `id`, a `header` of 1 to 12 characters, a string `question`, an optional `context` array of 1 or more items, an optional `flow` array of 2 or more steps, a Boolean `multiSelect`, and 2 to 6 options. The first context bullet states the bottom line; any remaining bullets give the reasons. Every option has a unique string `label`, a string `description`, and may set Boolean `recommended`; at most one option is recommended, and it must be first.

Each context item is a one-sentence string or an object `{"text": "<one sentence>", "items": ["<one sentence>", ...]}` with one or more nested sub-bullets. Put numbered points in `items` without numbering their text. Each flow step is a one-sentence string or an object `{"text": "<one sentence decision>", "branches": ["<one sentence outcome>", ...]}` with 2 to 4 branches. The chart has a small Flow label, bordered boxes and down arrows; decision outcomes sit side by side or stack beside a shared connector when the screen is too narrow.

Context bullets, sub-bullets, flow steps, branch outcomes and option descriptions each contain one sentence. Validation rejects a sentence break at `.`, `!` or `?` followed by whitespace and an uppercase letter or digit; decimals such as `0.60` and versions such as `v1.3.0` stay valid. Context bullets and sub-bullets cannot start with list numbering such as `1.` or `2)`. Errors name the offending text.

```json
{
  "title": "Choose a route",
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
      "flow": [
        "Make the smallest change.",
        {
          "text": "Did the checks pass?",
          "branches": ["Deliver the passing result.", "Repair the failing result."]
        },
        "Record the result."
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

## Store contract

The configured store command accepts `publish <file> [tab-id]` and prints exactly one JSON line containing the new page's `id` and `url`. It accepts `answers <id>` and prints the latest submission JSON, exits 3 when an existing page has no submission, or exits 4 when the page no longer exists. Any other failure is nonzero.

The published page can make authenticated `GET` and `POST` requests to its own URL plus `/answers`, and can load the manifest files at its own origin under `/assets/<name>`. `GET <page URL>/answers` returns HTTP 404 when no submission exists. The store's content policy allows inline `script` and `style` elements on pages. A posted submission is a JSON object no larger than 64 KiB and replaces the prior submission. Only the user after login, and the store command, can reach any route.

When cf-page-store has its optional Discord webhook secret and herdr-connect is configured, a submitted answer wakes the asking herdr tab without polling. On that prompt, run `phone-page answers <id>` and continue from the stored submission.

After publishing, reply in one line that says what the page contains and what the user should do, followed by its link. After the user replies `done`, run `phone-page answers <id>` and continue from that submission.
