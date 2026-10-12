# Commands

- `phone-page publish <file> [id]` publishes a read page, or replaces page `id` in place, and prints the store's one-line JSON object containing `id` and `url`.
- `phone-page ask <spec> [id]` validates a question spec, renders it, publishes it or replaces page `id` in place, and prints the same JSON object. When publishing a new page and `HERDR_TAB_ID` is nonempty, it passes that asking tab to the store; without it, the page has no asking tab. A replaced page keeps its id, url and asking tab.
- `phone-page delete <id>` deletes the page and removes its inbox line. Exit 4 means the page was already gone; its inbox line is still removed.
- `phone-page inbox` prunes and prints the local inbox of open question pages.
- `phone-page render <spec>` validates a question spec and prints the complete HTML page.
- `phone-page answers <id>` prints the latest submission. Exit 3 means an existing page has no submission; exit 4 means the page no longer exists.
- `phone-page assets <directory>` downloads and verifies every file in `assets.json` before installing it in the directory.

`publish`, `ask`, `delete` and `answers` preserve the store command's output and exit status.

After a successful `ask`, `~/.agents/local/inbox.md` receives `- <YYYY-MM-DD HH:MM> <spec title>: <page url>`; an `ask` with an id replaces the line whose url ends in that id, and `delete` removes it. `inbox` removes answered or deleted pages and keeps entries when the store cannot be read. These commands lock the inbox with `flock` on `~/.agents/local/inbox.md.lock`; install it if unavailable. `PHONE_PAGE_INBOX_FILE` overrides the path for tests.
