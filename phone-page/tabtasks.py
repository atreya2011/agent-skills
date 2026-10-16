#!/usr/bin/env python3
import argparse
import csv
import datetime
import json
import os
import re
import subprocess
import sys

HERDR_SOURCE = "herdr"
MISC_PROJECT = "misc"
DELETED_SUFFIX = " (deleted)"
COS_SEGMENTS = ("cos", "global-pm")
PAGE_ID_PATTERN = re.compile(r'page "([^"]+)"')
TAB_ID_PATTERN = re.compile(r"^w[A-Za-z0-9]+:t[A-Za-z0-9]+$")
UNMATCHED_HEADER = ("headline", "desk", "page_id")
CLOSE_LOG_HEADER = ("time", "page_id", "area", "status", "action", "mode")


class CommandError(Exception):
    pass


class Tab:
    def __init__(self, tab_id, workspace_id, workspace_label, tab_label, cwd):
        self.tab_id = tab_id
        self.workspace_id = workspace_id
        self.workspace_label = workspace_label
        self.tab_label = tab_label
        self.cwd = cwd


def home():
    return os.path.expanduser("~")


def state_dir():
    return os.environ.get("PHONE_PAGE_STATE_DIR") or os.path.join(
        os.environ.get("XDG_STATE_HOME") or os.path.join(home(), ".local", "state"), "phone-page"
    )


def platform_dir():
    return os.path.join(state_dir(), "platform-input")


def task_bin():
    return os.environ.get("TASK_BIN") or os.path.join(home(), ".local", "bin", "task")


def route_bin():
    return os.environ.get("ROUTE_BIN") or os.path.join(home(), ".local", "bin", "route")


def run(argv, stdin=None):
    try:
        completed = subprocess.run(argv, input=stdin, capture_output=True, text=True)
    except OSError as exc:
        raise CommandError(f"cannot run {argv[0]}: {exc.strerror}") from exc
    if completed.returncode != 0:
        detail = completed.stderr.strip().splitlines()
        raise CommandError(f"{' '.join(argv[:2])} exited {completed.returncode}: {detail[-1] if detail else ''}")
    return completed.stdout


def now():
    return datetime.datetime.now().astimezone()


def parse_timestamp(value):
    return datetime.datetime.fromisoformat(value.replace("Z", "+00:00"))


def read_rows(path, header=False):
    if not os.path.exists(path):
        return []
    with open(path, encoding="utf-8", newline="") as handle:
        lines = [line.rstrip("\r\n") for line in handle]
    rows = [line.split("\t") for line in lines if line.strip() and not line.lstrip().startswith("#")]
    return rows[1:] if header and rows else rows


def column(row, index):
    return row[index].strip() if index < len(row) else ""


class Rules:
    def __init__(self, directory):
        self.by_tab = {}
        for row in read_rows(os.path.join(directory, "tab-overrides.tsv"), header=True):
            self.by_tab[column(row, 0)] = (column(row, 1), column(row, 2))
        self.by_label = {}
        for row in read_rows(os.path.join(directory, "tab-overrides.v2.tsv")):
            self.by_label[(column(row, 0), column(row, 1))] = (column(row, 2), column(row, 3))
        self.prefixes = []
        for row in read_rows(os.path.join(directory, "tab-areas.v2.tsv")):
            self.prefixes.append((column(row, 0), column(row, 1), column(row, 2)))
        self.prefixes.sort(key=lambda rule: len(rule[0]), reverse=True)

    def folder_rule(self, cwd):
        if any(any(mark in segment for mark in COS_SEGMENTS) for segment in cwd.split("/")):
            return "cos", ""
        for prefix, area, owner in self.prefixes:
            if cwd.startswith(prefix) or cwd + "/" == prefix:
                return area, owner
        return "", ""

    def classify(self, tab):
        area, owner = self.folder_rule(tab.cwd)
        override = self.by_tab.get(tab.tab_id) or self.by_label.get((tab.workspace_label, tab.tab_label))
        if override and override[0]:
            return override
        if override and override[1]:
            return area, override[1]
        return area, owner


