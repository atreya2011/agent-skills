# Commands

Write page content as a read or question spec and publish it through its renderer; both commands print the store's one-line JSON object containing `id` and `url`. An optional id replaces the existing page in place.

- `phone-page read <spec> [id]` validates a [read spec](read-spec.md), renders it and publishes or replaces the page; it does not create a question inbox entry or pass an asking tab to the store.
- `phone-page publish <file> [id]` passes an already-rendered HTML file to the store, or replaces page `id` in place; author read pages through `read` rather than this low-level command.
- `phone-page ask <spec> [id]` validates a question spec, renders it, publishes it or replaces page `id` in place, and prints the same JSON object. When publishing a new page and `HERDR_TAB_ID` is nonempty, it passes that asking tab to the store; without it, the page has no asking tab. A replaced page keeps its id, url and asking tab.
- `phone-page delete <id>` deletes the page and removes its inbox line. Exit 4 means the page was already gone; its inbox line is still removed.
- `phone-page inbox` prunes and prints the local inbox of open question pages.
- `phone-page render <spec>` validates a read or question spec and prints the complete HTML page.
- `phone-page answers <id>` prints the latest submission. Exit 3 means an existing page has no submission; exit 4 means the page no longer exists.
- `phone-page assets <directory>` downloads and verifies every file in `assets.json` before installing it in the directory.
- `phone-page attach [--dry-run]` annotates the Taskwarrior task of each page's asking tab with `phone-page <id> <url>`, and with `phone-page <id> answered <time>` once the listener has delivered the answer; pages without a task go to the unmatched file. Read [tab-tasks.md](tab-tasks.md).
- `phone-page close [--dry-run]` deletes answered pages and pages whose asking tab is gone, and logs every page's status. Read [tab-tasks.md](tab-tasks.md).

`publish`, `read`, `ask`, `delete` and `answers` preserve the store command's output and exit status. `attach` and `close` run the store command's `pages` command to list pages.

The renderer identifies a read spec by `sections` and a question spec by `questions`; a spec cannot contain both. `read` and `ask` require their matching spec type.

After a successful `ask`, `~/.agents/local/inbox.md` receives `- <YYYY-MM-DD HH:MM> <spec title>: <page url>`; an `ask` with an id replaces the line whose url ends in that id, and `delete` removes it. `inbox` removes answered or deleted pages and keeps entries when the store cannot be read. These commands lock the inbox with `flock` on `~/.agents/local/inbox.md.lock`; install it if unavailable. `PHONE_PAGE_INBOX_FILE` overrides the path for tests.
