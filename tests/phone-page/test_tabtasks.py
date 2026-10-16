#!/usr/bin/env python3
import csv
import importlib.util
import json
import os
import stat
import subprocess
import sys
import tempfile
import unittest

REPO = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
PHONE_PAGE = os.path.join(REPO, "phone-page", "phone-page")
HERDR_TAB_SYNC = os.path.join(REPO, "phone-page", "herdr-tab-sync")
MODULE_PATH = os.path.join(REPO, "phone-page", "tabtasks.py")

FAKE_TASK = """#!/usr/bin/env python3
import json, os, sys, uuid
if os.environ.get("FAKE_TASK_FAIL"):
    sys.exit(1)
state = os.environ["FAKE_TASK_STATE"]
tasks = json.load(open(state)) if os.path.exists(state) else []
args = [a for a in sys.argv[1:] if not a.startswith("rc.")]
with open(os.environ["FAKE_TASK_LOG"], "a") as log:
    log.write(" ".join(args) + "\\n")
if args == ["export"]:
    print(json.dumps(tasks))
    sys.exit(0)
if args == ["import", "-"]:
    fields = json.loads(sys.stdin.read())
    fields.setdefault("uuid", str(uuid.uuid4()))
    tasks.append(fields)
elif len(args) == 2 and args[1] == "done":
    [t for t in tasks if t["uuid"] == args[0]][0]["status"] = "completed"
elif len(args) == 3 and args[1] == "annotate":
    [t for t in tasks if t["uuid"] == args[0]][0].setdefault("annotations", []).append({"description": args[2]})
else:
    sys.exit(2)
json.dump(tasks, open(state, "w"))
"""

FAKE_HERDR = """#!/usr/bin/env python3
import os, sys
if os.environ.get("FAKE_HERDR_FAIL"):
    sys.exit(1)
path = os.path.join(os.environ["FAKE_HERDR_DIR"], "-".join(sys.argv[1:]) + ".json")
sys.stdout.write(open(path).read())
"""

FAKE_JOURNALCTL = """#!/usr/bin/env python3
import os, sys
if os.environ.get("FAKE_JOURNAL_FAIL"):
    sys.exit(1)
sys.stdout.write(open(os.environ["FAKE_JOURNAL_FILE"]).read())
"""

FAKE_ROUTE = """#!/usr/bin/env python3
import sys
for word in sys.argv[1:]:
    if word.startswith("area=") and word[5:]:
        print("Desk " + word[5:])
        sys.exit(0)
print("Chief of Staff")
"""

FAKE_STORE = """#!/usr/bin/env python3
import os, sys
if sys.argv[1] == "pages":
    sys.stdout.write(open(os.environ["FAKE_STORE_PAGES"]).read())
elif sys.argv[1] == "delete":
    with open(os.environ["FAKE_STORE_DELETES"], "a") as log:
        log.write(sys.argv[2] + "\\n")
else:
    sys.exit(2)
"""

AREAS = "\t".join(["/work/repos/", "work", "Work Eng"]) + "\n" + "\t".join(["/work/repos/deep/", "personal", "Me"]) + "\n"
OVERRIDES = "tab\tarea\towner\tnote\nw2:t1\tsidegig\tSide\tmanual\n"
OVERRIDES_V2 = "gamma\tspecial\tfinance\tFin\n"


def load_module():
    spec = importlib.util.spec_from_file_location("tabtasks", MODULE_PATH)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def write(path, content, executable=False):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as handle:
        handle.write(content)
    if executable:
        os.chmod(path, os.stat(path).st_mode | stat.S_IXUSR)


def tab(tab_id, cwd, workspace="ws", label="tab"):
    return {"tab_id": tab_id, "workspace_id": tab_id.split(":")[0], "workspace_label": workspace, "tab_label": label, "cwd": cwd}


