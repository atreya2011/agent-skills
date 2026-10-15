# Answer notifications

Run every command in this file from the `cf-page-store` directory.

Before updating an existing store, apply `migrations/0002_herdr_tab_id.sql` once with `bunx --bun wrangler d1 execute "$DATABASE_NAME" --remote --file migrations/0002_herdr_tab_id.sql --config "$DEPLOY_CONFIG"`. Existing pages keep a null asking tab.

The `EVENTS` binding points to one SQLite-backed `PageEvents` Durable Object. Its Wrangler migration is `v1` with `new_sqlite_classes: ["PageEvents"]`, separate from the D1 migration above. For an existing rendered deploy config, copy these two sections from the template before deploying:

```sh
jq --slurpfile template wrangler.template.jsonc \
  '.durable_objects = $template[0].durable_objects | .migrations = $template[0].migrations' \
  "$DEPLOY_CONFIG" >"$DEPLOY_CONFIG.new"
mv "$DEPLOY_CONFIG.new" "$DEPLOY_CONFIG"
bun install --frozen-lockfile
bunx --bun wrangler deploy --config "$DEPLOY_CONFIG"
```

`GET /api/events` upgrades to a WebSocket behind the same Cloudflare Access authentication as the other API routes. After storing an answer for a page with an asking tab, the Worker schedules `{id, page_id, tab_id}` for background publication to every listener and returns HTTP 204 without waiting for publication. Publication failures are logged and do not fail the saved answer. Events contain no titles or answers, persist in SQLite until a listener sends `{"ack": "<event-id>"}`, and replay on every new connection. Delivery can repeat if a listener disconnects before its acknowledgment arrives.

Run `cf-page-store listen` with the latest Bun release, `herdr`, and the existing service-token keyring entries available. The listener uses `ws` to send Access headers, validates event identifiers, and prompts the single matching agent pane when its state is idle, done, or working. If no eligible pane exists, it logs the page and tab instead. It acknowledges each handled event and reconnects with backoff from one to sixty seconds. Credentials never appear in its output.

Every thirty seconds after connecting, the listener sends the text message `ping`. The Durable Object automatically replies with the text message `pong` through `setWebSocketAutoResponse`, including while hibernating. If no `pong` arrives within ten seconds, the listener terminates the connection and reconnects through its existing backoff and replay flow. Pong messages do not enter event processing. These are application text messages because the Cloudflare edge can answer WebSocket protocol pings before they reach the Durable Object. Deploy the updated Worker before restarting the updated listener.

On Linux, install the user service after the command is linked into `PATH`:

- Check the executable locations from `command -v bun herdr` against the service unit's configured `PATH`.
- If either executable is outside that `PATH`, run `systemctl --user edit cf-page-store-listener.service` and add a `[Service]` section with `Environment=PATH=<bun-directory>:<herdr-directory>:<existing-service-PATH>`, including both executable directories and the existing service `PATH`.

```sh
mkdir -p "$HOME/.config/systemd/user"
# If Bun is outside the service PATH, link it into ~/.local/bin first.
install -m 0644 cf-page-store-listener.service "$HOME/.config/systemd/user/"
systemctl --user daemon-reload
systemctl --user enable --now cf-page-store-listener.service
```

Link the Bun executable with `ln -s "$(command -v bun)" "$HOME/.local/bin/bun"` when that destination does not exist. The service resolves `cf-page-store` through `%h/.local/bin` and needs `bun`, `herdr`, and `secret-tool` on its configured `PATH`, plus an unlocked keyring. Its `Restart=always` setting restarts a listener that exits.
