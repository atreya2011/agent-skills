---
name: cf-page-store
description: Run one private Cloudflare store for phone-page pages and their latest submissions.
---

# Cloudflare page store

`cf-page-store` is phone-page's store command. It implements the [store contract](references/store-contract.md) on a Cloudflare Worker. The Worker requires a valid Cloudflare Access token on every route, including assets. `PUT /api/pages/<id>` and `DELETE /api/pages/<id>` back the update and delete commands.
Answered pages are deleted three days after their last submission.
Unanswered pages are deleted seven days after they were last published or updated.

## Local file

`~/.agents/local/cf-page-store.md` holds every machine value. Read it when it exists. Its complete format is six lines in this order:

```text
Store address: <workers.dev URL>
Cloudflare account id: <account id>
Team domain: <full HTTPS team domain>
Audience tag: <Access application audience tag>
D1 database id: <database id>
Deploy config: <absolute path to the rendered Wrangler config>
```

The OS keyring holds three generic-password entries under service `cf-page-store`: accounts `api-token`, `service-token-id` and `service-token-secret`. No token belongs in the local file.

## Install

Read [references/install.md](references/install.md) before installing the store. Cloudflare commands run once during install; never run them in continuous integration.

## Answer notifications

When `cf-page-store listen` runs on the asking machine, a submitted answer wakes the recorded herdr tab without polling. Read [references/answer-notifications.md](references/answer-notifications.md) before updating an existing store or installing, running or debugging the listener.

## Commands

- `cf-page-store publish <file> [tab-id]`, `cf-page-store update <id> <file>`, `cf-page-store delete <id>` and `cf-page-store answers <id>` follow the [store contract](references/store-contract.md). Read it before changing a command or the Worker.
- `publish` and `update` print exactly the Worker's one-line JSON object.
- `answers` prints nothing when it exits 3. After an answers HTTP 404, it checks the page itself to distinguish exit 3 from exit 4; other read failures remain errors.
- `cf-page-store listen` maintains the answer-event WebSocket, checks it with a text heartbeat every thirty seconds, and wakes asking Herdr tabs.
- `cf-page-store pages` prints every page as one JSON array through a read-only D1 query that Wrangler runs with the `api-token` keyring entry, the account id and the deploy config from the local file. It runs from the skill directory, so `bun install --frozen-lockfile` must have run there.

The optional tab id is sent as `X-Herdr-Tab-Id`; the Worker accepts only `w` plus one or more ASCII letters or digits, then `:t` and one or more ASCII letters or digits. Invalid values return HTTP 400.

All commands read the service-token id and secret from the OS keyring and never print either value.