def page(page_id, tab_id, submitted_at=None, title="Page"):
    return {"id": page_id, "title": title, "created_at": "2026-10-10 01:02:03", "herdr_tab_id": tab_id, "submitted_at": submitted_at, "url": f"https://store.example/p/{page_id}"}


def rows(path):
    with open(path, encoding="utf-8", newline="") as handle:
        return list(csv.reader(handle, delimiter="\t"))


class Fixture:
    def __init__(self):
        self.root = tempfile.mkdtemp()
        self.home = os.path.join(self.root, "home")
        self.bin = os.path.join(self.root, "bin")
        self.state = os.path.join(self.home, ".local", "state", "phone-page")
        self.platform = os.path.join(self.state, "platform-input")
        self.herdr_dir = os.path.join(self.root, "herdr")
        self.task_state = os.path.join(self.root, "tasks.json")
        self.task_log = os.path.join(self.root, "task.log")
        self.journal = os.path.join(self.root, "journal.log")
        self.pages = os.path.join(self.root, "pages.json")
        self.deletes = os.path.join(self.root, "deletes.log")
        self.inbox = os.path.join(self.root, "inbox.md")
        write(os.path.join(self.home, ".local", "bin", "task"), FAKE_TASK, executable=True)
        write(os.path.join(self.home, ".local", "bin", "route"), FAKE_ROUTE, executable=True)
        write(os.path.join(self.bin, "herdr"), FAKE_HERDR, executable=True)
        write(os.path.join(self.bin, "journalctl"), FAKE_JOURNALCTL, executable=True)
        write(os.path.join(self.bin, "store"), FAKE_STORE, executable=True)
        write(os.path.join(self.home, ".agents", "local", "phone-page.md"), os.path.join(self.bin, "store") + "\n")
        write(os.path.join(self.platform, "tab-areas.v2.tsv"), AREAS)
        write(os.path.join(self.platform, "tab-overrides.tsv"), OVERRIDES)
        write(os.path.join(self.platform, "tab-overrides.v2.tsv"), OVERRIDES_V2)
        os.makedirs(self.herdr_dir)
        write(self.journal, "")
        write(self.pages, "[]")
        self.set_tasks([])
        self.extra_env = {}

    def set_tasks(self, tasks):
        write(self.task_state, json.dumps(tasks))

    def tasks(self):
        with open(self.task_state, encoding="utf-8") as handle:
            return json.load(handle)

    def set_platform_tabs(self, tabs, generated="2026-10-16T09:00:00+09:00", name="work.json"):
        write(os.path.join(self.platform, "tabs", "2026-10-16", name), json.dumps({"generated": generated, "tabs": tabs}))

    def set_herdr(self, tabs, cwds=None):
        cwds = cwds or {}
        write(os.path.join(self.herdr_dir, "tab-list.json"), json.dumps({"result": {"tabs": [{"tab_id": t["tab_id"], "workspace_id": t["workspace_id"], "label": t["tab_label"]} for t in tabs]}}))
        write(os.path.join(self.herdr_dir, "workspace-list.json"), json.dumps({"result": {"workspaces": [{"workspace_id": t["workspace_id"], "label": t["workspace_label"]} for t in tabs]}}))
        write(os.path.join(self.herdr_dir, "agent-list.json"), json.dumps({"result": {"agents": [{"tab_id": t["tab_id"], "cwd": cwds.get(t["tab_id"], t["cwd"])} for t in tabs]}}))

    def set_pages(self, pages):
        write(self.pages, json.dumps(pages))

    def set_journal(self, text):
        write(self.journal, text)

    def env(self):
        env = dict(os.environ)
        env.update({
            "HOME": self.home,
            "PATH": self.bin + os.pathsep + env.get("PATH", ""),
            "FAKE_TASK_STATE": self.task_state,
            "FAKE_TASK_LOG": self.task_log,
            "FAKE_HERDR_DIR": self.herdr_dir,
            "FAKE_JOURNAL_FILE": self.journal,
            "FAKE_STORE_PAGES": self.pages,
            "FAKE_STORE_DELETES": self.deletes,
            "PHONE_PAGE_INBOX_FILE": self.inbox,
        })
        for key in ("FAKE_TASK_FAIL", "FAKE_HERDR_FAIL", "FAKE_JOURNAL_FAIL", "TASK_BIN", "ROUTE_BIN", "PHONE_PAGE_STATE_DIR"):
            env.pop(key, None)
        env.update(self.extra_env)
        return env

    def run(self, argv):
        completed = subprocess.run(argv, env=self.env(), capture_output=True, text=True)
        lines = [line.split("\t") for line in completed.stdout.splitlines()]
        return completed.returncode, lines, completed.stderr

    def deleted(self):
        if not os.path.exists(self.deletes):
            return []
        with open(self.deletes, encoding="utf-8") as handle:
            return handle.read().split()