def route_owner(area):
    terms = [f"area={area}"] if area else [f"source={HERDR_SOURCE}"]
    return run([route_bin(), *terms]).strip()


def owner_for(area, owner):
    return owner or route_owner(area)


def project_for(cwd):
    path = cwd[: -len(DELETED_SUFFIX)] if cwd.endswith(DELETED_SUFFIX) else cwd
    path = path.rstrip("/")
    if not path or path == home():
        return MISC_PROJECT
    return os.path.basename(path)


def load_platform_tabs(directory):
    root = os.path.join(directory, "tabs")
    if not os.path.isdir(root):
        return None, "no platform tab files"
    dates = sorted(entry for entry in os.listdir(root) if os.path.isdir(os.path.join(root, entry)))
    if not dates:
        return None, "no platform tab files"
    day = os.path.join(root, dates[-1])
    files = sorted(name for name in os.listdir(day) if name.endswith(".json"))
    tabs = {}
    generated = {}
    stamps = []
    for name in files:
        with open(os.path.join(day, name), encoding="utf-8") as handle:
            document = json.load(handle)
        stamp = document.get("generated") or ""
        if stamp:
            stamps.append(stamp)
        for entry in document.get("tabs") or []:
            tab_id = entry.get("tab_id") or ""
            cwd = entry.get("cwd") or ""
            if not TAB_ID_PATTERN.match(tab_id) or not cwd:
                return None, f"platform file {name} lacks tab ids or folders"
            if tab_id in generated and generated[tab_id] >= stamp:
                continue
            generated[tab_id] = stamp
            tabs[tab_id] = Tab(tab_id, entry.get("workspace_id") or "", entry.get("workspace_label") or "", entry.get("tab_label") or "", cwd)
    if not tabs:
        return None, "no platform tab files"
    stamps.sort()
    age = ""
    if stamps:
        hours = (now() - parse_timestamp(stamps[-1])).total_seconds() / 3600
        age = f", generated {stamps[0]}..{stamps[-1]}, age {hours:.1f}h"
    return tabs, f"platform files {day} ({len(files)} files{age})"


def herdr_json(*words):
    return json.loads(run(["herdr", *words]))["result"]


def load_herdr_tabs():
    workspaces = {entry["workspace_id"]: entry.get("label") or "" for entry in herdr_json("workspace", "list")["workspaces"]}
    cwds = {}
    for agent in herdr_json("agent", "list")["agents"]:
        cwds.setdefault(agent["tab_id"], agent.get("cwd") or "")
    tabs = {}
    for entry in herdr_json("tab", "list")["tabs"]:
        tab_id = entry["tab_id"]
        tabs[tab_id] = Tab(tab_id, entry.get("workspace_id") or "", workspaces.get(entry.get("workspace_id"), ""), entry.get("label") or "", cwds.get(tab_id, ""))
    return tabs


def load_tabs(directory):
    tabs, source = load_platform_tabs(directory)
    if tabs is not None:
        return tabs, source
    return load_herdr_tabs(), f"herdr live listing ({source})"


def load_tabs_if_available(directory):
    try:
        return load_tabs(directory)[0]
    except CommandError:
        return {}


def live_tab_ids():
    return {entry["tab_id"] for entry in herdr_json("tab", "list")["tabs"]}


def export_tasks():
    output = run([task_bin(), "rc.verbose=nothing", "rc.hooks=off", "export"])
    return json.loads(output or "[]")


def task_for_tab(tasks, tab_id, pending_only):
    pending = None
    completed = None
    for task in tasks:
        if task.get("herdrtab") != tab_id:
            continue
        status = task.get("status")
        if status in ("pending", "waiting"):
            pending = pending or task
        elif status == "completed" and not pending_only:
            completed = completed or task
    return pending or completed


def annotations(task):
    return [entry.get("description", "") for entry in task.get("annotations") or []]


