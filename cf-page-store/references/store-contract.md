# Store contract

phone-page runs the store command named in its local file to publish pages and read their submissions. `cf-page-store` implements this contract; any other store command must implement it too.

## Commands

- `publish <file> [tab-id]` publishes one HTML file and prints exactly one JSON line containing the new page's `id` and `url`. The optional tab id is the asking herdr tab.
- `update <id> <file>` replaces that page's content in place while keeping its id, url and asking tab, discards its submission, and prints the same JSON line.
- `delete <id>` removes the page and its submission.
- `answers <id>` prints the latest submission JSON, exits 3 when an existing page has no submission, or exits 4 when the page no longer exists.
- `update` and `delete` also exit 4 when the page no longer exists. Any other failure is nonzero.

## Page routes

- The published page can make authenticated `GET` and `POST` requests to its own URL plus `/answers`, and can load the manifest files at its own origin under `/assets/<name>`.
- `GET <page URL>/answers` returns HTTP 404 when no submission exists.
- The store's content policy allows inline `script` and `style` elements on pages.
- A posted submission is a JSON object no larger than 64 KiB and replaces the prior submission.
- Only the user after login, and the store command, can reach any route.
