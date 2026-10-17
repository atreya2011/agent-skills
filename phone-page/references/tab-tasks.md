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
- `tab-areas.v2.tsv`: columns folder prefix, area, owner; the longest matching prefix wins. A working directory with a folder whose name contains `cos`, `global-pm` or `chief-of-staff` goes to Chief of Staff before the prefixes apply.

`herdr-tab-sync` reads the read-only `herdr tab list`, `herdr agent list` and `herdr workspace list` listings by default; `--live` selects this mode explicitly. Tab listing working directories take precedence over agent working directories. `--platform` selects the Platform tab files and fails if they are missing or lack tab ids or folders.

Areas are `work`, `sidegig` or `personal`. Finance maps to area `personal`, owner `Finance`. CoS, shared, no-area and unmatched tabs map to area `personal`, owner `Chief of Staff`, with tag `triage`. Other owners come from `route area=<area>`; per-tab and label overrides beat route, with per-tab overrides taking precedence. The old owner column in folder prefix rows is not used. `TASK_BIN` and `ROUTE_BIN` override the `~/.local/bin/task` and `~/.local/bin/route` paths.

## herdr-tab-sync

`herdr-tab-sync [--live | --platform] [--dry-run]` prints one tab-separated line per tab: `source <input used>`, then `create`, `keep` or `close` with the tab id, project, area, owner and description, then a `summary` line. A tab without a pending task gets a task through `task import`; a pending task whose tab is gone gets the annotation `herdr tab <id> closed` and is marked done. Nothing is ever deleted. With `--dry-run` it only prints.

## phone-page attach

`phone-page attach [--dry-run]` lists pages with the store's `pages` command and, for each page with an asking tab and a task, adds the annotation `phone-page <id> <url>` once. When the page has a submission and the listener delivered it, it also adds `phone-page <id> answered <submitted_at>` once. A page whose tab has no task becomes a row of `unmatched.tsv` (`unmatched.dryrun.tsv` with `--dry-run`) in the state directory, with the headline, the desk from the tab's area and the page id; the file is rewritten on every run. Pages without an asking tab are only counted. It never creates tasks.

## Delivery

After successfully prompting the asking tab, the listener logs `delivered page=<id> tab=<tab> at=<ISO time>` to the user journal of `cf-page-store-listener`. A page counts as delivered only when it has a submission and that success line matches its page id and asking tab in `journalctl --user -u cf-page-store-listener`. A success line takes precedence over an earlier failure. A submission with a failure line but no matching success is undelivered; without either line, delivery is unknown. When the journal cannot be read, `attach` skips the answered annotation and `close` changes nothing.

## phone-page close

`phone-page close [--dry-run]` gives every page one status and appends one row per page to `close-log.tsv` in the state directory (`time`, `page_id`, `area`, `status`, `action`, `mode`):

| Status | Meaning | Action |
|--------|---------|--------|
| `answered` | submission delivered per the journal | delete |
| `undelivered` | submission with a failure line | none |
| `open` | no submission, asking tab still listed by `herdr tab list` | none |
| `tabgone` | no submission, asking tab gone | delete |
| `notab` | page has no asking tab | delete |
| `unknown` | no delivery evidence for a submission, or the journal or herdr could not be read | none |

A delete runs `phone-page delete <id>`, which also removes the inbox line. With `--dry-run` nothing is deleted and the log rows carry `dry-run`.