def task_import(fields):
    run([task_bin(), "rc.verbose=nothing", "rc.confirmation=off", "import", "-"], stdin=json.dumps(fields))


def task_done(uuid):
    run([task_bin(), "rc.verbose=nothing", "rc.confirmation=off", uuid, "done"])


def task_annotate(uuid, text):
    run([task_bin(), "rc.verbose=nothing", "rc.confirmation=off", uuid, "annotate", text])


def tab_task_fields(tab, area, owner):
    return {
        "description": f"{tab.tab_label} ({tab.workspace_label})",
        "project": project_for(tab.cwd),
        "herdrtab": tab.tab_id,
        "source": HERDR_SOURCE,
        "sourceid": tab.tab_id,
        "area": area,
        "owner": owner,
        "status": "pending",
    }


def emit(*values):
    print("\t".join(str(value) for value in values))


def command_sync(args):
    directory = platform_dir()
    tabs, source = load_tabs(directory)
    rules = Rules(directory)
    tasks = export_tasks()
    emit("source", source)
    created = 0
    closed = 0
    for tab in sorted(tabs.values(), key=lambda entry: entry.tab_id):
        area, owner = rules.classify(tab)
        owner = owner_for(area, owner)
        fields = tab_task_fields(tab, area, owner)
        if task_for_tab(tasks, tab.tab_id, pending_only=True):
            emit("keep", tab.tab_id, fields["project"], area, owner, fields["description"])
            continue
        created += 1
        emit("create", tab.tab_id, fields["project"], area, owner, fields["description"])
        if not args.dry_run:
            task_import(fields)
    for task in tasks:
        tab_id = task.get("herdrtab")
        if not tab_id or task.get("status") not in ("pending", "waiting") or tab_id in tabs:
            continue
        closed += 1
        emit("close", tab_id, task.get("project") or "", task.get("area") or "", task.get("owner") or "", task.get("description") or "")
        if not args.dry_run:
            task_annotate(task["uuid"], f"herdr tab {tab_id} closed")
            task_done(task["uuid"])
    emit("summary", f"create={created}", f"close={closed}", "mode=" + mode(args))
    return 0


def mode(args):
    return "dry-run" if args.dry_run else "live"


def store_command():
    command = os.environ.get("PHONE_PAGE_STORE_COMMAND")
    if not command:
        raise CommandError("PHONE_PAGE_STORE_COMMAND is not set")
    return command


def load_pages():
    pages = json.loads(run([store_command(), "pages"]) or "[]")
    for page in pages:
        page["herdr_tab_id"] = page.get("herdr_tab_id") or None
        page["submitted_at"] = page.get("submitted_at") or None
    return pages


def failed_page_ids():
    output = run(["journalctl", "--user", "-u", "cf-page-store-listener", "-o", "cat", "--no-pager"])
    return set(PAGE_ID_PATTERN.findall(output))


def delivered(page, failed):
    return bool(page["submitted_at"] and page["herdr_tab_id"] and page["id"] not in failed)


def write_tsv(path, header, rows):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8", newline="") as handle:
        writer = csv.writer(handle, delimiter="\t", lineterminator="\n")
        writer.writerow(header)
        writer.writerows(rows)


def append_tsv(path, header, rows):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    new = not os.path.exists(path) or os.path.getsize(path) == 0
    with open(path, "a", encoding="utf-8", newline="") as handle:
        writer = csv.writer(handle, delimiter="\t", lineterminator="\n")
        if new:
            writer.writerow(header)
        writer.writerows(rows)


def tab_area(tabs, rules, tab_id):
    tab = tabs.get(tab_id)
    return rules.classify(tab)[0] if tab else ""


def desk_for(tabs, rules, tab_id):
    tab = tabs.get(tab_id)
    if not tab:
        return route_owner("")
    area, owner = rules.classify(tab)
    return owner_for(area, owner)