def by_kind(lines, kind):
    return [line for line in lines if line[0] == kind]


def summary(lines):
    return dict(part.split("=", 1) for part in by_kind(lines, "summary")[0][1:])


class PureFunctionTests(unittest.TestCase):
    def setUp(self):
        self.module = load_module()

    def test_task_for_tab(self):
        pending = {"uuid": "p", "herdrtab": "w1:t1", "status": "pending"}
        waiting = {"uuid": "w", "herdrtab": "w1:t1", "status": "waiting"}
        completed = {"uuid": "c", "herdrtab": "w1:t1", "status": "completed"}
        deleted = {"uuid": "d", "herdrtab": "w1:t1", "status": "deleted"}
        cases = [
            ("pending wins over completed", [completed, pending], "w1:t1", True, "p"),
            ("waiting counts as pending", [waiting], "w1:t1", True, "w"),
            ("completed ignored when pending only", [completed], "w1:t1", True, None),
            ("completed used for attach", [completed], "w1:t1", False, "c"),
            ("deleted never matches", [deleted], "w1:t1", False, None),
            ("other tab never matches", [pending], "w1:t2", False, None),
            ("no herdrtab never matches", [{"uuid": "x", "status": "pending"}], "w1:t1", False, None),
        ]
        for name, tasks, tab_id, pending_only, expected in cases:
            with self.subTest(name):
                task = self.module.task_for_tab(tasks, tab_id, pending_only)
                self.assertEqual(task["uuid"] if task else None, expected)

    def test_project_for(self):
        home = os.path.expanduser("~")
        cases = [
            ("repo folder", "/work/repos/alpha", "alpha"),
            ("trailing slash", "/work/repos/alpha/", "alpha"),
            ("deleted folder", "/work/repos/beta (deleted)", "beta"),
            ("home folder", home, "misc"),
            ("home with slash", home + "/", "misc"),
            ("empty", "", "misc"),
        ]
        for name, cwd, expected in cases:
            with self.subTest(name):
                self.assertEqual(self.module.project_for(cwd), expected)

    def test_page_status(self):
        failed = {"bad"}
        tabs = {"w1:t1"}
        cases = [
            ("answered", page("ok", "w1:t1", "2026-10-11 00:00:00"), failed, tabs, "answered"),
            ("undelivered", page("bad", "w1:t1", "2026-10-11 00:00:00"), failed, tabs, "undelivered"),
            ("open", page("o", "w1:t1"), failed, tabs, "open"),
            ("tab gone", page("g", "w9:t9"), failed, tabs, "tabgone"),
            ("no tab", page("n", None, "2026-10-11 00:00:00"), failed, tabs, "notab"),
            ("journal unknown", page("ok", "w1:t1", "2026-10-11 00:00:00"), None, tabs, "unknown"),
            ("tabs unknown", page("g", "w9:t9"), failed, None, "unknown"),
        ]
        for name, entry, failed_ids, tab_ids, expected in cases:
            with self.subTest(name):
                self.assertEqual(self.module.page_status(entry, failed_ids, tab_ids), expected)


