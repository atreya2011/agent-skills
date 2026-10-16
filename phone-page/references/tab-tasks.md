# Tab tasks

Every herdr tab is one Taskwarrior task, and the tab's folder is the task's project, so a phone page can hang off the task of the tab that asked it. `herdr-tab-sync` keeps the tasks in step with the tabs; `phone-page attach` links pages to those tasks; `phone-page close` removes pages that are finished.

## Model

- A task carries the tab in the `herdrtab` user-defined attribute, so `~/.taskrc` needs `uda.herdrtab.type=string` and `uda.herdrtab.label=herdr Tab`. The sync also sets `source=herdr` and `sourceid=<tab id>`.
- The project is the last folder name of the tab's working directory, with a ` (deleted)` suffix dropped; a tab opened from the home directory has project `misc`.
- The description is `<tab label> (<workspace label>)`.
- One function, `task_for_tab`, finds a tab's task: the pending or waiting task whose `herdrtab` equals the tab id, or for `attach` a completed one when no pending task exists. Deleted tasks never match.

## Inputs

The state directory is `~/.local/state/phone-page`; `PHONE_PAGE_STATE_DIR` overrides it. Its `platform-input` folder holds:

- `tabs/<date>/*.json`: tab lists with `generated`, and `tabs[]` entries with `tab_id`, `workspace_id`, `workspace_label`, `tab_label` and `cwd`. The newest date folder is used, and for a tab in several files the newest `generated` wins.
- `tab-overrides.tsv`: header `tab area owner note`, keyed by tab id; beats every other rule.
- `tab-overrides.v2.tsv`: columns workspace label, tab label, area, owner; `#` lines are comments.
- `tab-areas.v2.tsv`: columns folder prefix, area, owner; the longest matching prefix wins. A working directory with a folder whose name contains `cos` or `global-pm` has area `cos` before the prefixes apply.

When the tab files are missing or lack tab ids or folders, the commands read `herdr tab list`, `herdr agent list` and `herdr workspace list` instead and say so in the `source` line. An owner that no override or prefix row gives comes from `route area=<area>`, or `route source=herdr` for a tab with no area. `TASK_BIN` and `ROUTE_BIN` override the `~/.local/bin/task` and `~/.local/bin/route` paths.

## herdr-tab-sync

`herdr-tab-sync [--dry-run]` prints one tab-separated line per tab: `source <input used>`, then `create`, `keep` or `close` with the tab id, project, area, owner and description, then a `summary` line. A tab without a pending task gets a task through `task import`; a pending task whose tab is gone gets the annotation `herdr tab <id> closed` and is marked done. Nothing is ever deleted. With `--dry-run` it only prints.

## phone-page attach

`phone-page attach [--dry-run]` lists pages with the store's `pages` command and, for each page with an asking tab and a task, adds the annotation `phone-page <id> <url>` once. When the page has a submission and the listener delivered it, it also adds `phone-page <id> answered <submitted_at>` once. A page whose tab has no task becomes a row of `unmatched.tsv` (`unmatched.dryrun.tsv` with `--dry-run`) in the state directory, with the headline, the desk from the tab's area and the page id; the file is rewritten on every run. Pages without an asking tab are only counted. It never creates tasks.

## Delivery

The listener logs no successful delivery; it only writes failure lines such as `no eligible agent for page "<id>" tab "<tab>"` to the user journal of `cf-page-store-listener`. A page therefore counts as delivered when it has an asking tab, a submission and no failure line naming its id in `journalctl --user -u cf-page-store-listener`. When the journal cannot be read, delivery is unknown: `attach` skips the answered annotation and `close` changes nothing.

## phone-page close

`phone-page close [--dry-run]` gives every page one status and appends one row per page to `close-log.tsv` in the state directory (`time`, `page_id`, `area`, `status`, `action`, `mode`):

| Status | Meaning | Action |
|--------|---------|--------|
| `answered` | submission delivered per the journal | delete |
| `undelivered` | submission with a failure line | none |
| `open` | no submission, asking tab still listed by `herdr tab list` | none |
| `tabgone` | no submission, asking tab gone | delete |
| `notab` | page has no asking tab | delete |
| `unknown` | the journal or herdr could not be read | none |

A delete runs `phone-page delete <id>`, which also removes the inbox line. With `--dry-run` nothing is deleted and the log rows carry `dry-run`.