def ensure_annotation(task, text, dry_run):
    if text in annotations(task):
        return False
    if not dry_run:
        task_annotate(task["uuid"], text)
    task.setdefault("annotations", []).append({"description": text})
    return True


def command_attach(args):
    directory = platform_dir()
    pages = load_pages()
    tasks = export_tasks()
    rules = Rules(directory)
    tabs = load_tabs_if_available(directory)
    try:
        failed = failed_page_ids()
    except CommandError as exc:
        failed = None
        emit("listener-log", "unknown", str(exc))
    matched = 0
    notab = 0
    unmatched = []
    for page in pages:
        if not page["herdr_tab_id"]:
            notab += 1
            emit("no-tab", page["id"], page["title"])
            continue
        task = task_for_tab(tasks, page["herdr_tab_id"], pending_only=False)
        if task is None:
            desk = desk_for(tabs, rules, page["herdr_tab_id"])
            unmatched.append((page["title"], desk, page["id"]))
            emit("unmatched", page["id"], desk, page["title"])
            continue
        matched += 1
        action = "annotate" if ensure_annotation(task, f"phone-page {page['id']} {page['url']}", args.dry_run) else "keep"
        if failed is not None and delivered(page, failed):
            if ensure_annotation(task, f"phone-page {page['id']} answered {page['submitted_at']}", args.dry_run):
                action += "+answered"
        emit("matched", page["id"], task["uuid"], action, page["title"])
    name = "unmatched.dryrun.tsv" if args.dry_run else "unmatched.tsv"
    write_tsv(os.path.join(state_dir(), name), UNMATCHED_HEADER, unmatched)
    emit("summary", f"matched={matched}", f"unmatched={len(unmatched)}", f"no-tab={notab}", "mode=" + mode(args))
    return 0


def page_status(page, failed, tab_ids):
    if failed is None or tab_ids is None:
        return "unknown"
    if not page["herdr_tab_id"]:
        return "notab"
    if page["submitted_at"]:
        return "answered" if page["id"] not in failed else "undelivered"
    return "open" if page["herdr_tab_id"] in tab_ids else "tabgone"


def delete_page(page_id):
    script = os.path.join(os.path.dirname(os.path.abspath(__file__)), "phone-page")
    run([script, "delete", page_id])


def command_close(args):
    directory = platform_dir()
    pages = load_pages()
    rules = Rules(directory)
    tabs = load_tabs_if_available(directory)
    failed = None
    tab_ids = None
    try:
        failed = failed_page_ids()
        tab_ids = live_tab_ids()
    except CommandError as exc:
        emit("lookup", "unknown", str(exc))
    rows = []
    counts = {}
    stamp = now().isoformat(timespec="seconds")
    for page in pages:
        status = page_status(page, failed, tab_ids)
        action = "delete" if status in ("answered", "tabgone") else "none"
        counts[(status, action)] = counts.get((status, action), 0) + 1
        area = tab_area(tabs, rules, page["herdr_tab_id"]) if page["herdr_tab_id"] else ""
        rows.append((stamp, page["id"], area, status, action, mode(args)))
        emit(action, page["id"], status, area, page["title"])
        if action == "delete" and not args.dry_run:
            delete_page(page["id"])
    append_tsv(os.path.join(state_dir(), "close-log.tsv"), CLOSE_LOG_HEADER, rows)
    emit("summary", *[f"{status}/{action}={count}" for (status, action), count in sorted(counts.items())], "mode=" + mode(args))
    return 0


def main(argv=None):
    parser = argparse.ArgumentParser(prog="tabtasks")
    commands = parser.add_subparsers(dest="command", required=True)
    for name, handler in (("sync", command_sync), ("attach", command_attach), ("close", command_close)):
        sub = commands.add_parser(name)
        sub.add_argument("--dry-run", action="store_true")
        sub.set_defaults(handler=handler)
    args = parser.parse_args(argv)
    try:
        return args.handler(args)
    except (CommandError, json.JSONDecodeError, KeyError, OSError) as exc:
        print(f"{args.command}: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