class SyncTests(unittest.TestCase):
    def setUp(self):
        self.fixture = Fixture()
        self.home = self.fixture.home
        self.tabs = [
            tab("w1:t1", "/work/repos/alpha", "alpha", "alpha-dev"),
            tab("w1:t2", self.home, "alpha", "home-tab"),
            tab("w2:t1", "/work/repos/beta (deleted)", "beta", "beta-dev"),
            tab("w3:t1", "/other/cos-lab", "lab", "lab-tab"),
            tab("w3:t2", "/nowhere", "gamma", "special"),
            tab("w4:t1", "/work/repos/deep/delta", "delta", "delta-dev"),
        ]
        self.fixture.set_platform_tabs(self.tabs)
        self.fixture.set_tasks([
            {"uuid": "keep", "description": "kept", "herdrtab": "w1:t1", "status": "pending", "project": "alpha"},
            {"uuid": "gone", "description": "gone", "herdrtab": "w9:t9", "status": "pending", "project": "old", "area": "work", "owner": "Work Eng"},
            {"uuid": "old", "description": "old", "herdrtab": "w1:t2", "status": "completed"},
            {"uuid": "plain", "description": "plain", "status": "pending"},
        ])

    def test_plan_from_platform_files(self):
        code, lines, stderr = self.fixture.run([HERDR_TAB_SYNC, "--dry-run"])
        self.assertEqual(code, 0, stderr)
        self.assertTrue(by_kind(lines, "source")[0][1].startswith("platform files "))
        expected = {
            "w1:t1": ("keep", "alpha", "work", "Work Eng", "alpha-dev (alpha)"),
            "w1:t2": ("create", "misc", "", "Chief of Staff", "home-tab (alpha)"),
            "w2:t1": ("create", "beta", "sidegig", "Side", "beta-dev (beta)"),
            "w3:t1": ("create", "cos-lab", "cos", "Desk cos", "lab-tab (lab)"),
            "w3:t2": ("create", "nowhere", "finance", "Fin", "special (gamma)"),
            "w4:t1": ("create", "delta", "personal", "Me", "delta-dev (delta)"),
        }
        for tab_id, (kind, project, area, owner, description) in expected.items():
            with self.subTest(tab_id):
                line = [line for line in lines if len(line) > 1 and line[1] == tab_id and line[0] in ("create", "keep")][0]
                self.assertEqual(line, [kind, tab_id, project, area, owner, description])
        self.assertEqual(by_kind(lines, "close"), [["close", "w9:t9", "old", "work", "Work Eng", "gone"]])
        self.assertEqual(summary(lines), {"create": "5", "close": "1", "mode": "dry-run"})
        self.assertEqual([t["uuid"] for t in self.fixture.tasks()], ["keep", "gone", "old", "plain"])
        with open(self.fixture.task_log, encoding="utf-8") as handle:
            self.assertEqual(handle.read().split("\n")[:-1], ["export"])

    def test_live_run_writes_and_is_idempotent(self):
        code, lines, stderr = self.fixture.run([HERDR_TAB_SYNC])
        self.assertEqual(code, 0, stderr)
        self.assertEqual(summary(lines), {"create": "5", "close": "1", "mode": "live"})
        tasks = {t["uuid"]: t for t in self.fixture.tasks()}
        created = {t["herdrtab"]: t for t in tasks.values() if t.get("source") == "herdr"}
        self.assertEqual(sorted(created), ["w1:t2", "w2:t1", "w3:t1", "w3:t2", "w4:t1"])
        self.assertEqual(created["w2:t1"], {**created["w2:t1"], "description": "beta-dev (beta)", "project": "beta", "sourceid": "w2:t1", "area": "sidegig", "owner": "Side", "status": "pending"})
        self.assertEqual(tasks["gone"]["status"], "completed")
        self.assertEqual(tasks["gone"]["annotations"], [{"description": "herdr tab w9:t9 closed"}])
        self.assertEqual(tasks["old"]["status"], "completed")
        code, lines, stderr = self.fixture.run([HERDR_TAB_SYNC])
        self.assertEqual(code, 0, stderr)
        self.assertEqual(summary(lines), {"create": "0", "close": "0", "mode": "live"})
        self.assertEqual(len(self.fixture.tasks()), 9)

    def test_newest_platform_file_wins_per_tab(self):
        os.remove(os.path.join(self.fixture.platform, "tabs", "2026-10-16", "work.json"))
        self.fixture.set_platform_tabs([tab("w1:t1", "/work/repos/alpha", "alpha", "alpha-dev")], generated="2026-10-16T07:00:00+09:00", name="old.json")
        self.fixture.set_platform_tabs([tab("w1:t1", "/work/repos/alpha", "alpha", "alpha-renamed")], generated="2026-10-16T09:00:00+09:00", name="new.json")
        self.fixture.set_tasks([])
        code, lines, stderr = self.fixture.run([HERDR_TAB_SYNC, "--dry-run"])
        self.assertEqual(code, 0, stderr)
        self.assertEqual([line[5] for line in by_kind(lines, "create") if line[1] == "w1:t1"], ["alpha-renamed (alpha)"])
        self.assertIn("(2 files, generated 2026-10-16T07:00:00+09:00..2026-10-16T09:00:00+09:00, age ", by_kind(lines, "source")[0][1])

    def test_fallback_to_herdr(self):
        herdr_tabs = [tab("w5:t1", "/work/repos/epsilon", "epsilon", "eps-dev"), tab("w5:t2", self.home, "epsilon", "eps-home")]
        self.fixture.set_herdr(herdr_tabs)
        self.fixture.set_tasks([])
        cases = [
            ("no platform files", lambda: self.fixture.set_platform_tabs([], name="work.json")),
            ("platform file without tab ids", lambda: self.fixture.set_platform_tabs([{"tab_label": "x", "cwd": "/work"}])),
            ("platform file without folders", lambda: self.fixture.set_platform_tabs([{"tab_id": "w1:t1", "tab_label": "x"}])),
        ]
        for name, arrange in cases:
            with self.subTest(name):
                arrange()
                code, lines, stderr = self.fixture.run([HERDR_TAB_SYNC, "--dry-run"])
                self.assertEqual(code, 0, stderr)
                self.assertTrue(by_kind(lines, "source")[0][1].startswith("herdr live listing ("), lines)
                self.assertEqual(by_kind(lines, "create"), [
                    ["create", "w5:t1", "epsilon", "work", "Work Eng", "eps-dev (epsilon)"],
                    ["create", "w5:t2", "misc", "", "Chief of Staff", "eps-home (epsilon)"],
                ])

    def test_errors_exit_nonzero(self):
        cases = [
            ("herdr fails during fallback", {"FAKE_HERDR_FAIL": "1"}, lambda: self.fixture.set_platform_tabs([])),
            ("task export fails", {"FAKE_TASK_FAIL": "1"}, lambda: None),
        ]
        for name, env, arrange in cases:
            with self.subTest(name):
                arrange()
                self.fixture.extra_env = env
                code, lines, stderr = self.fixture.run([HERDR_TAB_SYNC, "--dry-run"])
                self.assertEqual(code, 1)
                self.assertTrue(stderr.startswith("sync: "), stderr)
                self.assertEqual(by_kind(lines, "create"), [])


