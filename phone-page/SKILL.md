---
name: phone-page
description: Publish designed read pages and tappable question pages for replies that do not fit cleanly in chat on a phone.
---

# Phone page

Use a page when a reply to the user needs more than two sentences, a diagram, a table or any question. A chat reply that points to a page has at most two sentences plus the link, with no code block, table or list. Put every question to the user on a question page, never in reply text or the agent CLI's question dialog. Chat between agents is unaffected.

Read [references/design.md](references/design.md) before making a read page. Use `phone-page ask` for a question page; do not hand-write its form.

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
- `phone-page ask <spec>` validates a question spec, renders it, publishes it and prints the same JSON object.
- `phone-page render <spec>` validates a question spec and prints the complete HTML page.
- `phone-page answers <id>` prints the latest submission. Exit 3 means that no submission exists.
- `phone-page assets <directory>` downloads and verifies every file in `assets.json` before installing it in the directory.

`publish`, `ask` and `answers` preserve the store command's output and exit status.

## Question spec

Write one JSON object with a string `title` and 1 to 10 questions. Every question has a unique string `id`, a `header` of 1 to 12 characters, a string `question`, an optional string `context`, a Boolean `multiSelect`, and 2 to 6 options. Every option has a unique string `label`, a string `description`, and may set Boolean `recommended`; at most one option is recommended, and it must be first.

```json
{
  "title": "Choose a route",
  "questions": [
    {
      "id": "route",
      "header": "Route",
      "question": "Which route should the work take?",
      "context": "The direct route has the fewest moving parts.",
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

The configured store command accepts `publish <file>` and prints exactly one JSON line containing the new page's `id` and `url`. It accepts `answers <id>` and prints the latest submission JSON, or exits 3 when none exists. Any other failure is nonzero.

The published page can make authenticated `GET` and `POST` requests to its own URL plus `/answers`, and can load the manifest files at its own origin under `/assets/<name>`. `GET <page URL>/answers` returns HTTP 404 when no submission exists. The store's content policy allows inline `script` and `style` elements on pages. A posted submission is a JSON object no larger than 64 KiB and replaces the prior submission. Only the user after login, and the store command, can reach any route.

After publishing, reply in one line that says what the page contains and what the user should do, followed by its link. After the user replies `done`, run `phone-page answers <id>` and continue from that submission.