class AttachTests(unittest.TestCase):
    def setUp(self):
        self.fixture = Fixture()
        self.fixture.set_platform_tabs([tab("w1:t1", "/work/repos/alpha", "alpha", "alpha-dev"), tab("w5:t5", "/work/repos/zeta", "zeta", "zeta-dev")])
        self.fixture.set_tasks([
            {"uuid": "t1", "description": "alpha-dev (alpha)", "herdrtab": "w1:t1", "status": "pending"},
            {"uuid": "t2", "description": "done tab", "herdrtab": "w2:t2", "status": "completed"},
        ])
        self.pages = [
            page("p1", "w1:t1", title="Open question"),
            page("p2", "w1:t1", "2026-10-11 00:00:00", title="Answered"),
            page("p3", "w1:t1", "2026-10-11 00:00:00", title="Failed delivery"),
            page("p4", "w5:t5", title="No task known tab"),
            page("p5", "w6:t6", title="No task unknown tab"),
            page("p6", None, "2026-10-11 00:00:00", title="No tab"),
            page("p7", "w2:t2", title="Completed task"),
        ]
        self.fixture.set_pages(self.pages)
        self.fixture.set_journal('page store listener: no eligible agent for page "p3" tab "w1:t1"\n')

    def annotations(self, uuid):
        return [a["description"] for t in self.fixture.tasks() if t["uuid"] == uuid for a in t.get("annotations", [])]

    def test_dry_run_plans_without_writing(self):
        code, lines, stderr = self.fixture.run([PHONE_PAGE, "attach", "--dry-run"])
        self.assertEqual(code, 0, stderr)
        self.assertEqual(summary(lines), {"matched": "4", "unmatched": "2", "no-tab": "1", "mode": "dry-run"})
        self.assertEqual(self.annotations("t1"), [])
        self.assertEqual(rows(os.path.join(self.fixture.state, "unmatched.dryrun.tsv")), [
            ["headline", "desk", "page_id"],
            ["No task known tab", "Work Eng", "p4"],
            ["No task unknown tab", "Chief of Staff", "p5"],
        ])
        self.assertFalse(os.path.exists(os.path.join(self.fixture.state, "unmatched.tsv")))

    def test_live_run_annotates_once(self):
        expected_t1 = [
            "phone-page p1 https://store.example/p/p1",
            "phone-page p2 https://store.example/p/p2",
            "phone-page p2 answered 2026-10-11 00:00:00",
            "phone-page p3 https://store.example/p/p3",
        ]
        for attempt in ("first run", "second run"):
            with self.subTest(attempt):
                code, lines, stderr = self.fixture.run([PHONE_PAGE, "attach"])
                self.assertEqual(code, 0, stderr)
                self.assertEqual(summary(lines), {"matched": "4", "unmatched": "2", "no-tab": "1", "mode": "live"})
                self.assertEqual(self.annotations("t1"), expected_t1)
                self.assertEqual(self.annotations("t2"), ["phone-page p7 https://store.example/p/p7"])
        actions = {line[1]: line[3] for line in by_kind(lines, "matched")}
        self.assertEqual(actions, {"p1": "keep", "p2": "keep", "p3": "keep", "p7": "keep"})
        self.assertEqual(rows(os.path.join(self.fixture.state, "unmatched.tsv"))[1:], [["No task known tab", "Work Eng", "p4"], ["No task unknown tab", "Chief of Staff", "p5"]])

    def test_listener_log_failure_skips_answered_marking(self):
        self.fixture.extra_env = {"FAKE_JOURNAL_FAIL": "1"}
        code, lines, stderr = self.fixture.run([PHONE_PAGE, "attach"])
        self.assertEqual(code, 0, stderr)
        self.assertEqual(by_kind(lines, "listener-log")[0][:2], ["listener-log", "unknown"])
        self.assertNotIn("phone-page p2 answered 2026-10-11 00:00:00", self.annotations("t1"))
        self.assertIn("phone-page p2 https://store.example/p/p2", self.annotations("t1"))

    def test_store_failure_exits_nonzero(self):
        write(self.fixture.pages, "not json")
        code, lines, stderr = self.fixture.run([PHONE_PAGE, "attach", "--dry-run"])
        self.assertEqual(code, 1)
        self.assertTrue(stderr.startswith("attach: "), stderr)
        self.assertEqual(self.annotations("t1"), [])


class CloseTests(unittest.TestCase):
    def setUp(self):
        self.fixture = Fixture()
        self.fixture.set_platform_tabs([tab("w1:t1", "/work/repos/alpha", "alpha", "alpha-dev")])
        self.fixture.set_herdr([tab("w1:t1", "/work/repos/alpha", "alpha", "alpha-dev")])
        self.pages = [
            page("answered", "w1:t1", "2026-10-11 00:00:00"),
            page("undelivered", "w1:t1", "2026-10-11 00:00:00"),
            page("open", "w1:t1"),
            page("tabgone", "w9:t9"),
            page("notab", None, "2026-10-11 00:00:00"),
        ]
        self.fixture.set_pages(self.pages)
        self.fixture.set_journal('page store listener: could not prompt page "undelivered" tab "w1:t1"\n')
        self.log = os.path.join(self.fixture.state, "close-log.tsv")

    def expected(self, lines):
        return {line[1]: (line[2], line[0]) for line in lines if line[0] in ("delete", "none")}

    def test_statuses_and_actions(self):
        cases = [
            ("dry run", ["--dry-run"], {}, {"answered": ("answered", "delete"), "undelivered": ("undelivered", "none"), "open": ("open", "none"), "tabgone": ("tabgone", "delete"), "notab": ("notab", "none")}, [], "dry-run"),
            ("live", [], {}, {"answered": ("answered", "delete"), "undelivered": ("undelivered", "none"), "open": ("open", "none"), "tabgone": ("tabgone", "delete"), "notab": ("notab", "none")}, ["answered", "tabgone"], "live"),
            ("herdr lookup fails", [], {"FAKE_HERDR_FAIL": "1"}, {name: ("unknown", "none") for name in ("answered", "undelivered", "open", "tabgone", "notab")}, [], "live"),
            ("listener log fails", [], {"FAKE_JOURNAL_FAIL": "1"}, {name: ("unknown", "none") for name in ("answered", "undelivered", "open", "tabgone", "notab")}, [], "live"),
        ]
        for name, flags, env, expected, deletes, mode in cases:
            with self.subTest(name):
                if os.path.exists(self.fixture.deletes):
                    os.remove(self.fixture.deletes)
                if os.path.exists(self.log):
                    os.remove(self.log)
                self.fixture.extra_env = env
                code, lines, stderr = self.fixture.run([PHONE_PAGE, "close", *flags])
                self.assertEqual(code, 0, stderr)
                self.assertEqual(self.expected(lines), expected)
                self.assertEqual(self.fixture.deleted(), deletes)
                self.assertNotIn("open", self.fixture.deleted())
                logged = rows(self.log)
                self.assertEqual(logged[0], ["time", "page_id", "area", "status", "action", "mode"])
                self.assertEqual({row[1]: (row[3], row[4], row[5]) for row in logged[1:]}, {page_id: (status, action, mode) for page_id, (status, action) in expected.items()})
                self.assertEqual({row[1]: row[2] for row in logged[1:] if row[1] in ("answered", "open")}, {"answered": "work", "open": "work"})

    def test_log_appends_across_runs(self):
        for _ in range(2):
            code, lines, stderr = self.fixture.run([PHONE_PAGE, "close", "--dry-run"])
            self.assertEqual(code, 0, stderr)
        self.assertEqual(len(rows(self.log)), 11)
        self.assertEqual(self.fixture.deleted(), [])

    def test_live_delete_removes_inbox_line(self):
        write(self.fixture.inbox, "- 2026-10-10 10:00 Open: https://store.example/p/open\n- 2026-10-10 10:00 Answered: https://store.example/p/answered\n")
        code, lines, stderr = self.fixture.run([PHONE_PAGE, "close"])
        self.assertEqual(code, 0, stderr)
        with open(self.fixture.inbox, encoding="utf-8") as handle:
            self.assertEqual(handle.read(), "- 2026-10-10 10:00 Open: https://store.example/p/open\n")


if __name__ == "__main__":
    unittest.main()
